package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlCCancelsWhenProcessing(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.processing = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	next, ok := updated.(tuiModel)
	if !ok {
		t.Fatalf("unexpected model type: %T", updated)
	}
	if !next.processing {
		t.Fatalf("processing should remain true until turnDone")
	}
	if cmd != nil {
		t.Fatal("expected no quit cmd while processing")
	}
}

func TestCtrlCQuitsWhenIdle(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.processing = false
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit cmd when idle")
	}
}

func TestTurnDoneCancelledUpdatesStatus(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.processing = true
	m.turnStream = make(chan tea.Msg, 1)

	updated, _ := m.Update(turnDoneMsg{err: errTurnCancelled})
	next := updated.(tuiModel)
	if next.processing {
		t.Fatal("expected processing=false after turn done")
	}
	if s.lastCode != "cancelled" {
		t.Fatalf("expected cancelled code, got %q", s.lastCode)
	}
}

func TestInsertAtCursorAndMove(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.insertAtCursor("abc")
	if m.inputValue != "abc" || m.cursorPos != 3 {
		t.Fatalf("unexpected input=%q cursor=%d", m.inputValue, m.cursorPos)
	}
	m.cursorPos = 1
	m.insertAtCursor("X")
	if m.inputValue != "aXbc" || m.cursorPos != 2 {
		t.Fatalf("unexpected input=%q cursor=%d", m.inputValue, m.cursorPos)
	}
}

func TestCtrlJInsertNewline(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.inputValue = "hello"
	m.cursorPos = 5
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	next := updated.(tuiModel)
	if next.inputValue != "hello\n" {
		t.Fatalf("unexpected input=%q", next.inputValue)
	}
}

func TestEnterAndCtrlSShareSubmitPath(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.inputValue = "hello"
	m.cursorPos = len([]rune(m.inputValue))

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(tuiModel)
	if !next.processing {
		t.Fatal("expected enter to start turn processing")
	}
	if cmd == nil {
		t.Fatal("expected enter to return async command")
	}
	if next.turnStream == nil {
		t.Fatal("expected turn stream to be created")
	}
	if cap(next.turnStream) != turnStreamBufferSize {
		t.Fatalf("unexpected turn stream buffer size: %d", cap(next.turnStream))
	}

	m2 := newTUIModel(newState(), true)
	m2.inputValue = "hello"
	m2.cursorPos = len([]rune(m2.inputValue))
	updated, cmd = m2.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	next2 := updated.(tuiModel)
	if !next2.processing {
		t.Fatal("expected ctrl+s to start turn processing")
	}
	if cmd == nil {
		t.Fatal("expected ctrl+s to return async command")
	}
	if next2.turnStream == nil {
		t.Fatal("expected ctrl+s turn stream to be created")
	}
	if cap(next2.turnStream) != turnStreamBufferSize {
		t.Fatalf("unexpected ctrl+s turn stream buffer size: %d", cap(next2.turnStream))
	}
}

func TestSubmitClearCommandResetsRuntimeWithoutProcessing(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.inputValue = "/clear"
	m.cursorPos = len([]rune(m.inputValue))

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(tuiModel)
	if next.processing {
		t.Fatal("clear should not start processing")
	}
	if cmd != nil {
		t.Fatal("clear should not return async command")
	}
	if next.turnStream != nil {
		t.Fatal("clear should not create turn stream")
	}
}

func TestRenderInputWithCursor(t *testing.T) {
	out := renderInputWithCursor("ab\ncd", 2)
	if out == "" {
		t.Fatal("expected rendered input")
	}
	if !strings.HasPrefix(out, "› ") {
		t.Fatalf("unexpected prefix: %q", out)
	}
}

func TestInputHistoryUpDownWithDraftRestore(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.history = []string{"/status", "/approvals", "hello world"}
	m.historyPos = len(m.history)
	m.inputValue = "/pen"
	m.cursorPos = len([]rune(m.inputValue))

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	next := updated.(tuiModel)
	if next.inputValue != "hello world" {
		t.Fatalf("expected latest history, got %q", next.inputValue)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyUp})
	next = updated.(tuiModel)
	if next.inputValue != "/approvals" {
		t.Fatalf("expected previous history, got %q", next.inputValue)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyDown})
	next = updated.(tuiModel)
	if next.inputValue != "hello world" {
		t.Fatalf("expected forward history, got %q", next.inputValue)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyDown})
	next = updated.(tuiModel)
	if next.inputValue != "/pen" {
		t.Fatalf("expected draft restored, got %q", next.inputValue)
	}
}

func TestCommitInputHistoryDedupAndCap(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.commitInputHistory("/status")
	m.commitInputHistory("/status")
	if len(m.history) != 1 {
		t.Fatalf("expected dedup history size=1, got %d", len(m.history))
	}
	for i := 0; i < 250; i++ {
		m.commitInputHistory("cmd" + string(rune('a'+(i%26))))
	}
	if len(m.history) > 200 {
		t.Fatalf("expected history cap <= 200, got %d", len(m.history))
	}
}
