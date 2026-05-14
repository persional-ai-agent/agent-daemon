package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

type terminalRuntime struct {
	eventBus  *runtimeEventBus
	parser    *runtimeEventParser
	stateTree *uiStateTree
	renderer  *uiRenderEngine
}

func newTerminalRuntime(width int) *terminalRuntime {
	return &terminalRuntime{
		eventBus:  newRuntimeEventBus(),
		parser:    newRuntimeEventParser(),
		stateTree: newUIStateTree(),
		renderer:  newUIRenderEngine(width),
	}
}

func (r *terminalRuntime) resetContent(welcome string) {
	r.stateTree = newUIStateTree()
	r.parser = newRuntimeEventParser()
	r.renderer.reset()
	if strings.TrimSpace(welcome) != "" {
		r.stateTree.addSystemNode(welcome)
	}
}

func (r *terminalRuntime) addSystemText(text string) {
	r.stateTree.addSystemNode(text)
}

func (r *terminalRuntime) addError(text string) {
	r.stateTree.addErrorNode(text)
}

func (r *terminalRuntime) setWidth(width int) {
	r.renderer.setWidth(width)
}

func (r *terminalRuntime) toggleThinkingExpanded() bool {
	return r.stateTree.toggleThinkingExpanded()
}

func (r *terminalRuntime) startTurn(userText string) {
	r.publish(runtimeEvent{Type: runtimeEventUserInput, Text: userText})
	r.consumePendingEvents()
}

func (r *terminalRuntime) endTurn() {
	r.publish(runtimeEvent{Type: runtimeEventBlockFlush})
	r.consumePendingEvents()
}

func (r *terminalRuntime) publishTurnEvent(evt map[string]any) {
	events := mapTurnEventToRuntimeEvents(evt)
	for _, event := range events {
		r.publish(event)
	}
}

func (r *terminalRuntime) publishLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(trimmed, "assistant: ") || strings.HasPrefix(trimmed, "result: ") {
		return
	}
	if strings.HasPrefix(trimmed, "tool_started: ") || strings.HasPrefix(trimmed, "tool_finished: ") {
		return
	}
	r.publish(runtimeEvent{Type: runtimeEventSystemText, Text: line})
}

func (r *terminalRuntime) publish(event runtimeEvent) {
	r.eventBus.Publish(event)
}

func (r *terminalRuntime) consumePendingEvents() {
	for _, event := range r.eventBus.Drain() {
		r.parser.Apply(r.stateTree, event)
	}
}

func (r *terminalRuntime) render(force bool) (string, bool) {
	return r.renderer.Render(r.stateTree, force)
}

type runtimeEventType int

const (
	runtimeEventTokenDelta runtimeEventType = iota
	runtimeEventAssistantFinal
	runtimeEventToolStart
	runtimeEventToolEnd
	runtimeEventThinkingStart
	runtimeEventThinkingDelta
	runtimeEventThinkingEnd
	runtimeEventSystemText
	runtimeEventError
	runtimeEventUserInput
	runtimeEventBlockFlush
)

type runtimeEvent struct {
	Type       runtimeEventType
	Text       string
	ToolName   string
	ToolCallID string
	Status     string
}

type runtimeEventBus struct {
	queue []runtimeEvent
}

func newRuntimeEventBus() *runtimeEventBus {
	return &runtimeEventBus{}
}

func (b *runtimeEventBus) Publish(event runtimeEvent) {
	b.queue = append(b.queue, event)
}

func (b *runtimeEventBus) Drain() []runtimeEvent {
	if len(b.queue) == 0 {
		return nil
	}
	events := make([]runtimeEvent, len(b.queue))
	copy(events, b.queue)
	b.queue = b.queue[:0]
	return events
}

type parserState int

const (
	parserStateNormal parserState = iota
	parserStateThinking
)

type runtimeEventParser struct {
	state       parserState
	pendingText string
	toolStarts  map[string]time.Time
}

func newRuntimeEventParser() *runtimeEventParser {
	return &runtimeEventParser{state: parserStateNormal, toolStarts: make(map[string]time.Time)}
}

func (p *runtimeEventParser) Apply(tree *uiStateTree, event runtimeEvent) {
	switch event.Type {
	case runtimeEventUserInput:
		p.flushPendingText(tree)
		tree.closeStreamingAssistantNode()
		tree.endThinkingBlock()
		tree.addBlankNode()
		tree.addUserNode(event.Text)
	case runtimeEventTokenDelta:
		p.consumeAssistantDelta(tree, event.Text)
	case runtimeEventAssistantFinal:
		p.flushPendingText(tree)
		tree.finalizeAssistantNode(event.Text)
		tree.endThinkingBlock()
		p.state = parserStateNormal
	case runtimeEventToolStart:
		p.onToolStart(tree, event)
	case runtimeEventToolEnd:
		p.onToolEnd(tree, event)
	case runtimeEventThinkingStart:
		tree.startThinkingBlock()
	case runtimeEventThinkingDelta:
		tree.appendThinkingToken(event.Text)
	case runtimeEventThinkingEnd:
		tree.endThinkingBlock()
	case runtimeEventSystemText:
		tree.addSystemNode(event.Text)
	case runtimeEventError:
		tree.addErrorNode(event.Text)
	case runtimeEventBlockFlush:
		p.flushPendingText(tree)
		tree.closeStreamingAssistantNode()
		tree.endThinkingBlock()
		p.state = parserStateNormal
	}
}

func (p *runtimeEventParser) onToolStart(tree *uiStateTree, event runtimeEvent) {
	callID := strings.TrimSpace(event.ToolCallID)
	if callID != "" {
		p.toolStarts[callID] = time.Now()
	}
	name := normalizeToolName(event)
	tree.upsertToolNode(callID, name, "running", 0)
}

func (p *runtimeEventParser) onToolEnd(tree *uiStateTree, event runtimeEvent) {
	callID := strings.TrimSpace(event.ToolCallID)
	duration := time.Duration(0)
	if callID != "" {
		if startAt, ok := p.toolStarts[callID]; ok {
			duration = time.Since(startAt)
			delete(p.toolStarts, callID)
		}
	}
	name := normalizeToolName(event)
	status := strings.TrimSpace(event.Status)
	if status == "" {
		status = "completed"
	}
	tree.upsertToolNode(callID, name, status, duration)
}

func (p *runtimeEventParser) consumeAssistantDelta(tree *uiStateTree, delta string) {
	if delta == "" {
		return
	}
	p.pendingText += delta
	const openTag = "<thinking>"
	const closeTag = "</thinking>"

	for {
		switch p.state {
		case parserStateNormal:
			idx := strings.Index(p.pendingText, openTag)
			if idx < 0 {
				flushLen := len(p.pendingText) - suffixPrefixOverlap(p.pendingText, openTag)
				if flushLen > 0 {
					tree.appendAssistantToken(p.pendingText[:flushLen])
					p.pendingText = p.pendingText[flushLen:]
				}
				return
			}
			if idx > 0 {
				tree.appendAssistantToken(p.pendingText[:idx])
			}
			p.pendingText = p.pendingText[idx+len(openTag):]
			tree.startThinkingBlock()
			p.state = parserStateThinking
		case parserStateThinking:
			idx := strings.Index(p.pendingText, closeTag)
			if idx < 0 {
				flushLen := len(p.pendingText) - suffixPrefixOverlap(p.pendingText, closeTag)
				if flushLen > 0 {
					tree.appendThinkingToken(p.pendingText[:flushLen])
					p.pendingText = p.pendingText[flushLen:]
				}
				return
			}
			if idx > 0 {
				tree.appendThinkingToken(p.pendingText[:idx])
			}
			p.pendingText = p.pendingText[idx+len(closeTag):]
			tree.endThinkingBlock()
			p.state = parserStateNormal
		}
	}
}

func (p *runtimeEventParser) flushPendingText(tree *uiStateTree) {
	if p.pendingText == "" {
		return
	}
	if p.state == parserStateThinking {
		tree.appendThinkingToken(p.pendingText)
	} else {
		tree.appendAssistantToken(p.pendingText)
	}
	p.pendingText = ""
}

func suffixPrefixOverlap(s, prefix string) int {
	max := len(prefix) - 1
	if max <= 0 {
		return 0
	}
	if len(s) < max {
		max = len(s)
	}
	for length := max; length > 0; length-- {
		if strings.HasSuffix(s, prefix[:length]) {
			return length
		}
	}
	return 0
}

func normalizeToolName(event runtimeEvent) string {
	if strings.TrimSpace(event.ToolName) != "" {
		return strings.TrimSpace(event.ToolName)
	}
	if strings.TrimSpace(event.Text) != "" {
		return strings.TrimSpace(event.Text)
	}
	return "unknown"
}

type uiNodeType int

const (
	nodeUser uiNodeType = iota
	nodeTool
	nodeThinking
	nodeSystem
	nodeError
	nodeAssistant
)

type uiNode struct {
	ID         string
	Type       uiNodeType
	Content    string
	Status     string
	Dirty      bool
	Expanded   bool
	ToolCallID string
}

type uiStateTree struct {
	nodes              []*uiNode
	nextID             int
	streamingAssistant string
	thinkingNodeID     string
	thinkingExpanded   bool
	toolNodeByCallID   map[string]string
}

func newUIStateTree() *uiStateTree {
	return &uiStateTree{toolNodeByCallID: make(map[string]string)}
}

func (tree *uiStateTree) addNode(nodeType uiNodeType, content string, status string) *uiNode {
	node := &uiNode{
		ID:      fmt.Sprintf("node-%d", tree.nextID),
		Type:    nodeType,
		Content: content,
		Status:  status,
		Dirty:   true,
	}
	tree.nextID++
	tree.nodes = append(tree.nodes, node)
	return node
}

func (tree *uiStateTree) addBlankNode() {
	tree.addNode(nodeSystem, "", "done")
}

func (tree *uiStateTree) addUserNode(content string) {
	tree.addNode(nodeUser, content, "done")
}

func (tree *uiStateTree) addSystemNode(content string) {
	tree.addNode(nodeSystem, content, "done")
}

func (tree *uiStateTree) addErrorNode(content string) {
	tree.addNode(nodeError, content, "done")
}

func (tree *uiStateTree) appendAssistantToken(token string) {
	if token == "" {
		return
	}
	node := tree.ensureAssistantStreamingNode()
	node.Content += token
	node.Status = "streaming"
	node.Dirty = true
}

func (tree *uiStateTree) finalizeAssistantNode(content string) {
	node := tree.findStreamingAssistantNode()
	if node == nil {
		if strings.TrimSpace(content) == "" {
			return
		}
		node = tree.addNode(nodeAssistant, "", "streaming")
	}
	if strings.TrimSpace(content) != "" {
		node.Content = content
	}
	node.Status = "done"
	node.Dirty = true
	tree.streamingAssistant = ""
}

func (tree *uiStateTree) closeStreamingAssistantNode() {
	node := tree.findStreamingAssistantNode()
	if node == nil {
		return
	}
	node.Status = "done"
	node.Dirty = true
	tree.streamingAssistant = ""
}

func (tree *uiStateTree) ensureAssistantStreamingNode() *uiNode {
	if node := tree.findStreamingAssistantNode(); node != nil {
		return node
	}
	node := tree.addNode(nodeAssistant, "", "streaming")
	tree.streamingAssistant = node.ID
	return node
}

func (tree *uiStateTree) findStreamingAssistantNode() *uiNode {
	if tree.streamingAssistant == "" {
		return nil
	}
	for _, node := range tree.nodes {
		if node.ID == tree.streamingAssistant {
			return node
		}
	}
	return nil
}

func (tree *uiStateTree) startThinkingBlock() {
	if node := tree.findThinkingNode(); node != nil {
		node.Status = "streaming"
		node.Dirty = true
		return
	}
	node := tree.addNode(nodeThinking, "", "streaming")
	node.Expanded = tree.thinkingExpanded
	tree.thinkingNodeID = node.ID
}

func (tree *uiStateTree) appendThinkingToken(token string) {
	if token == "" {
		return
	}
	node := tree.findThinkingNode()
	if node == nil {
		tree.startThinkingBlock()
		node = tree.findThinkingNode()
	}
	if node == nil {
		return
	}
	node.Content += token
	node.Dirty = true
}

func (tree *uiStateTree) endThinkingBlock() {
	node := tree.findThinkingNode()
	if node == nil {
		return
	}
	node.Status = "done"
	node.Dirty = true
	tree.thinkingNodeID = ""
}

func (tree *uiStateTree) findThinkingNode() *uiNode {
	if tree.thinkingNodeID == "" {
		return nil
	}
	for _, node := range tree.nodes {
		if node.ID == tree.thinkingNodeID {
			return node
		}
	}
	return nil
}

func (tree *uiStateTree) toggleThinkingExpanded() bool {
	tree.thinkingExpanded = !tree.thinkingExpanded
	for _, node := range tree.nodes {
		if node != nil && node.Type == nodeThinking {
			node.Expanded = tree.thinkingExpanded
			node.Dirty = true
		}
	}
	return tree.thinkingExpanded
}

func (tree *uiStateTree) upsertToolNode(callID, name, status string, duration time.Duration) {
	callID = strings.TrimSpace(callID)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unknown"
	}
	if callID == "" {
		tree.addNode(nodeTool, formatToolLine(name, status, duration), status)
		return
	}
	if nodeID, ok := tree.toolNodeByCallID[callID]; ok {
		node := tree.findNodeByID(nodeID)
		if node != nil {
			node.Content = formatToolLine(name, status, duration)
			node.Status = status
			node.Dirty = true
			if node.ToolCallID == "" {
				node.ToolCallID = callID
			}
			return
		}
	}
	node := tree.addNode(nodeTool, formatToolLine(name, status, duration), status)
	node.ToolCallID = callID
	tree.toolNodeByCallID[callID] = node.ID
}

func (tree *uiStateTree) findNodeByID(id string) *uiNode {
	if id == "" {
		return nil
	}
	for _, node := range tree.nodes {
		if node != nil && node.ID == id {
			return node
		}
	}
	return nil
}

func formatToolLine(name, status string, duration time.Duration) string {
	status = strings.ToLower(strings.TrimSpace(status))
	suffix := ""
	if duration > 0 {
		suffix = " (" + formatDuration(duration) + ")"
	}
	switch status {
	case "running", "started", "pending":
		return "⚙ Running: " + name
	case "failed", "error":
		return "✖ Failed: " + name + suffix
	default:
		return "✔ Done: " + name + suffix
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Truncate(time.Millisecond).String()
}

type uiRenderEngine struct {
	width      int
	markdown   *glamour.TermRenderer
	mdWidth    int
	cache      map[string]string
	lastOutput string
	userStyle  lipgloss.Style
	metaStyle  lipgloss.Style
	errorStyle lipgloss.Style
}

func newUIRenderEngine(width int) *uiRenderEngine {
	return &uiRenderEngine{
		width:      width,
		cache:      make(map[string]string),
		userStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#A8A8A8")).Padding(0, 1),
		metaStyle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")),
		errorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Bold(true),
	}
}

func (engine *uiRenderEngine) setWidth(width int) {
	if width <= 0 {
		return
	}
	if width == engine.width {
		return
	}
	engine.width = width
	engine.mdWidth = 0
	for key := range engine.cache {
		delete(engine.cache, key)
	}
}

func (engine *uiRenderEngine) reset() {
	engine.cache = make(map[string]string)
	engine.lastOutput = ""
	engine.markdown = nil
	engine.mdWidth = 0
}

func (engine *uiRenderEngine) Render(tree *uiStateTree, force bool) (string, bool) {
	parts := make([]string, 0, len(tree.nodes))
	contentChanged := false
	for _, node := range tree.nodes {
		if node == nil {
			continue
		}
		rendered, ok := engine.cache[node.ID]
		if !ok || node.Dirty || force {
			rendered = engine.renderNode(node)
			engine.cache[node.ID] = rendered
			node.Dirty = false
			contentChanged = true
		}
		parts = append(parts, rendered)
	}
	output := strings.Join(parts, "\n")
	if !contentChanged && output == engine.lastOutput {
		return "", false
	}
	engine.lastOutput = output
	return output, true
}

func (engine *uiRenderEngine) renderNode(node *uiNode) string {
	if node == nil {
		return ""
	}
	switch node.Type {
	case nodeUser:
		return engine.renderUserLine(node.Content)
	case nodeTool:
		return engine.renderMetaLine(node.Content)
	case nodeThinking:
		return engine.renderThinkingNode(node)
	case nodeError:
		return engine.errorStyle.Render(node.Content)
	case nodeAssistant:
		if node.Status == "done" {
			rendered, err := engine.renderMarkdown(node.Content)
			if err == nil {
				return strings.TrimSuffix(rendered, "\n")
			}
		}
		return node.Content
	case nodeSystem:
		fallthrough
	default:
		return node.Content
	}
}

func (engine *uiRenderEngine) renderThinkingNode(node *uiNode) string {
	if node == nil {
		return ""
	}
	prefix := "▶ Thinking"
	if node.Expanded {
		prefix = "▼ Thinking"
	}
	meta := fmt.Sprintf("%s (%s)", prefix, formatThinkingSize(node.Content))
	if !node.Expanded {
		return engine.metaStyle.Render(meta)
	}
	content := strings.TrimSpace(node.Content)
	if content == "" {
		return engine.metaStyle.Render(meta)
	}
	return engine.metaStyle.Render(meta) + "\n" + content
}

func formatThinkingSize(content string) string {
	if strings.TrimSpace(content) == "" {
		return "empty"
	}
	runeCount := len([]rune(content))
	return fmt.Sprintf("%d chars", runeCount)
}

func (engine *uiRenderEngine) renderUserLine(raw string) string {
	width := engine.effectiveWidth()
	lineWidth := lipgloss.Width(raw)
	if lineWidth < width {
		raw = raw + strings.Repeat(" ", width-lineWidth)
	}
	return engine.userStyle.Render(raw)
}

func (engine *uiRenderEngine) renderMetaLine(raw string) string {
	width := engine.effectiveWidth()
	wrapped := wordwrap.String(raw, width)
	lines := strings.Split(wrapped, "\n")
	for idx := range lines {
		lines[idx] = engine.metaStyle.Render(lines[idx])
	}
	return strings.Join(lines, "\n")
}

func (engine *uiRenderEngine) renderMarkdown(content string) (string, error) {
	width := engine.effectiveWidth()
	if engine.markdown == nil || engine.mdWidth != width {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return "", err
		}
		engine.markdown = renderer
		engine.mdWidth = width
	}
	return engine.markdown.Render(content)
}

func (engine *uiRenderEngine) effectiveWidth() int {
	if engine.width <= 0 {
		return 80
	}
	if engine.width < 20 {
		return 20
	}
	return engine.width
}

func mapTurnEventToRuntimeEvents(evt map[string]any) []runtimeEvent {
	evtType := extractEventType(evt)
	switch evtType {
	case "model_stream_event":
		return mapModelStreamEventToRuntimeEvents(evt)
	case "assistant_message":
		if text, _ := evt["content"].(string); text != "" {
			return []runtimeEvent{{Type: runtimeEventTokenDelta, Text: text}}
		}
		return nil
	case "result":
		if text, _ := evt["final_response"].(string); strings.TrimSpace(text) != "" {
			return []runtimeEvent{{Type: runtimeEventAssistantFinal, Text: text}}
		}
		return nil
	case "completed":
		if text, _ := evt["content"].(string); strings.TrimSpace(text) != "" {
			return []runtimeEvent{{Type: runtimeEventAssistantFinal, Text: text}}
		}
		return nil
	case "tool_started":
		data, _ := evt["data"].(map[string]any)
		return []runtimeEvent{{
			Type:       runtimeEventToolStart,
			ToolName:   extractToolName(evt),
			ToolCallID: asStringFromMap(data, "tool_call_id"),
			Status:     asStringFromMap(data, "status"),
		}}
	case "tool_finished":
		data, _ := evt["data"].(map[string]any)
		status := asStringFromMap(data, "status")
		if status == "" {
			status = "completed"
		}
		return []runtimeEvent{{
			Type:       runtimeEventToolEnd,
			ToolName:   extractToolName(evt),
			ToolCallID: asStringFromMap(data, "tool_call_id"),
			Status:     status,
		}}
	case "error":
		if text, _ := evt["error"].(string); text != "" {
			return []runtimeEvent{{Type: runtimeEventError, Text: text}}
		}
		return nil
	default:
		return nil
	}
}

func mapModelStreamEventToRuntimeEvents(evt map[string]any) []runtimeEvent {
	data, _ := evt["data"].(map[string]any)
	if data == nil {
		return nil
	}
	eventType := strings.ToLower(strings.TrimSpace(asStringFromMap(data, "event_type")))
	eventData, _ := data["event_data"].(map[string]any)

	switch eventType {
	case "text_delta":
		text := asStringFromMap(eventData, "text")
		if text == "" {
			text = asStringFromMap(eventData, "delta")
		}
		if text == "" {
			return nil
		}
		return []runtimeEvent{{Type: runtimeEventTokenDelta, Text: text}}
	case "tool_call_start", "tool_args_start":
		return []runtimeEvent{{
			Type:       runtimeEventToolStart,
			ToolName:   asStringFromMap(eventData, "tool_name"),
			ToolCallID: asStringFromMap(eventData, "tool_call_id"),
			Status:     "running",
		}}
	case "tool_call_done":
		return []runtimeEvent{{
			Type:       runtimeEventToolEnd,
			ToolName:   asStringFromMap(eventData, "tool_name"),
			ToolCallID: asStringFromMap(eventData, "tool_call_id"),
			Status:     "completed",
		}}
	}

	if strings.Contains(eventType, "reasoning") {
		if strings.Contains(eventType, "delta") {
			text := asStringFromMap(eventData, "text")
			if text == "" {
				text = asStringFromMap(eventData, "delta")
			}
			if text == "" {
				return nil
			}
			return []runtimeEvent{
				{Type: runtimeEventThinkingStart},
				{Type: runtimeEventThinkingDelta, Text: text},
			}
		}
		if strings.Contains(eventType, "done") {
			text := asStringFromMap(eventData, "text")
			events := make([]runtimeEvent, 0, 3)
			events = append(events, runtimeEvent{Type: runtimeEventThinkingStart})
			if text != "" {
				events = append(events, runtimeEvent{Type: runtimeEventThinkingDelta, Text: text})
			}
			events = append(events, runtimeEvent{Type: runtimeEventThinkingEnd})
			return events
		}
	}

	return nil
}

func extractEventType(evt map[string]any) string {
	if evt == nil {
		return ""
	}
	if eventType, _ := evt["type"].(string); eventType != "" {
		return eventType
	}
	eventType, _ := evt["Type"].(string)
	return eventType
}

func extractToolName(evt map[string]any) string {
	toolName, _ := evt["tool"].(string)
	if toolName != "" {
		return toolName
	}
	toolName, _ = evt["tool_name"].(string)
	if toolName != "" {
		return toolName
	}
	toolName, _ = evt["ToolName"].(string)
	if toolName != "" {
		return toolName
	}
	if data, _ := evt["data"].(map[string]any); data != nil {
		if name := asStringFromMap(data, "tool_name"); name != "" {
			return name
		}
	}
	return "unknown"
}

func asStringFromMap(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if value, _ := data[key].(string); strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return ""
}
