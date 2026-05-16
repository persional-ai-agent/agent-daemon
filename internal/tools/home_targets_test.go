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

func TestParseSetHomeArgs(t *testing.T) {
	p, c, err := ParseSetHomeArgs([]string{"telegram:100"})
	if err != nil || p != "telegram" || c != "100" {
		t.Fatalf("single token parse mismatch: p=%q c=%q err=%v", p, c, err)
	}
	p, c, err = ParseSetHomeArgs([]string{"Telegram", " 200 "})
	if err != nil || p != "telegram" || c != "200" {
		t.Fatalf("double token parse mismatch: p=%q c=%q err=%v", p, c, err)
	}
	if _, _, err = ParseSetHomeArgs([]string{}); err == nil {
		t.Fatal("expected error for empty args")
	}
	if _, _, err = ParseSetHomeArgs([]string{"telegram"}); err == nil {
		t.Fatal("expected error for missing chat id")
	}
}

func TestBuildSetHomePayload(t *testing.T) {
	got := BuildSetHomePayload("Telegram", "100")
	if got["platform"] != "telegram" || got["chat_id"] != "100" || got["target"] != "telegram:100" || got["home_target"] != "100" || got["env"] != "TELEGRAM_HOME_CHANNEL" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}
