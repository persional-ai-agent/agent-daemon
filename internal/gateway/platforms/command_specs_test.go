package platforms

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestGatewayPlatformCommandSpecsShape(t *testing.T) {
	specs := GatewayPlatformCommandSpecs()
	if len(specs) == 0 {
		t.Fatal("empty command specs")
	}

	seen := map[string]bool{}
	for _, s := range specs {
		if s.Name == "" {
			t.Fatal("empty command name")
		}
		if seen[s.Name] {
			t.Fatalf("duplicate command name in specs: %s", s.Name)
		}
		seen[s.Name] = true
		if s.Description == "" {
			t.Fatalf("empty description for command: %s", s.Name)
		}
		if !s.Telegram || !s.Discord {
			t.Fatalf("command should be enabled for telegram+discord: %s", s.Name)
		}
	}
}

func TestGatewayPlatformCommandSpecsDiscordOptionsContract(t *testing.T) {
	specByName := map[string]gatewayPlatformCommandSpec{}
	for _, s := range GatewayPlatformCommandSpecs() {
		specByName[s.Name] = s
	}

	pair := specByName["pair"]
	if len(pair.DiscordOpts) != 1 || pair.DiscordOpts[0].Name != "code" || pair.DiscordOpts[0].Type != discordgo.ApplicationCommandOptionString || !pair.DiscordOpts[0].Required {
		t.Fatalf("pair option mismatch: %+v", pair.DiscordOpts)
	}

	grant := specByName["grant"]
	if len(grant.DiscordOpts) != 2 {
		t.Fatalf("grant options mismatch: %+v", grant.DiscordOpts)
	}
	if grant.DiscordOpts[0].Name != "pattern" || grant.DiscordOpts[0].Type != discordgo.ApplicationCommandOptionString || grant.DiscordOpts[0].Required {
		t.Fatalf("grant pattern option mismatch: %+v", grant.DiscordOpts[0])
	}
	if grant.DiscordOpts[1].Name != "ttl" || grant.DiscordOpts[1].Type != discordgo.ApplicationCommandOptionInteger || grant.DiscordOpts[1].Required {
		t.Fatalf("grant ttl option mismatch: %+v", grant.DiscordOpts[1])
	}

	revoke := specByName["revoke"]
	if len(revoke.DiscordOpts) != 1 || revoke.DiscordOpts[0].Name != "pattern" || revoke.DiscordOpts[0].Type != discordgo.ApplicationCommandOptionString || revoke.DiscordOpts[0].Required {
		t.Fatalf("revoke option mismatch: %+v", revoke.DiscordOpts)
	}

	approve := specByName["approve"]
	if len(approve.DiscordOpts) != 1 || approve.DiscordOpts[0].Name != "id" || approve.DiscordOpts[0].Type != discordgo.ApplicationCommandOptionString || approve.DiscordOpts[0].Required {
		t.Fatalf("approve option mismatch: %+v", approve.DiscordOpts)
	}
	deny := specByName["deny"]
	if len(deny.DiscordOpts) != 1 || deny.DiscordOpts[0].Name != "id" || deny.DiscordOpts[0].Type != discordgo.ApplicationCommandOptionString || deny.DiscordOpts[0].Required {
		t.Fatalf("deny option mismatch: %+v", deny.DiscordOpts)
	}
}
