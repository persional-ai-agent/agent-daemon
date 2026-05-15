package gateway

import (
	"sort"
	"strings"
)

type gatewayCommandSpec struct {
	Name         string
	HelpUsage    string
	Aliases      []string
}

var gatewayCommandCatalog = []gatewayCommandSpec{
	{Name: "pair", HelpUsage: "/pair <code>"},
	{Name: "unpair"},
	{Name: "cancel", Aliases: []string{"abort", "stop"}},
	{Name: "queue", Aliases: []string{"q"}},
	{Name: "status", Aliases: []string{"s"}},
	{Name: "pending", Aliases: []string{"pendings"}},
	{Name: "approvals", Aliases: []string{"approval"}},
	{Name: "grant", HelpUsage: "/grant [ttl], /grant pattern <name> [ttl]"},
	{Name: "revoke", HelpUsage: "/revoke, /revoke pattern <name>"},
	{Name: "approve", HelpUsage: "/approve <id>"},
	{Name: "deny", HelpUsage: "/deny <id>"},
	{Name: "help", Aliases: []string{"h"}},
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

func GatewayApprovalSlashCommands() []string {
	out := make([]string, 0, len(gatewayApprovalCommandSet))
	for _, name := range gatewayHelpCommandOrder {
		if _, ok := gatewayApprovalCommandSet[name]; ok {
			out = append(out, "/"+name)
		}
	}
	return out
}

func GatewayHelpText(yuanbao bool) string {
	parts := make([]string, 0, len(gatewayHelpCommandOrder))
	for _, name := range gatewayHelpCommandOrder {
		parts = append(parts, gatewayHelpCommandEntry(name))
	}
	text := "Commands: " + strings.Join(parts, ", ")
	if yuanbao {
		text += "\nQuick reply aliases: 状态, 待审批, 审批, 批准, 拒绝, 帮助"
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
