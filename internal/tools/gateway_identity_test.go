package tools

import "testing"

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
