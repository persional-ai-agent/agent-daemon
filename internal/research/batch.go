package research

import (
	"bufio"
	"context"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	ID         string           `json:"id"`
	SessionID  string           `json:"session_id"`
	Input      string           `json:"input"`
	StartedAt  string           `json:"started_at"`
	FinishedAt string           `json:"finished_at"`
	DurationMS int64            `json:"duration_ms"`
	Success    bool             `json:"success"`
	Error      string           `json:"error,omitempty"`
	Result     *core.RunResult  `json:"result,omitempty"`
	Events     []core.AgentEvent `json:"events,omitempty"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
}

type BatchReport struct {
	Total      int    `json:"total"`
	Succeeded  int    `json:"succeeded"`
	Failed     int    `json:"failed"`
	OutputPath string `json:"output_path"`
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
	for _, task := range tasks {
		start := time.Now()
		events := make([]core.AgentEvent, 0, 32)
		engCopy := *eng
		engCopy.EventSink = func(evt core.AgentEvent) {
			events = append(events, evt)
		}
		res, runErr := engCopy.Run(ctx, task.SessionID, task.Input, agent.DefaultSystemPrompt(), nil)
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
			report.Failed++
		} else {
			item.Result = res
			report.Succeeded++
		}
		bs, _ := json.Marshal(item)
		if _, err := w.Write(append(bs, '\n')); err != nil {
			return report, err
		}
	}
	return report, nil
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
		if t.Success {
			success++
		}
		totalDuration += t.DurationMS
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
	}, nil
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
