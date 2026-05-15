package gateway

import "testing"

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
