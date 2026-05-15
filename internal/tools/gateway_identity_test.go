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
