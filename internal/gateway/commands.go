package gateway

import (
	"sort"
	"strings"
)

type gatewayCommandSpec struct {
	Name         string
	Description  string
	HelpUsage    string
	Aliases      []string
}

var gatewayCommandCatalog = []gatewayCommandSpec{
	{Name: "pair", Description: "pair with gateway using a code", HelpUsage: "/pair <code>"},
	{Name: "unpair", Description: "remove current gateway pairing"},
	{Name: "session", Description: "show or switch active session", HelpUsage: "/session [session_id]"},
	{Name: "history", Description: "show recent messages in active session", HelpUsage: "/history [n]"},
	{Name: "show", Description: "show session messages with pagination", HelpUsage: "/show [session_id] [offset] [limit]"},
	{Name: "next", Description: "show next page based on last /show"},
	{Name: "prev", Description: "show previous page based on last /show"},
	{Name: "sessions", Description: "list recent sessions", HelpUsage: "/sessions [n]"},
	{Name: "pick", Description: "switch active session from last /sessions list", HelpUsage: "/pick <index>"},
	{Name: "stats", Description: "show session stats", HelpUsage: "/stats [session_id]"},
	{Name: "new", Description: "switch to a new active session", HelpUsage: "/new [session_id]"},
	{Name: "reset", Description: "reset active session context"},
	{Name: "resume", Description: "switch to an existing active session", HelpUsage: "/resume <session_id>"},
	{Name: "recover", Description: "recover context by switching session and replaying input", HelpUsage: "/recover context"},
	{Name: "retry", Description: "replay latest user input"},
	{Name: "undo", Description: "undo last turn by branching to a new active session"},
	{Name: "clear", Description: "clear active session context by switching to a new session"},
	{Name: "reload", Description: "reload active session message count from store"},
	{Name: "save", Description: "export active session messages to json", HelpUsage: "/save [path]"},
	{Name: "cancel", Description: "cancel the running task", Aliases: []string{"abort", "stop"}},
	{Name: "compress", Description: "compact current session context", HelpUsage: "/compress [tail_messages]"},
	{Name: "queue", Description: "show queued task count", Aliases: []string{"q"}},
	{Name: "status", Description: "show current session status", Aliases: []string{"s"}},
	{Name: "pending", Description: "show latest pending approval", Aliases: []string{"pendings"}},
	{Name: "approvals", Description: "show active approvals", Aliases: []string{"approval"}},
	{Name: "grant", Description: "grant session or pattern approval", HelpUsage: "/grant [ttl], /grant pattern <name> [ttl]"},
	{Name: "revoke", Description: "revoke session or pattern approval", HelpUsage: "/revoke, /revoke pattern <name>"},
	{Name: "approve", Description: "approve a pending approval id", HelpUsage: "/approve <id>"},
	{Name: "deny", Description: "deny a pending approval id", HelpUsage: "/deny <id>"},
	{Name: "help", Description: "show supported commands", Aliases: []string{"h"}},
}

var gatewayApprovalCommandSet = map[string]struct{}{
	"approve":   {},
	"deny":      {},
	"pending":   {},
	"approvals": {},
	"grant":     {},
	"revoke":    {},
	"status":    {},
	"help":      {},
}

var gatewayCommandAuthRequiredSet = map[string]struct{}{
	"cancel":    {},
	"new":       {},
	"reset":     {},
	"resume":    {},
	"session":   {},
	"history":   {},
	"show":      {},
	"next":      {},
	"prev":      {},
	"sessions":  {},
	"pick":      {},
	"stats":     {},
	"recover":   {},
	"retry":     {},
	"undo":      {},
	"clear":     {},
	"reload":    {},
	"save":      {},
	"compress":  {},
	"queue":     {},
	"status":    {},
	"approve":   {},
	"deny":      {},
	"approvals": {},
	"pending":   {},
	"grant":     {},
	"revoke":    {},
}

var yuanbaoQuickReplyAliasToCanonical = map[string]string{
	"批准": "/approve",
	"同意": "/approve",
	"通过": "/approve",
	"拒绝": "/deny",
	"驳回": "/deny",
	"状态": "/status",
	"待审批": "/pending",
	"审批": "/approvals",
	"帮助": "/help",
}

var (
	builtInGatewayCommandSet  map[string]struct{}
	gatewayAliasToCanonical   map[string]string
	gatewayCommandSpecByName  map[string]gatewayCommandSpec
	gatewayHelpCommandOrder   []string
)

func init() {
	builtInGatewayCommandSet = make(map[string]struct{}, len(gatewayCommandCatalog))
	gatewayAliasToCanonical = map[string]string{}
	gatewayCommandSpecByName = make(map[string]gatewayCommandSpec, len(gatewayCommandCatalog))
	gatewayHelpCommandOrder = make([]string, 0, len(gatewayCommandCatalog))

	for _, spec := range gatewayCommandCatalog {
		name := strings.ToLower(strings.TrimSpace(spec.Name))
		if name == "" {
			continue
		}
		spec.Name = name
		builtInGatewayCommandSet[name] = struct{}{}
		gatewayCommandSpecByName[name] = spec
		gatewayHelpCommandOrder = append(gatewayHelpCommandOrder, name)
		for _, alias := range spec.Aliases {
			alias = strings.ToLower(strings.TrimSpace(alias))
			if alias == "" || alias == name {
				continue
			}
			gatewayAliasToCanonical[alias] = name
		}
	}
}

func IsBuiltInGatewayCommand(name string) bool {
	_, ok := builtInGatewayCommandSet[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func ResolveGatewayCommand(name string) (canonical string, ok bool) {
	head := strings.ToLower(strings.TrimSpace(name))
	if head == "" {
		return "", false
	}
	if IsBuiltInGatewayCommand(head) {
		return head, true
	}
	if mapped, exists := gatewayAliasToCanonical[head]; exists {
		return mapped, true
	}
	return "", false
}

// CanonicalizeGatewaySlashText normalizes slash command head casing/aliases while preserving arguments.
// Unknown slash heads are lower-cased and retained.
func CanonicalizeGatewaySlashText(text string) string {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return ""
	}
	head := strings.TrimPrefix(parts[0], "/")
	if canonical, ok := ResolveGatewayCommand(head); ok {
		if len(parts) == 1 {
			return "/" + canonical
		}
		return "/" + canonical + " " + strings.Join(parts[1:], " ")
	}
	if len(parts) == 1 {
		return "/" + strings.ToLower(head)
	}
	return "/" + strings.ToLower(head) + " " + strings.Join(parts[1:], " ")
}

func BuiltInGatewaySlashCommands() []string {
	out := make([]string, 0, len(builtInGatewayCommandSet))
	for name := range builtInGatewayCommandSet {
		out = append(out, "/"+name)
	}
	sort.Strings(out)
	return out
}

func BuiltInGatewayCommandNames() []string {
	out := make([]string, 0, len(builtInGatewayCommandSet))
	for name := range builtInGatewayCommandSet {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func GatewayCommandOrder() []string {
	out := make([]string, 0, len(gatewayHelpCommandOrder))
	out = append(out, gatewayHelpCommandOrder...)
	return out
}

func GatewayCommandDescriptions() map[string]string {
	out := make(map[string]string, len(gatewayCommandSpecByName))
	for name, spec := range gatewayCommandSpecByName {
		out[name] = strings.TrimSpace(spec.Description)
	}
	return out
}

func GatewayCommandAliases() map[string]string {
	out := make(map[string]string, len(gatewayAliasToCanonical))
	for k, v := range gatewayAliasToCanonical {
		out[k] = v
	}
	return out
}

func GatewayCommandUsage(name string) string {
	return gatewayHelpCommandEntry(name)
}

func GatewayGrantPatternUsage() string {
	return "/grant pattern <name> [ttl]"
}

func GatewayRevokePatternUsage() string {
	return "/revoke pattern <name>"
}

func GatewayGrantPatternOrRevokePatternUsage() string {
	return "Usage: " + GatewayGrantPatternUsage() + " or " + GatewayRevokePatternUsage()
}

func GatewayGrantRevokeCombinedUsage() string {
	return "Usage: " + GatewayCommandUsage("grant") + ", " + GatewayCommandUsage("revoke")
}

func GatewayApprovalSlashCommands() []string {
	out := make([]string, 0, len(gatewayApprovalCommandSet))
	for _, name := range gatewayHelpCommandOrder {
		if _, ok := gatewayApprovalCommandSet[name]; ok {
			out = append(out, "/"+name)
		}
	}
	return out
}

func GatewayCommandsRequiringAuthorization() []string {
	out := make([]string, 0, len(gatewayCommandAuthRequiredSet))
	for _, name := range gatewayHelpCommandOrder {
		if _, ok := gatewayCommandAuthRequiredSet[name]; ok {
			out = append(out, "/"+name)
		}
	}
	return out
}

func GatewayCommandRequiresAuthorization(name string) bool {
	head := strings.ToLower(strings.TrimSpace(name))
	head = strings.TrimPrefix(head, "/")
	_, ok := gatewayCommandAuthRequiredSet[head]
	return ok
}

func ResolveYuanbaoQuickReplyCommand(head string) (slash string, ok bool) {
	head = strings.TrimSpace(head)
	if head == "" {
		return "", false
	}
	slash, ok = yuanbaoQuickReplyAliasToCanonical[head]
	return slash, ok
}

func YuanbaoQuickReplyAliasesText() string {
	order := []string{"状态", "待审批", "审批", "批准", "拒绝", "帮助"}
	return strings.Join(order, ", ")
}

func GatewayHelpText(yuanbao bool) string {
	parts := make([]string, 0, len(gatewayHelpCommandOrder))
	for _, name := range gatewayHelpCommandOrder {
		parts = append(parts, gatewayHelpCommandEntry(name))
	}
	text := "Commands: " + strings.Join(parts, ", ")
	if yuanbao {
		text += "\nQuick reply aliases: " + YuanbaoQuickReplyAliasesText()
	}
	return text
}

func gatewayHelpCommandEntry(name string) string {
	spec, ok := gatewayCommandSpecByName[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "/" + strings.ToLower(strings.TrimSpace(name))
	}
	if strings.TrimSpace(spec.HelpUsage) != "" {
		return strings.TrimSpace(spec.HelpUsage)
	}
	return "/" + spec.Name
}
