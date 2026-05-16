package tools

import (
	"os"
	"strings"
)

const GatewayPolicyMentionRequiredEnvVar = "AGENT_GATEWAY_MENTION_REQUIRED"
const GatewayPolicyGroupDMEnvVar = "AGENT_GATEWAY_GROUP_DM_POLICY"
const GatewayPolicyIgnoredChannelsEnvVar = "AGENT_GATEWAY_IGNORED_CHANNELS"
const GatewayPolicyFreeResponseChannelsEnvVar = "AGENT_GATEWAY_FREE_RESPONSE_CHANNELS"
const GatewayPolicyMentionKeywordsEnvVar = "AGENT_GATEWAY_MENTION_KEYWORDS"

type GatewayInteractionPolicy struct {
	MentionRequired      bool
	GroupDMPolicy        string
	IgnoredChannels      []string
	FreeResponseChannels []string
	MentionKeywords      []string
}

func NormalizeGatewayGroupDMPolicy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "group_only", "group":
		return "group_only"
	case "dm_only", "dm":
		return "dm_only"
	default:
		return "both"
	}
}

func ParsePolicyChannelList(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func ParseMentionKeywords(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		k := strings.ToLower(v)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, v)
	}
	return out
}

func ResolveGatewayInteractionPolicy(workdir string) (GatewayInteractionPolicy, error) {
	mentionRaw, err := GetGatewaySetting(workdir, "policy_mention_required")
	if err != nil {
		return GatewayInteractionPolicy{}, err
	}
	if v := strings.TrimSpace(os.Getenv(GatewayPolicyMentionRequiredEnvVar)); v != "" {
		mentionRaw = v
	}
	groupDMRaw, err := GetGatewaySetting(workdir, "policy_group_dm")
	if err != nil {
		return GatewayInteractionPolicy{}, err
	}
	if v := strings.TrimSpace(os.Getenv(GatewayPolicyGroupDMEnvVar)); v != "" {
		groupDMRaw = v
	}
	ignoredRaw, err := GetGatewaySetting(workdir, "policy_ignored_channels")
	if err != nil {
		return GatewayInteractionPolicy{}, err
	}
	if v := strings.TrimSpace(os.Getenv(GatewayPolicyIgnoredChannelsEnvVar)); v != "" {
		ignoredRaw = v
	}
	freeRaw, err := GetGatewaySetting(workdir, "policy_free_response_channels")
	if err != nil {
		return GatewayInteractionPolicy{}, err
	}
	if v := strings.TrimSpace(os.Getenv(GatewayPolicyFreeResponseChannelsEnvVar)); v != "" {
		freeRaw = v
	}
	keywordRaw, err := GetGatewaySetting(workdir, "policy_mention_keywords")
	if err != nil {
		return GatewayInteractionPolicy{}, err
	}
	if v := strings.TrimSpace(os.Getenv(GatewayPolicyMentionKeywordsEnvVar)); v != "" {
		keywordRaw = v
	}
	p := GatewayInteractionPolicy{
		MentionRequired:      strings.EqualFold(strings.TrimSpace(mentionRaw), "true") || strings.EqualFold(strings.TrimSpace(mentionRaw), "on") || strings.TrimSpace(mentionRaw) == "1",
		GroupDMPolicy:        NormalizeGatewayGroupDMPolicy(groupDMRaw),
		IgnoredChannels:      ParsePolicyChannelList(ignoredRaw),
		FreeResponseChannels: ParsePolicyChannelList(freeRaw),
		MentionKeywords:      ParseMentionKeywords(keywordRaw),
	}
	if len(p.MentionKeywords) == 0 {
		p.MentionKeywords = []string{"@agent", "@bot"}
	}
	return p, nil
}

func UpdateGatewayInteractionPolicy(workdir string, p GatewayInteractionPolicy) error {
	mentionVal := "false"
	if p.MentionRequired {
		mentionVal = "true"
	}
	if err := SetGatewaySetting(workdir, "policy_mention_required", mentionVal); err != nil {
		return err
	}
	if err := SetGatewaySetting(workdir, "policy_group_dm", NormalizeGatewayGroupDMPolicy(p.GroupDMPolicy)); err != nil {
		return err
	}
	if err := SetGatewaySetting(workdir, "policy_ignored_channels", strings.Join(ParsePolicyChannelList(strings.Join(p.IgnoredChannels, ",")), ",")); err != nil {
		return err
	}
	if err := SetGatewaySetting(workdir, "policy_free_response_channels", strings.Join(ParsePolicyChannelList(strings.Join(p.FreeResponseChannels, ",")), ",")); err != nil {
		return err
	}
	if err := SetGatewaySetting(workdir, "policy_mention_keywords", strings.Join(ParseMentionKeywords(strings.Join(p.MentionKeywords, ",")), ",")); err != nil {
		return err
	}
	_ = os.Setenv(GatewayPolicyMentionRequiredEnvVar, mentionVal)
	_ = os.Setenv(GatewayPolicyGroupDMEnvVar, NormalizeGatewayGroupDMPolicy(p.GroupDMPolicy))
	return nil
}

func ChannelPolicyKeys(platform, chatID string) []string {
	p := strings.ToLower(strings.TrimSpace(platform))
	c := strings.TrimSpace(chatID)
	if p == "" || c == "" {
		return nil
	}
	return []string{
		p + ":" + c,
		c,
	}
}

func MatchPolicyChannel(list []string, platform, chatID string) bool {
	if len(list) == 0 {
		return false
	}
	keys := ChannelPolicyKeys(platform, chatID)
	if len(keys) == 0 {
		return false
	}
	set := map[string]bool{}
	for _, it := range list {
		set[strings.ToLower(strings.TrimSpace(it))] = true
	}
	for _, k := range keys {
		if set[strings.ToLower(strings.TrimSpace(k))] {
			return true
		}
	}
	return false
}
