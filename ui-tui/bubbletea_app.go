package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

type turnDoneMsg struct {
	err   error
	quit  bool
}

type doctorDoneMsg struct {
	items []doctorItem
	ok    bool
}

type tuiModel struct {
	state      *appState
	inputValue string
	viewport   viewport.Model
	lines      []string
	width      int
	height     int
	processing bool
	turnStream chan tea.Msg
	mdRenderer *glamour.TermRenderer
	mdWidth    int
	mdBuffer   string
	mdLineIdx  int
	mdDirty    bool
	userStyle  lipgloss.Style
	metaStyle  lipgloss.Style
	runDoctor  bool
}

const streamRenderInterval = 80 * time.Millisecond
const userLinePrefix = "\x00user-input\x00"
const metaLinePrefix = "\x00meta-line\x00"

func newTUIModel(state *appState, noDoctor bool) tuiModel {
	_ = noDoctor
	vp := viewport.New(80, 20)
	vp.SetContent("欢迎使用 ui-tui（Bubble Tea 模式）\n")
	return tuiModel{
		state:     state,
		viewport:  vp,
		lines:     []string{"欢迎使用 ui-tui（Bubble Tea 模式）"},
		userStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#A8A8A8")).Padding(0, 1),
		metaStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")),
		runDoctor: false,
	}
}

func (m tuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.WindowSize()}
	if m.runDoctor {
		cmds = append(cmds, startDoctorCmd(m.state))
	}
	return tea.Batch(cmds...)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputH := 3
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - inputH - 4
		if m.viewport.Height < 5 {
			m.viewport.Height = 5
		}
		m.syncViewport()
		return m, nil
	case turnDoneMsg:
		m.flushMarkdownBuffer(true)
		m.processing = false
		m.turnStream = nil
		if msg.err != nil {
			m.appendLine(fmt.Sprintf("error: %v", msg.err))
			m.state.setErrStatus(msg.err)
		} else if msg.quit {
			return m, tea.Quit
		} else {
			m.state.setStatus(true, "ok", "turn completed")
		}
		return m, nil
	case doctorDoneMsg:
		if msg.ok {
			m.state.setStatus(true, "ok", "startup doctor passed")
		} else {
			m.state.setStatus(false, "doctor_failed", "startup doctor found failures")
		}
		return m, nil
	case turnLineMsg:
		if strings.TrimSpace(msg.line) != "" {
			if isIntermediateInfoLine(msg.line) {
				m.appendLine(metaLinePrefix + msg.line)
			} else
			if m.mdLineIdx < 0 || !isAssistantFinalLine(msg.line) {
				m.appendLine(msg.line)
			}
		}
		if m.processing && m.turnStream != nil {
			return m, waitTurnStreamCmd(m.turnStream)
		}
		return m, nil
	case turnEventMsg:
		if msg.event != nil {
			m.handleTurnEvent(msg.event)
			m.state.addEvent(msg.event)
		}
		if m.processing && m.turnStream != nil {
			return m, waitTurnStreamCmd(m.turnStream)
		}
		return m, nil
	case streamRenderTickMsg:
		if m.processing {
			m.flushMarkdownBuffer(false)
			return m, streamRenderTickCmd(streamRenderInterval)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "pgup":
			m.viewport.HalfViewUp()
			return m, nil
		case "pgdown":
			m.viewport.HalfViewDown()
			return m, nil
		case "up":
			if m.processing {
				return m, nil
			}
		case "down":
			if m.processing {
				return m, nil
			}
		case "enter":
			if m.processing {
				return m, nil
			}
			raw := strings.TrimSpace(m.inputValue)
			m.inputValue = ""
			if raw == "" {
				return m, nil
			}
			if raw == "/quit" || raw == "/exit" {
				return m, tea.Quit
			}
			if raw == "/clear" {
				m.lines = nil
				m.syncViewport()
			}
			m.appendLine("")
			m.appendLine(userLinePrefix + raw)
			m.resetMarkdownBuffer()
			m.processing = true
			m.turnStream = make(chan tea.Msg, 64)
			return m, tea.Batch(
				startTurnCmd(m.turnStream, m.state, raw),
				waitTurnStreamCmd(m.turnStream),
				streamRenderTickCmd(streamRenderInterval),
			)
		case "backspace", "ctrl+h":
			if !m.processing && m.inputValue != "" {
				rs := []rune(m.inputValue)
				m.inputValue = string(rs[:len(rs)-1])
			}
			return m, nil
		}
		if !m.processing && len(msg.Runes) > 0 {
			m.inputValue += string(msg.Runes)
			return m, nil
		}
	}
	return m, nil
}

func (m tuiModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("session=%s status=%s/%s transport=%s", m.state.session, m.state.lastStatus, m.state.lastCode, m.state.activeTransport),
	)
	footer := ""
	if m.processing {
		footer = "处理中...（Enter 提交，PgUp/PgDn 翻页，Ctrl+C 退出）"
	} else {
		footer = "Enter 提交，PgUp/PgDn 翻页，Ctrl+C 退出"
	}
	inputLine := "› " + m.inputValue
	if strings.TrimSpace(m.inputValue) == "" {
		inputLine = "› 输入消息或命令（/help, /quit）"
	}
	return title + "\n" + m.viewport.View() + "\n\n" + inputLine + "\n" + footer
}

type turnLineMsg struct {
	line string
}

type turnEventMsg struct {
	event map[string]any
}

type streamRenderTickMsg struct{}
func startTurnCmd(stream chan tea.Msg, state *appState, text string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer close(stream)
			_, err, quit := handleTUICommand(state, text, func(evt map[string]any) {
				stream <- turnEventMsg{event: evt}
			}, func(line string) {
				stream <- turnLineMsg{line: line}
			})
			stream <- turnDoneMsg{err: err, quit: quit}
		}()
		return nil
	}
}

func waitTurnStreamCmd(stream chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return turnDoneMsg{}
		}
		return msg
	}
}

func streamRenderTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return streamRenderTickMsg{}
	})
}

func startDoctorCmd(state *appState) tea.Cmd {
	return func() tea.Msg {
		items, ok := state.runDoctor()
		return doctorDoneMsg{items: items, ok: ok}
	}
}

func (m *tuiModel) appendLine(line string) {
	m.lines = append(m.lines, line)
	if len(m.lines) > m.state.chatMaxLines && m.state.chatMaxLines > 0 {
		m.lines = m.lines[len(m.lines)-m.state.chatMaxLines:]
	}
	m.syncViewport()
}

func (m *tuiModel) syncViewport() {
	content := m.renderViewportContent()
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *tuiModel) renderViewportContent() string {
	parts := make([]string, 0, len(m.lines))
	markdownChunk := make([]string, 0, len(m.lines))
	flushMarkdown := func() {
		if len(markdownChunk) == 0 {
			return
		}
		content := strings.Join(markdownChunk, "\n")
		rendered, err := m.renderMarkdown(content)
		if err == nil {
			parts = append(parts, strings.TrimSuffix(rendered, "\n"))
		} else {
			parts = append(parts, content)
		}
		markdownChunk = markdownChunk[:0]
	}
	for _, line := range m.lines {
		if strings.HasPrefix(line, userLinePrefix) {
			flushMarkdown()
			raw := strings.TrimPrefix(line, userLinePrefix)
			parts = append(parts, m.renderUserLine(raw))
			continue
		}
		if strings.HasPrefix(line, metaLinePrefix) {
			flushMarkdown()
			raw := strings.TrimPrefix(line, metaLinePrefix)
			parts = append(parts, m.renderMetaLine(raw))
			continue
		}
		markdownChunk = append(markdownChunk, line)
	}
	flushMarkdown()
	return strings.Join(parts, "\n")
}

func (m *tuiModel) renderUserLine(raw string) string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	if width < 20 {
		width = 20
	}
	lineWidth := lipgloss.Width(raw)
	if lineWidth < width {
		raw = raw + strings.Repeat(" ", width-lineWidth)
	}
	return m.userStyle.Render(raw)
}

func (m *tuiModel) renderMetaLine(raw string) string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	if width < 20 {
		width = 20
	}
	wrapped := wordwrap.String(raw, width)
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		lines[i] = m.metaStyle.Render(line)
	}
	return strings.Join(lines, "\n")
}

func (m *tuiModel) resetMarkdownBuffer() {
	m.mdBuffer = ""
	m.mdLineIdx = -1
	m.mdDirty = false
}

func (m *tuiModel) ensureMarkdownLine() {
	if m.mdLineIdx >= 0 {
		return
	}
	m.lines = append(m.lines, "")
	m.mdLineIdx = len(m.lines) - 1
}

func (m *tuiModel) flushMarkdownBuffer(force bool) {
	if m.mdLineIdx < 0 {
		return
	}
	if !m.mdDirty && !force {
		return
	}
	if m.mdLineIdx >= len(m.lines) {
		m.mdLineIdx = -1
		return
	}
	m.lines[m.mdLineIdx] = m.mdBuffer
	m.mdDirty = false
	m.syncViewport()
}

func (m *tuiModel) appendMarkdownDelta(text string) {
	if text == "" {
		return
	}
	m.ensureMarkdownLine()
	m.mdBuffer += text
	m.mdDirty = true
}

func (m *tuiModel) setMarkdownFinal(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	m.ensureMarkdownLine()
	m.mdBuffer = text
	m.mdDirty = true
	m.flushMarkdownBuffer(true)
}

func (m *tuiModel) handleTurnEvent(evt map[string]any) {
	if evt == nil {
		return
	}
	evtType, _ := evt["type"].(string)
	if evtType == "" {
		evtType, _ = evt["Type"].(string)
	}
	if evtType == "model_stream_event" {
		m.appendMarkdownDelta(extractModelStreamDelta(evt))
		return
	}
	if evtType == "result" {
		if text, _ := evt["final_response"].(string); text != "" {
			m.setMarkdownFinal(text)
		}
	}
}

func extractModelStreamDelta(evt map[string]any) string {
	data, _ := evt["data"].(map[string]any)
	if data == nil {
		return ""
	}
	eventData, _ := data["event_data"].(map[string]any)
	if eventData == nil {
		return ""
	}
	text, _ := eventData["text"].(string)
	return text
}

func isAssistantFinalLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "assistant: ") || strings.HasPrefix(trimmed, "result: ")
}

func isIntermediateInfoLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "tool_started: ") ||
		strings.HasPrefix(trimmed, "tool_finished: ")
}

func (m *tuiModel) renderMarkdown(content string) (string, error) {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	if width < 20 {
		width = 20
	}
	if m.mdRenderer == nil || m.mdWidth != width {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return "", err
		}
		m.mdRenderer = renderer
		m.mdWidth = width
	}
	return m.mdRenderer.Render(content)
}

func runBubbleTeaUI(s *appState, noDoctor bool) error {
	termenv.SetDefaultOutput(termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor), termenv.WithTTY(false)))
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stdout, termenv.WithProfile(termenv.TrueColor), termenv.WithTTY(false)))
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
	model := newTUIModel(s, noDoctor)
	program := tea.NewProgram(model)
	_, err := program.Run()
	return err
}
