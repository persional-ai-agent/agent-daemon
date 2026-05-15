package tools

import "testing"

func TestSetAndGetHomeTarget(t *testing.T) {
	workdir := t.TempDir()
	if err := SetHomeTarget(workdir, "telegram", "100"); err != nil {
		t.Fatal(err)
	}
	got, err := GetHomeTarget(workdir, "telegram")
	if err != nil {
		t.Fatal(err)
	}
	if got != "100" {
		t.Fatalf("home target=%q want=100", got)
	}
}

func TestResolveHomeTargetPrefersEnv(t *testing.T) {
	workdir := t.TempDir()
	if err := SetHomeTarget(workdir, "telegram", "100"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TELEGRAM_HOME_CHANNEL", "200")
	if got := ResolveHomeTarget(workdir, "telegram"); got != "200" {
		t.Fatalf("resolved=%q want=200", got)
	}
}
