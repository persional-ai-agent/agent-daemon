package main

import "testing"

func TestSlashCompletionsRoot(t *testing.T) {
	got := slashCompletions("/re")
	if len(got) == 0 {
		t.Fatal("expected root completions")
	}
	foundReconnect := false
	for _, v := range got {
		if v == "/reconnect" {
			foundReconnect = true
			break
		}
	}
	if !foundReconnect {
		t.Fatalf("expected /reconnect in completions: %v", got)
	}
}

func TestSlashCompletionsSubcommand(t *testing.T) {
	got := slashCompletions("/reconnect t")
	if len(got) != 1 || got[0] != "/reconnect timeout" {
		t.Fatalf("unexpected completions: %v", got)
	}
	got = slashCompletions("/reconnect timeout ")
	if len(got) == 0 {
		t.Fatal("expected third-token completions")
	}
}

func TestTUIModelApplyCompletionCycle(t *testing.T) {
	s := newState()
	m := newTUIModel(s, true)
	m.inputValue = "/re"
	m.applyCompletion()
	first := m.inputValue
	if first == "/re" {
		t.Fatalf("expected completion applied")
	}
	m.applyCompletion()
	second := m.inputValue
	if second == first && len(m.compItems) > 1 {
		t.Fatalf("expected cycle to next completion, got same: %q", second)
	}
}
