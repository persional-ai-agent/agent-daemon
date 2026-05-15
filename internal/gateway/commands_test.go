package gateway

import (
	"strings"
	"testing"
)

func TestIsBuiltInGatewayCommandCaseInsensitive(t *testing.T) {
	if !IsBuiltInGatewayCommand("APPROVE") {
		t.Fatal("expected APPROVE to be built-in")
	}
	if !IsBuiltInGatewayCommand(" status ") {
		t.Fatal("expected status to be built-in")
	}
	if IsBuiltInGatewayCommand("unknown") {
		t.Fatal("unknown should not be built-in")
	}
}

func TestBuiltInGatewaySlashCommandsContainsCoreSet(t *testing.T) {
	list := BuiltInGatewaySlashCommands()
	want := map[string]bool{
		"/pair": true, "/unpair": true, "/session": true, "/history": true, "/stats": true, "/new": true, "/reset": true, "/resume": true, "/recover": true, "/retry": true, "/undo": true, "/cancel": true, "/compress": true, "/queue": true,
		"/status": true, "/pending": true, "/approvals": true, "/grant": true,
		"/revoke": true, "/approve": true, "/deny": true, "/help": true,
	}
	for _, v := range list {
		delete(want, v)
	}
	if len(want) != 0 {
		t.Fatalf("missing built-ins: %+v", want)
	}
}

func TestResolveGatewayCommandAliases(t *testing.T) {
	cases := map[string]string{
		"approval": "approvals",
		"pendings": "pending",
		"abort":    "cancel",
		"stop":     "cancel",
		"q":        "queue",
		"s":        "status",
		"h":        "help",
		"approve":  "approve",
	}
	for in, want := range cases {
		got, ok := ResolveGatewayCommand(in)
		if !ok || got != want {
			t.Fatalf("resolve(%q)=(%q,%v) want=(%q,true)", in, got, ok, want)
		}
	}
	if got, ok := ResolveGatewayCommand("unknown"); ok || got != "" {
		t.Fatalf("resolve unknown got=(%q,%v)", got, ok)
	}
}

func TestCanonicalizeGatewaySlashText(t *testing.T) {
	cases := map[string]string{
		"/STATUS":          "/status",
		"/approval":        "/approvals",
		"/approve ap-1":    "/approve ap-1",
		"/APPROVE ap-1":    "/approve ap-1",
		"/unknown CMD":     "/unknown CMD",
		"   /HeLp   ":      "/help",
		"   /grant 3600  ": "/grant 3600",
	}
	for in, want := range cases {
		if got := CanonicalizeGatewaySlashText(in); got != want {
			t.Fatalf("input=%q got=%q want=%q", in, got, want)
		}
	}
}

func TestGatewayHelpTextIncludesBuiltIns(t *testing.T) {
	help := GatewayHelpText(false)
	if !strings.HasPrefix(help, "Commands: ") {
		t.Fatalf("unexpected help prefix: %q", help)
	}
	for _, name := range BuiltInGatewaySlashCommands() {
		if !strings.Contains(help, name) {
			t.Fatalf("help missing command %s: %q", name, help)
		}
	}
	if !strings.Contains(help, "/grant pattern <name> [ttl]") {
		t.Fatalf("help missing grant pattern usage: %q", help)
	}
	if !strings.Contains(help, "/revoke pattern <name>") {
		t.Fatalf("help missing revoke pattern usage: %q", help)
	}
}

func TestGatewayHelpTextYuanbaoAddsQuickReplies(t *testing.T) {
	help := GatewayHelpText(true)
	if !strings.Contains(help, "Quick reply aliases:") {
		t.Fatalf("expected quick reply aliases in help: %q", help)
	}
}

func TestGatewayCommandAliasesIntegrity(t *testing.T) {
	aliases := GatewayCommandAliases()
	if len(aliases) == 0 {
		t.Fatal("expected non-empty alias map")
	}
	for alias, canonical := range aliases {
		if alias == canonical {
			t.Fatalf("alias should not equal canonical: %q", alias)
		}
		if IsBuiltInGatewayCommand(alias) {
			t.Fatalf("alias should not shadow built-in: %q", alias)
		}
		if !IsBuiltInGatewayCommand(canonical) {
			t.Fatalf("alias points to non built-in command: %q -> %q", alias, canonical)
		}
	}
}

func TestGatewayCommandUsage(t *testing.T) {
	if got := GatewayCommandUsage("approve"); got != "/approve <id>" {
		t.Fatalf("unexpected approve usage: %q", got)
	}
	if got := GatewayCommandUsage("status"); got != "/status" {
		t.Fatalf("unexpected status usage: %q", got)
	}
}

func TestGatewayApprovalSlashCommands(t *testing.T) {
	got := GatewayApprovalSlashCommands()
	want := map[string]bool{
		"/approve": true, "/deny": true, "/pending": true, "/approvals": true,
		"/grant": true, "/revoke": true, "/status": true, "/help": true,
	}
	if len(got) != len(want) {
		t.Fatalf("approval command count mismatch: got=%d want=%d list=%v", len(got), len(want), got)
	}
	for _, item := range got {
		if !want[item] {
			t.Fatalf("unexpected approval command: %s", item)
		}
		delete(want, item)
	}
	if len(want) != 0 {
		t.Fatalf("missing approval commands: %+v", want)
	}
}

func TestResolveYuanbaoQuickReplyCommand(t *testing.T) {
	cases := map[string]string{
		"批准":  "/approve",
		"同意":  "/approve",
		"通过":  "/approve",
		"拒绝":  "/deny",
		"驳回":  "/deny",
		"状态":  "/status",
		"待审批": "/pending",
		"审批":  "/approvals",
		"帮助":  "/help",
	}
	for in, want := range cases {
		got, ok := ResolveYuanbaoQuickReplyCommand(in)
		if !ok || got != want {
			t.Fatalf("ResolveYuanbaoQuickReplyCommand(%q)=(%q,%v) want=(%q,true)", in, got, ok, want)
		}
	}
	if got, ok := ResolveYuanbaoQuickReplyCommand("未知"); ok || got != "" {
		t.Fatalf("unexpected unknown quick-reply mapping: (%q,%v)", got, ok)
	}
}

func TestYuanbaoQuickReplyAliasesText(t *testing.T) {
	got := YuanbaoQuickReplyAliasesText()
	want := "状态, 待审批, 审批, 批准, 拒绝, 帮助"
	if got != want {
		t.Fatalf("quick reply aliases text mismatch: got=%q want=%q", got, want)
	}
}

func TestGrantRevokeUsageHelpers(t *testing.T) {
	if got := GatewayGrantPatternUsage(); got != "/grant pattern <name> [ttl]" {
		t.Fatalf("unexpected grant pattern usage: %q", got)
	}
	if got := GatewayRevokePatternUsage(); got != "/revoke pattern <name>" {
		t.Fatalf("unexpected revoke pattern usage: %q", got)
	}
	if got := GatewayGrantRevokeCombinedUsage(); got != "Usage: /grant [ttl], /grant pattern <name> [ttl], /revoke, /revoke pattern <name>" {
		t.Fatalf("unexpected combined usage: %q", got)
	}
}

func TestGatewayCommandRequiresAuthorization(t *testing.T) {
	authRequired := GatewayCommandsRequiringAuthorization()
	if len(authRequired) == 0 {
		t.Fatal("expected non-empty auth-required command set")
	}
	want := map[string]bool{
		"/session": true, "/history": true, "/stats": true, "/new": true, "/reset": true, "/resume": true, "/recover": true, "/retry": true, "/undo": true, "/cancel": true, "/compress": true, "/queue": true, "/status": true, "/approve": true, "/deny": true,
		"/approvals": true, "/pending": true, "/grant": true, "/revoke": true,
	}
	if len(authRequired) != len(want) {
		t.Fatalf("auth-required command count mismatch: got=%d want=%d list=%v", len(authRequired), len(want), authRequired)
	}
	for _, cmd := range authRequired {
		if !want[cmd] {
			t.Fatalf("unexpected auth-required command: %s", cmd)
		}
		delete(want, cmd)
	}
	if len(want) != 0 {
		t.Fatalf("missing auth-required commands: %+v", want)
	}

	if GatewayCommandRequiresAuthorization("/pair") {
		t.Fatal("/pair should not require authorization")
	}
	if !GatewayCommandRequiresAuthorization("/approve") {
		t.Fatal("/approve should require authorization")
	}
	if !GatewayCommandRequiresAuthorization("status") {
		t.Fatal("status should require authorization")
	}
}

func TestGatewayCommandOrderAndDescriptions(t *testing.T) {
	order := GatewayCommandOrder()
	if len(order) == 0 {
		t.Fatal("empty command order")
	}
	seen := map[string]bool{}
	for _, name := range order {
		if seen[name] {
			t.Fatalf("duplicate command in order: %s", name)
		}
		seen[name] = true
	}

	descs := GatewayCommandDescriptions()
	for _, name := range order {
		if strings.TrimSpace(descs[name]) == "" {
			t.Fatalf("empty description for command: %s", name)
		}
	}
}
