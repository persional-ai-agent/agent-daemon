package tools

import (
	"reflect"
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
