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

func TestResolveAndUpdateGatewayModelPreference(t *testing.T) {
	workdir := t.TempDir()
	if err := SetGatewaySetting(workdir, "model_provider", "openai"); err != nil {
		t.Fatal(err)
	}
	if err := SetGatewaySetting(workdir, "model_name", "gpt-5"); err != nil {
		t.Fatal(err)
	}
	pref, err := ResolveGatewayModelPreference(workdir)
	if err != nil {
		t.Fatal(err)
	}
	if pref.Provider != "openai" || pref.Model != "gpt-5" {
		t.Fatalf("unexpected pref: %+v", pref)
	}
	if err := UpdateGatewayModelPreference(workdir, GatewayModelSpec{Provider: "codex", Model: "gpt-5-codex"}); err != nil {
		t.Fatal(err)
	}
	pref, err = ResolveGatewayModelPreference(workdir)
	if err != nil {
		t.Fatal(err)
	}
	if pref.Provider != "codex" || pref.Model != "gpt-5-codex" {
		t.Fatalf("unexpected updated pref: %+v", pref)
	}
}

func TestDisplayGatewayModelPreference(t *testing.T) {
	got := DisplayGatewayModelPreference(GatewayModelPreference{})
	if got.Provider != "openai" || got.Model != "(default)" || got.BaseURL != "(default)" {
		t.Fatalf("unexpected default display pref: %+v", got)
	}
}
