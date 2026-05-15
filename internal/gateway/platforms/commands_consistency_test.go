package platforms

import (
	"sort"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestGatewayPlatformCommandSpecsCoverCatalog(t *testing.T) {
	specs := GatewayPlatformCommandSpecs()
	if len(specs) == 0 {
		t.Fatal("empty platform command specs")
	}
	fromCatalog := map[string]bool{}
	for _, name := range gateway.BuiltInGatewayCommandNames() {
		fromCatalog[name] = true
	}
	for _, spec := range specs {
		if !fromCatalog[spec.Name] {
			t.Fatalf("spec has command not in gateway catalog: %s", spec.Name)
		}
		if !spec.Telegram || !spec.Discord {
			t.Fatalf("spec must be enabled for both telegram/discord: %s", spec.Name)
		}
		if spec.Description == "" {
			t.Fatalf("spec description empty: %s", spec.Name)
		}
		delete(fromCatalog, spec.Name)
	}
	if len(fromCatalog) != 0 {
		t.Fatalf("catalog commands missing in platform specs: %+v", fromCatalog)
	}
}

func TestTelegramCommandsExactlyFromSpecs(t *testing.T) {
	specs := GatewayPlatformCommandSpecs()
	expected := telegramCommandsFromSpecs(specs)
	actual := TelegramCommands()
	if len(actual) != len(expected) {
		t.Fatalf("telegram command count mismatch: got=%d want=%d", len(actual), len(expected))
	}
	for i := range expected {
		if actual[i].Command != expected[i].Command || actual[i].Description != expected[i].Description {
			t.Fatalf("telegram command mismatch at %d: got=%+v want=%+v", i, actual[i], expected[i])
		}
	}
}

func TestDiscordCommandsExactlyFromSpecs(t *testing.T) {
	specs := GatewayPlatformCommandSpecs()
	expected := discordCommandsFromSpecs(specs)
	actual := DiscordApplicationCommands()
	if len(actual) != len(expected) {
		t.Fatalf("discord command count mismatch: got=%d want=%d", len(actual), len(expected))
	}
	for i := range expected {
		if actual[i].Name != expected[i].Name || actual[i].Description != expected[i].Description {
			t.Fatalf("discord command mismatch at %d: got=(%s,%s) want=(%s,%s)", i, actual[i].Name, actual[i].Description, expected[i].Name, expected[i].Description)
		}
		if len(actual[i].Options) != len(expected[i].Options) {
			t.Fatalf("discord option count mismatch for %s: got=%d want=%d", actual[i].Name, len(actual[i].Options), len(expected[i].Options))
		}
		for j := range expected[i].Options {
			gotOpt := actual[i].Options[j]
			wantOpt := expected[i].Options[j]
			if gotOpt.Name != wantOpt.Name || gotOpt.Description != wantOpt.Description || gotOpt.Type != wantOpt.Type || gotOpt.Required != wantOpt.Required {
				t.Fatalf("discord option mismatch for %s[%d]: got=%+v want=%+v", actual[i].Name, j, gotOpt, wantOpt)
			}
		}
	}
}

func TestTelegramDiscordApprovalCommandsConsistency(t *testing.T) {
	telegram := map[string]bool{}
	for _, c := range TelegramCommands() {
		telegram["/"+c.Command] = true
	}
	discord := map[string]bool{}
	for _, c := range DiscordApplicationCommands() {
		discord["/"+c.Name] = true
	}
	must := gateway.GatewayApprovalSlashCommands()
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
