package platforms

import (
	"context"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
	"github.com/bwmarrin/discordgo"
)

func TestDiscordAdapterName(t *testing.T) {
	a := &DiscordAdapter{}
	if a.Name() != "discord" {
		t.Errorf("Name() = %q, want %q", a.Name(), "discord")
	}
}

func TestDiscordAdapterOnMessage(t *testing.T) {
	a := &DiscordAdapter{}
	called := false
	a.OnMessage(context.Background(), func(_ context.Context, _ gateway.MessageEvent) {
		called = true
	})
	if a.handler == nil {
		t.Error("handler should not be nil after OnMessage")
	}
	_ = called
}

func TestTruncateDiscord(t *testing.T) {
	if s := truncateDiscord("hello", 2000); s != "hello" {
		t.Errorf("short string: %q", s)
	}
	long := make([]byte, 3000)
	for i := range long {
		long[i] = 'x'
	}
	s := truncateDiscord(string(long), 2000)
	if s[:3] != "xxx" {
		t.Errorf("should start with xxx: %q", s[:3])
	}
	if s[len(s)-3:] != "..." {
		t.Errorf("should end with ...: %q", s[len(s)-3:])
	}
}

func TestRenderDiscordSlashCommand(t *testing.T) {
	tests := []struct {
		name string
		data discordgo.ApplicationCommandInteractionData
		want string
	}{
		{
			name: "pair with code",
			data: discordgo.ApplicationCommandInteractionData{
				Name: "pair",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "code", Value: "abc123"},
				},
			},
			want: "/pair abc123",
		},
		{
			name: "approve with id",
			data: discordgo.ApplicationCommandInteractionData{
				Name: "approve",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "id", Value: "ap-1"},
				},
			},
			want: "/approve ap-1",
		},
		{
			name: "grant pattern with ttl",
			data: discordgo.ApplicationCommandInteractionData{
				Name: "grant",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "pattern", Value: "tool_*"},
					{Name: "ttl", Value: float64(3600)},
				},
			},
			want: "/grant pattern tool_* 3600",
		},
		{
			name: "revoke pattern",
			data: discordgo.ApplicationCommandInteractionData{
				Name: "revoke",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "pattern", Value: "tool_*"},
				},
			},
			want: "/revoke pattern tool_*",
		},
		{
			name: "simple built-in",
			data: discordgo.ApplicationCommandInteractionData{Name: "status"},
			want: "/status",
		},
		{
			name: "uppercase built-in",
			data: discordgo.ApplicationCommandInteractionData{Name: "STATUS"},
			want: "/status",
		},
		{
			name: "alias command",
			data: discordgo.ApplicationCommandInteractionData{Name: "approval"},
			want: "/approvals",
		},
		{
			name: "unknown command",
			data: discordgo.ApplicationCommandInteractionData{Name: "unknown"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderDiscordSlashCommand(tt.data)
			if got != tt.want {
				t.Fatalf("renderDiscordSlashCommand got=%q want=%q", got, tt.want)
			}
		})
	}
}
