package platforms

import (
	"sort"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestTelegramDiscordApprovalCommandsConsistency(t *testing.T) {
	telegram := map[string]bool{}
	for _, c := range TelegramCommands() {
		telegram["/"+c.Command] = true
	}
	discord := map[string]bool{}
	for _, c := range DiscordApplicationCommands() {
		discord["/"+c.Name] = true
	}
	must := []string{"/approve", "/deny", "/pending", "/approvals", "/grant", "/revoke", "/status", "/help"}
	for _, name := range must {
		if !telegram[name] {
			t.Fatalf("telegram missing command: %s", name)
		}
		if !discord[name] {
			t.Fatalf("discord missing command: %s", name)
		}
	}
}

func TestGatewayBuiltInCommandsCrossPlatformConsistency(t *testing.T) {
	telegram := map[string]bool{}
	for _, c := range TelegramCommands() {
		telegram["/"+c.Command] = true
	}
	discord := map[string]bool{}
	for _, c := range DiscordApplicationCommands() {
		discord["/"+c.Name] = true
	}
	shared := gateway.BuiltInGatewaySlashCommands()
	for _, name := range shared {
		if !telegram[name] {
			t.Fatalf("telegram missing command: %s", name)
		}
		if !discord[name] {
			t.Fatalf("discord missing command: %s", name)
		}
		if !isBuiltInGatewaySlashCommand(name[1:]) {
			t.Fatalf("slack builtin list missing command: %s", name)
		}
	}

	// Exact-set match between platform command definitions and gateway catalog.
	expect := map[string]bool{}
	for _, name := range shared {
		expect[name] = true
	}
	if len(telegram) != len(expect) {
		t.Fatalf("telegram command size mismatch: got=%d want=%d", len(telegram), len(expect))
	}
	if len(discord) != len(expect) {
		t.Fatalf("discord command size mismatch: got=%d want=%d", len(discord), len(expect))
	}
	for name := range telegram {
		if !expect[name] {
			t.Fatalf("telegram has extra command not in catalog: %s", name)
		}
	}
	for name := range discord {
		if !expect[name] {
			t.Fatalf("discord has extra command not in catalog: %s", name)
		}
	}
}

func TestGatewayCatalogAndSlackBuiltinSetMatch(t *testing.T) {
	fromCatalog := gateway.BuiltInGatewayCommandNames()
	fromSlack := make([]string, 0, len(fromCatalog))
	for _, name := range fromCatalog {
		if isBuiltInGatewaySlashCommand(name) {
			fromSlack = append(fromSlack, name)
		}
	}
	sort.Strings(fromCatalog)
	sort.Strings(fromSlack)
	if len(fromCatalog) != len(fromSlack) {
		t.Fatalf("slack builtin set size mismatch: catalog=%d slack=%d", len(fromCatalog), len(fromSlack))
	}
	for i := range fromCatalog {
		if fromCatalog[i] != fromSlack[i] {
			t.Fatalf("slack/catalog mismatch at %d: %q vs %q", i, fromCatalog[i], fromSlack[i])
		}
	}
}
