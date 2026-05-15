package platforms

import (
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
