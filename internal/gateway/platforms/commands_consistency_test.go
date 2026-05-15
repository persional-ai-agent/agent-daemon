package platforms

import (
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
}
