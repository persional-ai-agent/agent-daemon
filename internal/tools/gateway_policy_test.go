package tools

import "testing"

func TestNormalizeGatewayGroupDMPolicy(t *testing.T) {
	if got := NormalizeGatewayGroupDMPolicy("group"); got != "group_only" {
		t.Fatalf("group policy=%q", got)
	}
	if got := NormalizeGatewayGroupDMPolicy("dm_only"); got != "dm_only" {
		t.Fatalf("dm policy=%q", got)
	}
	if got := NormalizeGatewayGroupDMPolicy("whatever"); got != "both" {
		t.Fatalf("default policy=%q", got)
	}
}

func TestParsePolicyLists(t *testing.T) {
	ch := ParsePolicyChannelList(" telegram:1001,1001,telegram:1001 ,,")
	if len(ch) != 2 || ch[0] != "telegram:1001" || ch[1] != "1001" {
		t.Fatalf("unexpected channel list: %#v", ch)
	}
	mk := ParseMentionKeywords(" @bot, @agent,@bot ")
	if len(mk) != 2 || mk[0] != "@bot" || mk[1] != "@agent" {
		t.Fatalf("unexpected mention keywords: %#v", mk)
	}
}

func TestResolveAndMatchGatewayInteractionPolicy(t *testing.T) {
	workdir := t.TempDir()
	if err := SetGatewaySetting(workdir, "policy_mention_required", "true"); err != nil {
		t.Fatal(err)
	}
	if err := SetGatewaySetting(workdir, "policy_group_dm", "group_only"); err != nil {
		t.Fatal(err)
	}
	if err := SetGatewaySetting(workdir, "policy_ignored_channels", "telegram:1001"); err != nil {
		t.Fatal(err)
	}
	if err := SetGatewaySetting(workdir, "policy_free_response_channels", "telegram:2002"); err != nil {
		t.Fatal(err)
	}
	p, err := ResolveGatewayInteractionPolicy(workdir)
	if err != nil {
		t.Fatal(err)
	}
	if !p.MentionRequired || p.GroupDMPolicy != "group_only" {
		t.Fatalf("unexpected policy: %+v", p)
	}
	if !MatchPolicyChannel(p.IgnoredChannels, "telegram", "1001") {
		t.Fatalf("expected ignored channel match: %+v", p.IgnoredChannels)
	}
	if MatchPolicyChannel(p.IgnoredChannels, "telegram", "9999") {
		t.Fatalf("unexpected ignored channel match: %+v", p.IgnoredChannels)
	}
}
