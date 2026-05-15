package platforms

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestSlackDiscordCommandNormalizationConsistency(t *testing.T) {
	cases := []struct {
		name    string
		slack   string
		discord string
		want    string
	}{
		{name: "status", slack: "/STATUS", discord: "STATUS", want: "/status"},
		{name: "approval alias", slack: "/approval", discord: "approval", want: "/approvals"},
		{name: "cancel alias", slack: "/abort", discord: "abort", want: "/cancel"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSlack := renderSlackSlashCommand(fakeSlashCommand("/agent", tc.slack))
			gotDiscord := renderDiscordSlashCommand(discordgo.ApplicationCommandInteractionData{Name: tc.discord})
			if gotSlack != tc.want {
				t.Fatalf("slack got=%q want=%q", gotSlack, tc.want)
			}
			if gotDiscord != tc.want {
				t.Fatalf("discord got=%q want=%q", gotDiscord, tc.want)
			}
		})
	}
}

func TestSlashNormalizationConsistencyAcrossPlatforms(t *testing.T) {
	cases := []struct {
		name      string
		slack     string
		discord   string
		telegram  string
		yuanbao   string
		want      string
	}{
		{
			name:     "status",
			slack:    "/STATUS",
			discord:  "STATUS",
			telegram: "/Status@agent_bot",
			yuanbao:  "/status",
			want:     "/status",
		},
		{
			name:     "approval alias",
			slack:    "/approval",
			discord:  "approval",
			telegram: "/Approval@agent_bot",
			yuanbao:  "/approval",
			want:     "/approvals",
		},
		{
			name:     "cancel alias",
			slack:    "/abort",
			discord:  "abort",
			telegram: "/Abort@agent_bot",
			yuanbao:  "/abort",
			want:     "/cancel",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSlack := renderSlackSlashCommand(fakeSlashCommand("/agent", tc.slack))
			gotDiscord := renderDiscordSlashCommand(discordgo.ApplicationCommandInteractionData{Name: tc.discord})
			gotTelegram := normalizeInboundSlashText(stripTelegramCommandMention(tc.telegram))
			gotYuanbao := normalizeInboundSlashText(tc.yuanbao)
			if gotSlack != tc.want {
				t.Fatalf("slack got=%q want=%q", gotSlack, tc.want)
			}
			if gotDiscord != tc.want {
				t.Fatalf("discord got=%q want=%q", gotDiscord, tc.want)
			}
			if gotTelegram != tc.want {
				t.Fatalf("telegram got=%q want=%q", gotTelegram, tc.want)
			}
			if gotYuanbao != tc.want {
				t.Fatalf("yuanbao got=%q want=%q", gotYuanbao, tc.want)
			}
		})
	}
}

func stripTelegramCommandMention(text string) string {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return text
	}
	head := parts[0]
	if !strings.HasPrefix(head, "/") {
		return text
	}
	if idx := strings.Index(head, "@"); idx > 1 {
		head = head[:idx]
	}
	parts[0] = head
	return strings.Join(parts, " ")
}
