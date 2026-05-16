package tools

import "testing"

func TestBuildSlashPayload(t *testing.T) {
	got := BuildSlashPayload(" /help ")
	if got["slash"] != "/help" {
		t.Fatalf("unexpected slash payload: %+v", got)
	}
}

func TestBuildSlashModePayload(t *testing.T) {
	got := BuildSlashModePayload(" /personality ", " reset ")
	if got["slash"] != "/personality" || got["mode"] != "reset" {
		t.Fatalf("unexpected slash mode payload: %+v", got)
	}
}

func TestBuildSlashSubcommandPayload(t *testing.T) {
	got := BuildSlashSubcommandPayload(" /tools ", " show ")
	if got["slash"] != "/tools" || got["subcommand"] != "show" {
		t.Fatalf("unexpected slash subcommand payload: %+v", got)
	}
}

func TestBuildSlashModePayloadWithExtra(t *testing.T) {
	got := BuildSlashModePayloadWithExtra("/skills", "show", map[string]any{"name": "abc"})
	if got["slash"] != "/skills" || got["mode"] != "show" || got["name"] != "abc" {
		t.Fatalf("unexpected mode payload with extra: %+v", got)
	}
}

func TestBuildSlashSubcommandPayloadWithExtra(t *testing.T) {
	got := BuildSlashSubcommandPayloadWithExtra("/tools", "show", map[string]any{"tool": "send_message"})
	if got["slash"] != "/tools" || got["subcommand"] != "show" || got["tool"] != "send_message" {
		t.Fatalf("unexpected subcommand payload with extra: %+v", got)
	}
}

func TestAttachSlashPayload(t *testing.T) {
	got := AttachSlashPayload(map[string]any{"mode": "show", "name": "abc"}, " /skills ")
	if got["slash"] != "/skills" {
		t.Fatalf("unexpected slash: %+v", got)
	}
	if got["mode"] != "show" || got["name"] != "abc" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}
