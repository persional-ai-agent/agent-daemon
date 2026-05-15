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
