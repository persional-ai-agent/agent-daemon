package gateway

import (
	"strings"
	"testing"
)

func TestIsBuiltInGatewayCommandCaseInsensitive(t *testing.T) {
	if !IsBuiltInGatewayCommand("APPROVE") {
		t.Fatal("expected APPROVE to be built-in")
	}
	if !IsBuiltInGatewayCommand(" status ") {
		t.Fatal("expected status to be built-in")
	}
	if IsBuiltInGatewayCommand("unknown") {
		t.Fatal("unknown should not be built-in")
	}
}

func TestBuiltInGatewaySlashCommandsContainsCoreSet(t *testing.T) {
	list := BuiltInGatewaySlashCommands()
	want := map[string]bool{
		"/pair": true, "/unpair": true, "/cancel": true, "/queue": true,
		"/status": true, "/pending": true, "/approvals": true, "/grant": true,
		"/revoke": true, "/approve": true, "/deny": true, "/help": true,
	}
	for _, v := range list {
		delete(want, v)
	}
	if len(want) != 0 {
		t.Fatalf("missing built-ins: %+v", want)
	}
}

func TestResolveGatewayCommandAliases(t *testing.T) {
	cases := map[string]string{
		"approval": "approvals",
		"pendings": "pending",
		"abort":    "cancel",
		"stop":     "cancel",
		"q":        "queue",
		"s":        "status",
		"h":        "help",
		"approve":  "approve",
	}
	for in, want := range cases {
		got, ok := ResolveGatewayCommand(in)
		if !ok || got != want {
			t.Fatalf("resolve(%q)=(%q,%v) want=(%q,true)", in, got, ok, want)
		}
	}
	if got, ok := ResolveGatewayCommand("unknown"); ok || got != "" {
		t.Fatalf("resolve unknown got=(%q,%v)", got, ok)
	}
}

func TestGatewayHelpTextIncludesBuiltIns(t *testing.T) {
	help := GatewayHelpText(false)
	if !strings.HasPrefix(help, "Commands: ") {
		t.Fatalf("unexpected help prefix: %q", help)
	}
	for _, name := range BuiltInGatewaySlashCommands() {
		if !strings.Contains(help, name) {
			t.Fatalf("help missing command %s: %q", name, help)
		}
	}
	if !strings.Contains(help, "/grant pattern <name> [ttl]") {
		t.Fatalf("help missing grant pattern usage: %q", help)
	}
	if !strings.Contains(help, "/revoke pattern <name>") {
		t.Fatalf("help missing revoke pattern usage: %q", help)
	}
}

func TestGatewayHelpTextYuanbaoAddsQuickReplies(t *testing.T) {
	help := GatewayHelpText(true)
	if !strings.Contains(help, "Quick reply aliases:") {
		t.Fatalf("expected quick reply aliases in help: %q", help)
	}
}

func TestGatewayCommandAliasesIntegrity(t *testing.T) {
	aliases := GatewayCommandAliases()
	if len(aliases) == 0 {
		t.Fatal("expected non-empty alias map")
	}
	for alias, canonical := range aliases {
		if alias == canonical {
			t.Fatalf("alias should not equal canonical: %q", alias)
		}
		if IsBuiltInGatewayCommand(alias) {
			t.Fatalf("alias should not shadow built-in: %q", alias)
		}
		if !IsBuiltInGatewayCommand(canonical) {
			t.Fatalf("alias points to non built-in command: %q -> %q", alias, canonical)
		}
	}
}
