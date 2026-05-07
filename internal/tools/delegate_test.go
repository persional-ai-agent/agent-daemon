package tools

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeDelegateRunner struct{}

func (fakeDelegateRunner) RunSubtask(_ context.Context, parentSessionID, goal, taskContext string, maxIterations int) (map[string]any, error) {
	return map[string]any{
		"parent_session_id": parentSessionID,
		"goal":              goal,
		"context":           taskContext,
		"max_iterations":    maxIterations,
	}, nil
}

func TestDelegateTaskSingle(t *testing.T) {
	b := &BuiltinTools{}
	res, err := b.delegateTask(context.Background(), map[string]any{
		"goal":           "fix bug",
		"context":        "repo context",
		"max_iterations": 7.0,
	}, ToolContext{SessionID: "s1", DelegateRunner: fakeDelegateRunner{}})
	if err != nil {
		t.Fatal(err)
	}
	if res["goal"] != "fix bug" || res["parent_session_id"] != "s1" {
		t.Fatalf("unexpected delegate result: %+v", res)
	}
	if res["status"] != "completed" || res["success"] != true {
		t.Fatalf("expected completed status, got %+v", res)
	}
}

func TestDelegateTaskBatch(t *testing.T) {
	b := &BuiltinTools{}
	res, err := b.delegateTask(context.Background(), map[string]any{
		"tasks": []any{
			map[string]any{"goal": "task-1", "context": "ctx-1"},
			map[string]any{"goal": "task-2", "context": "ctx-2"},
		},
		"max_iterations": 3.0,
	}, ToolContext{SessionID: "root", DelegateRunner: fakeDelegateRunner{}})
	if err != nil {
		t.Fatal(err)
	}
	if res["count"] != 2 {
		t.Fatalf("unexpected batch result: %+v", res)
	}
	if res["status"] != "completed" || res["success"] != true {
		t.Fatalf("expected completed batch status, got %+v", res)
	}
	results, _ := res["results"].([]map[string]any)
	if len(results) != 2 || results[0]["goal"] != "task-1" || results[1]["goal"] != "task-2" {
		t.Fatalf("unexpected batch order: %+v", res)
	}
	if results[0]["status"] != "completed" || results[1]["status"] != "completed" {
		t.Fatalf("expected completed item status, got %+v", res)
	}
}

type blockingDelegateRunner struct {
	started chan string
	release chan struct{}
	mu      sync.Mutex
	count   int
}

func (r *blockingDelegateRunner) RunSubtask(_ context.Context, parentSessionID, goal, taskContext string, maxIterations int) (map[string]any, error) {
	r.mu.Lock()
	r.count++
	r.mu.Unlock()
	r.started <- goal
	<-r.release
	return map[string]any{
		"parent_session_id": parentSessionID,
		"goal":              goal,
		"context":           taskContext,
		"max_iterations":    maxIterations,
	}, nil
}

func TestDelegateTaskBatchRunsConcurrently(t *testing.T) {
	b := &BuiltinTools{}
	runner := &blockingDelegateRunner{
		started: make(chan string, 3),
		release: make(chan struct{}),
	}

	done := make(chan struct{})
	var (
		res map[string]any
		err error
	)
	go func() {
		res, err = b.delegateTask(context.Background(), map[string]any{
			"tasks": []any{
				map[string]any{"goal": "task-1", "context": "ctx-1"},
				map[string]any{"goal": "task-2", "context": "ctx-2"},
				map[string]any{"goal": "task-3", "context": "ctx-3"},
			},
		}, ToolContext{SessionID: "root", DelegateRunner: runner})
		close(done)
	}()

	seen := map[string]bool{}
	timeout := time.After(2 * time.Second)
	for len(seen) < 3 {
		select {
		case goal := <-runner.started:
			seen[goal] = true
		case <-timeout:
			t.Fatalf("expected all tasks to start before release, saw=%v", seen)
		}
	}

	close(runner.release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("delegateTask did not finish after release")
	}

	if err != nil {
		t.Fatal(err)
	}
	if res["count"] != 3 {
		t.Fatalf("unexpected batch result: %+v", res)
	}
	results, _ := res["results"].([]map[string]any)
	if len(results) != 3 || results[0]["goal"] != "task-1" || results[1]["goal"] != "task-2" || results[2]["goal"] != "task-3" {
		t.Fatalf("unexpected concurrent batch order: %+v", res)
	}
}

func TestDelegateTaskBatchRespectsMaxConcurrency(t *testing.T) {
	b := &BuiltinTools{}
	runner := &blockingDelegateRunner{
		started: make(chan string, 3),
		release: make(chan struct{}),
	}

	done := make(chan struct{})
	var (
		res map[string]any
		err error
	)
	go func() {
		res, err = b.delegateTask(context.Background(), map[string]any{
			"tasks": []any{
				map[string]any{"goal": "task-1", "context": "ctx-1"},
				map[string]any{"goal": "task-2", "context": "ctx-2"},
				map[string]any{"goal": "task-3", "context": "ctx-3"},
			},
			"max_concurrency": 2.0,
		}, ToolContext{SessionID: "root", DelegateRunner: runner})
		close(done)
	}()

	seen := map[string]bool{}
	timeout := time.After(2 * time.Second)
	for len(seen) < 2 {
		select {
		case goal := <-runner.started:
			seen[goal] = true
		case <-timeout:
			t.Fatalf("expected two tasks to start before release, saw=%v", seen)
		}
	}

	select {
	case goal := <-runner.started:
		t.Fatalf("expected max_concurrency to cap starts at 2 before release, but saw extra start: %s", goal)
	case <-time.After(150 * time.Millisecond):
	}

	close(runner.release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("delegateTask did not finish after release")
	}

	if err != nil {
		t.Fatal(err)
	}
	if res["count"] != 3 {
		t.Fatalf("unexpected limited batch result: %+v", res)
	}
	results, _ := res["results"].([]map[string]any)
	if len(results) != 3 || results[0]["goal"] != "task-1" || results[1]["goal"] != "task-2" || results[2]["goal"] != "task-3" {
		t.Fatalf("unexpected limited batch order: %+v", res)
	}
}

type timeoutDelegateRunner struct{}

func (timeoutDelegateRunner) RunSubtask(ctx context.Context, parentSessionID, goal, taskContext string, maxIterations int) (map[string]any, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestDelegateTaskSingleTimeout(t *testing.T) {
	b := &BuiltinTools{}
	res, err := b.delegateTask(context.Background(), map[string]any{
		"goal":            "slow-task",
		"timeout_seconds": 1.0,
	}, ToolContext{SessionID: "s-timeout", DelegateRunner: timeoutDelegateRunner{}})
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "timeout" || res["success"] != false {
		t.Fatalf("expected timeout status, got %+v", res)
	}
	errText, _ := res["error"].(string)
	if !strings.Contains(errText, context.DeadlineExceeded.Error()) {
		t.Fatalf("expected deadline exceeded error text, got %+v", res)
	}
}

type failFastDelegateRunner struct {
	started chan string
}

func (r *failFastDelegateRunner) RunSubtask(ctx context.Context, parentSessionID, goal, taskContext string, maxIterations int) (map[string]any, error) {
	r.started <- goal
	if goal == "task-1" {
		return nil, errors.New("task-1 failed")
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestDelegateTaskBatchFailFastCancelsSiblings(t *testing.T) {
	b := &BuiltinTools{}
	runner := &failFastDelegateRunner{started: make(chan string, 3)}

	res, err := b.delegateTask(context.Background(), map[string]any{
		"tasks": []any{
			map[string]any{"goal": "task-1", "context": "ctx-1"},
			map[string]any{"goal": "task-2", "context": "ctx-2"},
			map[string]any{"goal": "task-3", "context": "ctx-3"},
		},
		"fail_fast": true,
	}, ToolContext{SessionID: "root", DelegateRunner: runner})
	if err != nil {
		t.Fatal(err)
	}
	results, _ := res["results"].([]map[string]any)
	if len(results) != 3 {
		t.Fatalf("unexpected fail_fast result: %+v", res)
	}
	if res["status"] != "failed" || res["success"] != false {
		t.Fatalf("expected failed batch status, got %+v", res)
	}
	if results[0]["error"] != "task-1 failed" {
		t.Fatalf("expected first task failure, got %+v", results[0])
	}
	for i := 1; i < len(results); i++ {
		errText, _ := results[i]["error"].(string)
		if errText == "" {
			t.Fatalf("expected sibling %d to be cancelled, got %+v", i, results[i])
		}
		if results[i]["status"] != "cancelled" {
			t.Fatalf("expected sibling %d cancelled status, got %+v", i, results[i])
		}
	}
}
