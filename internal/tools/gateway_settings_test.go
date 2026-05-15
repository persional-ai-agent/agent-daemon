package tools

import "testing"

func TestGatewaySettingsSetGet(t *testing.T) {
	workdir := t.TempDir()
	if err := SetGatewaySetting(workdir, "continuity_mode", "user_name"); err != nil {
		t.Fatal(err)
	}
	got, err := GetGatewaySetting(workdir, "continuity_mode")
	if err != nil {
		t.Fatal(err)
	}
	if got != "user_name" {
		t.Fatalf("setting=%q want=user_name", got)
	}
}

func TestResolveGatewayContinuityMode(t *testing.T) {
	workdir := t.TempDir()
	if err := SetGatewaySetting(workdir, "continuity_mode", "user_name"); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveGatewayContinuityMode(workdir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "user_name" {
		t.Fatalf("mode=%q want=user_name", got)
	}
	t.Setenv(GatewayContinuityEnvVar, "id")
	got, err = ResolveGatewayContinuityMode(workdir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "user_id" {
		t.Fatalf("mode=%q want=user_id", got)
	}
}
