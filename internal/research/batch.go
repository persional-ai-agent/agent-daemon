package research

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type Task struct {
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Input     string         `json:"input"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Trajectory struct {
	ID         string            `json:"id"`
	SessionID  string            `json:"session_id"`
	Input      string            `json:"input"`
	StartedAt  string            `json:"started_at"`
	FinishedAt string            `json:"finished_at"`
	DurationMS int64             `json:"duration_ms"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
	Result     *core.RunResult   `json:"result,omitempty"`
	Events     []core.AgentEvent `json:"events,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
	Reward     float64           `json:"reward,omitempty"`
	Outcome    map[string]any    `json:"outcome,omitempty"`
}

type BatchReport struct {
	Total      int    `json:"total"`
	Succeeded  int    `json:"succeeded"`
	Failed     int    `json:"failed"`
	OutputPath string `json:"output_path"`
}

type RunOptions struct {
	Concurrency int
	StopOnError bool
	TimeoutSec  int
}

type FilterOptions struct {
	Success *bool
	Tool    string
	Model   string
	Limit   int
}

func LoadTasks(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	out := make([]Task, 0, 64)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var t Task
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(t.Input) == "" {
			return nil, fmt.Errorf("line %d: input required", lineNo)
		}
		if strings.TrimSpace(t.ID) == "" {
			t.ID = uuid.NewString()
		}
		if strings.TrimSpace(t.SessionID) == "" {
			t.SessionID = "research-" + t.ID
		}
		out = append(out, t)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func RunBatch(ctx context.Context, eng *agent.Engine, tasks []Task, outPath string) (BatchReport, error) {
	return RunBatchWithOptions(ctx, eng, tasks, outPath, RunOptions{})
}

func RunBatchWithOptions(ctx context.Context, eng *agent.Engine, tasks []Task, outPath string, opts RunOptions) (BatchReport, error) {
	if eng == nil {
		return BatchReport{}, fmt.Errorf("engine is nil")
	}
	if len(tasks) == 0 {
		return BatchReport{}, fmt.Errorf("no tasks")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return BatchReport{}, err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return BatchReport{}, err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	report := BatchReport{Total: len(tasks), OutputPath: outPath}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}
	if concurrency == 1 {
		for _, task := range tasks {
			item := runTask(ctx, eng, task, opts.TimeoutSec)
			if item.Success {
				report.Succeeded++
			} else {
				report.Failed++
			}
			if err := writeTrajectoryLine(w, item); err != nil {
				return report, err
			}
			if opts.StopOnError && !item.Success {
				break
			}
		}
		return report, nil
	}
	type indexedTask struct {
		index int
		task  Task
	}
	type indexedResult struct {
		index int
		item  Trajectory
	}
	taskCh := make(chan indexedTask)
	resCh := make(chan indexedResult, len(tasks))
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range taskCh {
				item := runTask(ctx, eng, it.task, opts.TimeoutSec)
				resCh <- indexedResult{index: it.index, item: item}
			}
		}()
	}
	go func() {
		for i, task := range tasks {
			taskCh <- indexedTask{index: i, task: task}
		}
		close(taskCh)
		wg.Wait()
		close(resCh)
	}()
	all := make([]indexedResult, 0, len(tasks))
	for r := range resCh {
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].index < all[j].index })
	for _, r := range all {
		item := r.item
		if item.Success {
			report.Succeeded++
		} else {
			report.Failed++
		}
		if err := writeTrajectoryLine(w, item); err != nil {
			return report, err
		}
		if opts.StopOnError && !item.Success {
			break
		}
	}
	return report, nil
}

func runTask(ctx context.Context, eng *agent.Engine, task Task, timeoutSec int) Trajectory {
	start := time.Now()
	events := make([]core.AgentEvent, 0, 32)
	engCopy := *eng
	engCopy.EventSink = func(evt core.AgentEvent) {
		events = append(events, evt)
	}
	runCtx := ctx
	cancel := func() {}
	if timeoutSec > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	}
	res, runErr := engCopy.Run(runCtx, task.SessionID, task.Input, agent.DefaultSystemPrompt(), nil)
	cancel()
	end := time.Now()
	item := Trajectory{
		ID:         task.ID,
		SessionID:  task.SessionID,
		Input:      task.Input,
		StartedAt:  start.Format(time.RFC3339Nano),
		FinishedAt: end.Format(time.RFC3339Nano),
		DurationMS: end.Sub(start).Milliseconds(),
		Success:    runErr == nil,
		Events:     events,
		Metadata:   task.Metadata,
	}
	if runErr != nil {
		item.Error = runErr.Error()
	} else {
		item.Result = res
		item.Outcome = buildOutcome(res)
		item.Reward = deriveReward(item)
	}
	return item
}

func buildOutcome(res *core.RunResult) map[string]any {
	if res == nil {
		return nil
	}
	return map[string]any{
		"finished_naturally": res.FinishedNaturally,
		"turns_used":         res.TurnsUsed,
		"message_count":      len(res.Messages),
	}
}

func deriveReward(t Trajectory) float64 {
	if !t.Success {
		return 0
	}
	reward := 1.0
	if t.Result != nil && t.Result.FinishedNaturally {
		reward += 0.25
	}
	if t.DurationMS > 0 && t.DurationMS <= 2000 {
		reward += 0.1
	}
	return reward
}

func writeTrajectoryLine(w *bufio.Writer, item Trajectory) error {
	bs, _ := json.Marshal(item)
	_, err := w.Write(append(bs, '\n'))
	return err
}

func CompressTrajectories(inPath, outPath string, maxChars int) (map[string]any, error) {
	if maxChars <= 0 {
		maxChars = 4000
	}
	inFile, err := os.Open(inPath)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, err
	}
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	gz := gzip.NewWriter(outFile)
	defer gz.Close()
	w := bufio.NewWriter(gz)
	defer w.Flush()

	sc := bufio.NewScanner(inFile)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	total := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		total++
		var t Trajectory
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			continue
		}
		compact := map[string]any{
			"id":          t.ID,
			"session_id":  t.SessionID,
			"input":       truncate(t.Input, maxChars),
			"started_at":  t.StartedAt,
			"finished_at": t.FinishedAt,
			"duration_ms": t.DurationMS,
			"success":     t.Success,
			"error":       truncate(t.Error, maxChars),
			"metadata":    t.Metadata,
			"events":      len(t.Events),
		}
		if t.Result != nil {
			compact["final_response"] = truncate(t.Result.FinalResponse, maxChars)
			compact["turns_used"] = t.Result.TurnsUsed
		}
		bs, _ := json.Marshal(compact)
		if _, err := w.Write(append(bs, '\n')); err != nil {
			return nil, err
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"success":   true,
		"input":     inPath,
		"output":    outPath,
		"processed": total,
	}, nil
}

func StatsTrajectories(path string) (map[string]any, error) {
	return StatsTrajectoriesWithFilter(path, FilterOptions{})
}

func StatsTrajectoriesWithFilter(path string, filter FilterOptions) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	total := 0
	success := 0
	var totalDuration int64
	eventBuckets := map[string]int{}
	toolHits := map[string]int{}
	modelHits := map[string]int{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		total++
		var t Trajectory
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			continue
		}
		if !matchTrajectoryFilter(t, filter) {
			continue
		}
		if t.Success {
			success++
		}
		for _, evt := range t.Events {
			eventBuckets[evt.Type]++
			if strings.EqualFold(evt.Type, "tool_started") && strings.TrimSpace(evt.ToolName) != "" {
				toolHits[evt.ToolName]++
			}
			if strings.EqualFold(evt.Type, "model_stream_event") && evt.Data != nil {
				model := strings.TrimSpace(fmt.Sprintf("%v", evt.Data["model"]))
				if model != "" {
					modelHits[model]++
				}
			}
		}
		totalDuration += t.DurationMS
		if filter.Limit > 0 && total >= filter.Limit {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	avg := int64(0)
	if total > 0 {
		avg = totalDuration / int64(total)
	}
	return map[string]any{
		"success":             true,
		"total":               total,
		"succeeded":           success,
		"failed":              total - success,
		"average_duration_ms": avg,
		"events":              eventBuckets,
		"tools":               toolHits,
		"models":              modelHits,
	}, nil
}

func ExportTrajectories(inPath, outPath string, filter FilterOptions) (map[string]any, error) {
	inFile, err := os.Open(inPath)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, err
	}
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	w := bufio.NewWriter(outFile)
	defer w.Flush()
	sc := bufio.NewScanner(inFile)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	total := 0
	exported := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		total++
		var t Trajectory
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			continue
		}
		if !matchTrajectoryFilter(t, filter) {
			continue
		}
		record := map[string]any{
			"id":         t.ID,
			"session_id": t.SessionID,
			"input":      t.Input,
			"success":    t.Success,
			"error":      t.Error,
			"reward":     t.Reward,
			"outcome":    t.Outcome,
			"messages":   flattenMessages(t.Result),
			"events":     flattenEvents(t.Events),
			"provider":   modelFromEvents(t.Events),
			"label":      boolToLabel(t.Success),
		}
		bs, _ := json.Marshal(record)
		if _, err := w.Write(append(bs, '\n')); err != nil {
			return nil, err
		}
		exported++
		if filter.Limit > 0 && exported >= filter.Limit {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"success":  true,
		"input":    inPath,
		"output":   outPath,
		"total":    total,
		"exported": exported,
	}, nil
}

func ParseSuccessFilter(v string) (*bool, error) {
	s := strings.TrimSpace(strings.ToLower(v))
	if s == "" || s == "all" {
		return nil, nil
	}
	if s == "true" || s == "success" || s == "1" {
		b := true
		return &b, nil
	}
	if s == "false" || s == "failed" || s == "0" {
		b := false
		return &b, nil
	}
	parsed, err := strconv.ParseBool(s)
	if err != nil {
		return nil, fmt.Errorf("invalid success filter: %s", v)
	}
	return &parsed, nil
}

func matchTrajectoryFilter(t Trajectory, filter FilterOptions) bool {
	if filter.Success != nil && t.Success != *filter.Success {
		return false
	}
	tool := strings.TrimSpace(filter.Tool)
	if tool != "" {
		matched := false
		for _, evt := range t.Events {
			if strings.EqualFold(evt.Type, "tool_started") && strings.EqualFold(strings.TrimSpace(evt.ToolName), tool) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	model := strings.TrimSpace(filter.Model)
	if model != "" && !strings.EqualFold(modelFromEvents(t.Events), model) {
		return false
	}
	return true
}

func modelFromEvents(events []core.AgentEvent) string {
	for _, evt := range events {
		if !strings.EqualFold(evt.Type, "model_stream_event") || evt.Data == nil {
			continue
		}
		m := strings.TrimSpace(fmt.Sprintf("%v", evt.Data["model"]))
		if m != "" {
			return m
		}
	}
	return ""
}

func flattenMessages(result *core.RunResult) []map[string]any {
	if result == nil || len(result.Messages) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(result.Messages))
	for _, m := range result.Messages {
		out = append(out, map[string]any{
			"role":    m.Role,
			"content": m.Content,
			"name":    m.Name,
		})
	}
	return out
}

func flattenEvents(events []core.AgentEvent) []map[string]any {
	if len(events) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(events))
	for _, evt := range events {
		row := map[string]any{
			"type":    evt.Type,
			"turn":    evt.Turn,
			"content": evt.Content,
		}
		if evt.ToolName != "" {
			row["tool_name"] = evt.ToolName
		}
		if len(evt.Data) > 0 {
			row["data"] = evt.Data
		}
		out = append(out, row)
	}
	return out
}

func boolToLabel(ok bool) string {
	if ok {
		return "success"
	}
	return "failed"
}

func IsJSONL(path string) bool {
	s := strings.TrimSpace(strings.ToLower(path))
	return strings.HasSuffix(s, ".jsonl")
}

func NormalizeJSONL(line string) string {
	return strings.TrimSpace(line)
}

func PrettyCompactJSON(v any) string {
	bs, _ := json.Marshal(v)
	buf := bytes.NewBuffer(nil)
	_ = json.Compact(buf, bs)
	return buf.String()
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
