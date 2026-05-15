package main

import (
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
