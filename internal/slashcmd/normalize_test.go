package slashcmd

import "testing"

func TestNormalizeInputAliases(t *testing.T) {
	cases := map[string]string{
		"/Q":              "/quit",
		"gw":              "/gateway status",
		"/GW status":      "/gateway status",
		"/CFG get":        "/config get",
		"/SESS 10":        "/sessions 10",
		"/WB save demo":   "/workbench save demo",
		"/WF list":        "/workflow list",
		"/BM add demo":    "/bookmark add demo",
		"SHOW sid-1":      "/show sid-1",
		"SESSIONS 5":      "/sessions 5",
		"TOOL apply":      "/tool apply",
		"cfg get":         "/config get",
		"/CONFIG SET x y": "/config SET x y",
		"stop":            "/cancel",
		"/STOP":           "/cancel",
		"abort":           "/cancel",
		"/ABORT":          "/cancel",
	}
	for in, want := range cases {
		got := NormalizeInput(in)
		if got != want {
			t.Fatalf("input=%q got=%q want=%q", in, got, want)
		}
	}
}
