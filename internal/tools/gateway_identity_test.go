package tools

import (
	"strings"
	"testing"
)

func TestNormalizeContinuityMode(t *testing.T) {
	if got := NormalizeContinuityMode("id"); got != "user_id" {
		t.Fatalf("got=%q", got)
	}
	if got := NormalizeContinuityMode("name"); got != "user_name" {
		t.Fatalf("got=%q", got)
	}
	if got := NormalizeContinuityMode("xxx"); got != "off" {
		t.Fatalf("got=%q", got)
	}
}

func TestGatewayIdentityAndResolveSession(t *testing.T) {
	workdir := t.TempDir()
	if err := SetGatewaySetting(workdir, "continuity_mode", "user_name"); err != nil {
		t.Fatal(err)
	}
	if err := UpsertGatewayIdentity(workdir, "telegram", "u1", "gid-1"); err != nil {
		t.Fatal(err)
	}
	globalID, err := ResolveGatewayIdentity(workdir, "telegram", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if globalID != "gid-1" {
		t.Fatalf("globalID=%q", globalID)
	}
	resolved, err := ResolveGatewaySessionMapping(workdir, "telegram", "group", "1001", "u1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GlobalID != "gid-1" || resolved.GlobalSource != "mapped" {
		t.Fatalf("unexpected resolve: %+v", resolved)
	}
	if err := DeleteGatewayIdentity(workdir, "telegram", "u1"); err != nil {
		t.Fatal(err)
	}
	resolved, err = ResolveGatewaySessionMapping(workdir, "telegram", "group", "1001", "u1", "Alice Bob")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GlobalID != "uname:alice_bob" || resolved.GlobalSource != "auto" {
		t.Fatalf("unexpected auto resolve: %+v", resolved)
	}
}

func TestParseGatewayResolveArgs(t *testing.T) {
	got, err := ParseGatewayResolveArgs([]string{"Telegram", "group", "1001", "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Platform != "telegram" || got.ChatType != "group" || got.ChatID != "1001" || got.UserID != "u1" || got.UserName != "" {
		t.Fatalf("unexpected parsed args: %+v", got)
	}
	got, err = ParseGatewayResolveArgs([]string{"telegram", "group", "1001", "u1", "Alice"})
	if err != nil || got.UserName != "Alice" {
		t.Fatalf("unexpected parsed args with username: %+v err=%v", got, err)
	}
	if _, err := ParseGatewayResolveArgs([]string{"telegram", "group", "1001"}); err == nil {
		t.Fatal("expected invalid args error")
	}
}

func TestParseGatewayResolveArgsWithDefaults(t *testing.T) {
	def := GatewayResolveArgs{Platform: "discord", ChatType: "dm", ChatID: "c1", UserID: "u9", UserName: "Bob"}
	got, err := ParseGatewayResolveArgsWithDefaults(nil, def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Platform != "discord" || got.ChatType != "dm" || got.ChatID != "c1" || got.UserID != "u9" || got.UserName != "Bob" {
		t.Fatalf("unexpected defaults parse result: %+v", got)
	}
}

func TestParseGatewayIdentityArgs(t *testing.T) {
	ref, err := ParseGatewayIdentityRefArgs([]string{"Telegram", "u1"})
	if err != nil || ref.Platform != "telegram" || ref.UserID != "u1" {
		t.Fatalf("unexpected ref parse: %+v err=%v", ref, err)
	}
	setArgs, err := ParseGatewaySetIdentityArgs([]string{"Telegram", "u1", "gid-1"})
	if err != nil || setArgs.Platform != "telegram" || setArgs.UserID != "u1" || setArgs.GlobalID != "gid-1" {
		t.Fatalf("unexpected setid parse: %+v err=%v", setArgs, err)
	}
	globalID, err := ParseGatewayGlobalIDArg([]string{"gid-2"})
	if err != nil || globalID != "gid-2" {
		t.Fatalf("unexpected global id parse: %q err=%v", globalID, err)
	}
	if _, err := ParseGatewayIdentityRefArgs([]string{"telegram"}); err == nil {
		t.Fatal("expected invalid whoami/unsetid args error")
	}
	if _, err := ParseGatewaySetIdentityArgs([]string{"telegram", "u1"}); err == nil {
		t.Fatal("expected invalid setid args error")
	}
	if _, err := ParseGatewayGlobalIDArg([]string{}); err == nil {
		t.Fatal("expected invalid global id arg error")
	}
}

func TestParseGatewayContinuityModeArg(t *testing.T) {
	mode, err := ParseGatewayContinuityModeArg([]string{"name"})
	if err != nil || mode != "user_name" {
		t.Fatalf("unexpected continuity parse: mode=%q err=%v", mode, err)
	}
	mode, err = ParseGatewayContinuityModeArg([]string{"id"})
	if err != nil || mode != "user_id" {
		t.Fatalf("unexpected continuity parse: mode=%q err=%v", mode, err)
	}
	mode, err = ParseGatewayContinuityModeArg([]string{"off"})
	if err != nil || mode != "off" {
		t.Fatalf("unexpected continuity parse: mode=%q err=%v", mode, err)
	}
	if _, err := ParseGatewayContinuityModeArg([]string{}); err == nil {
		t.Fatal("expected invalid continuity arg length")
	}
}

func TestParseGatewayModelSpecArgs(t *testing.T) {
	got, err := ParseGatewayModelSpecArgs([]string{"openai:gpt-5"})
	if err != nil || got.Provider != "openai" || got.Model != "gpt-5" {
		t.Fatalf("unexpected one-arg model parse: %+v err=%v", got, err)
	}
	got, err = ParseGatewayModelSpecArgs([]string{"OpenAI", "gpt-5.1"})
	if err != nil || got.Provider != "openai" || got.Model != "gpt-5.1" {
		t.Fatalf("unexpected two-arg model parse: %+v err=%v", got, err)
	}
	if _, err := ParseGatewayModelSpecArgs([]string{"openai"}); err == nil {
		t.Fatal("expected invalid one-arg model parse")
	}
}

func TestGatewayWhoamiAndResolveHelpers(t *testing.T) {
	who := GatewayWhoamiResult{
		Platform:       "telegram",
		UserID:         "u1",
		UserName:       "Alice",
		ActiveSession:  "s1",
		GlobalID:       "gid-1",
		ContinuityMode: "user_id",
		AutoGlobalID:   "uid:u1",
	}
	whoPayload := BuildGatewayWhoamiPayload(who)
	if whoPayload["platform"] != "telegram" || whoPayload["global_id"] != "gid-1" || whoPayload["auto_global_id"] != "uid:u1" {
		t.Fatalf("unexpected whoami payload: %+v", whoPayload)
	}
	text := RenderGatewayWhoamiText(who)
	if !strings.Contains(text, "platform=telegram") || !strings.Contains(text, "global_id=gid-1") {
		t.Fatalf("unexpected whoami text: %q", text)
	}

	res := GatewaySessionResolveResult{
		Platform:        "telegram",
		ChatType:        "group",
		ChatID:          "1001",
		UserID:          "u1",
		UserName:        "Alice",
		RouteSession:    "agent:main:telegram:group:1001",
		MappedSession:   "agent:main:global:user:gid-1",
		ResolvedSession: "agent:main:global:user:gid-1",
		GlobalID:        "gid-1",
		GlobalSource:    "mapped",
		ContinuityMode:  "user_id",
	}
	payload := BuildGatewaySessionResolvePayload(res)
	if payload["resolved_session"] != "agent:main:global:user:gid-1" || payload["global_source"] != "mapped" {
		t.Fatalf("unexpected resolve payload: %+v", payload)
	}
	rt := RenderGatewaySessionResolveText(res)
	if !strings.Contains(rt, "resolved_session=agent:main:global:user:gid-1") {
		t.Fatalf("unexpected resolve text: %q", rt)
	}
}

func TestBuildGatewayIdentityPayload(t *testing.T) {
	got := BuildGatewayIdentityPayload("telegram", "u1", "gid-1", true, false)
	if got["platform"] != "telegram" || got["user_id"] != "u1" || got["global_id"] != "gid-1" || got["updated"] != true {
		t.Fatalf("unexpected identity payload: %+v", got)
	}
	got = BuildGatewayIdentityPayload("telegram", "u1", "", false, true)
	if got["deleted"] != true {
		t.Fatalf("expected deleted flag: %+v", got)
	}
	if _, ok := got["global_id"]; ok {
		t.Fatalf("unexpected global_id for delete payload: %+v", got)
	}
}

func TestBuildGatewayContinuityPayload(t *testing.T) {
	got := BuildGatewayContinuityPayload("name")
	if got["continuity_mode"] != "user_name" {
		t.Fatalf("unexpected continuity payload: %+v", got)
	}
}
