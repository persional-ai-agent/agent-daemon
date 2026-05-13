package tools

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiscordSchemasActionIsOptional(t *testing.T) {
	checkRequiredEmpty := func(name string, params map[string]any) {
		t.Helper()
		required, _ := params["required"].([]string)
		if len(required) != 0 && !reflect.DeepEqual(required, []string{}) {
			t.Fatalf("%s required=%v, want empty", name, required)
		}
	}

	checkRequiredEmpty("discord", discordToolParams())
	checkRequiredEmpty("discord_admin", discordAdminParams())
}

func TestDiscordSchemasDocumentDefaultAction(t *testing.T) {
	checkDefaultHint := func(name string, params map[string]any) {
		t.Helper()
		props, _ := params["properties"].(map[string]any)
		action, _ := props["action"].(map[string]any)
		desc, _ := action["description"].(string)
		if desc == "" || !containsIgnoreCase(desc, "default: list_guilds") {
			t.Fatalf("%s action description=%q, want default hint", name, desc)
		}
	}
	checkDefaultHint("discord", discordToolParams())
	checkDefaultHint("discord_admin", discordAdminParams())
}

func TestDiscordSchemaDocumentsFetchMessagesLimitBounds(t *testing.T) {
	props, _ := discordToolParams()["properties"].(map[string]any)
	limit, _ := props["limit"].(map[string]any)
	desc, _ := limit["description"].(string)
	if !strings.Contains(desc, "default 50") || !strings.Contains(desc, "max 100") {
		t.Fatalf("discord.limit description=%q, want default/max hint", desc)
	}
}

func containsIgnoreCase(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		ls := []rune(s)
		lsub := []rune(sub)
		// tiny helper without extra imports
		for i := 0; i+len(lsub) <= len(ls); i++ {
			ok := true
			for j := 0; j < len(lsub); j++ {
				a := ls[i+j]
				b := lsub[j]
				if 'A' <= a && a <= 'Z' {
					a += 'a' - 'A'
				}
				if 'A' <= b && b <= 'Z' {
					b += 'a' - 'A'
				}
				if a != b {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
		return false
	})()
}
