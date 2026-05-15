package gateway

import (
	"sort"
	"strings"
)

var builtInGatewayCommandSet = map[string]struct{}{
	"pair":      {},
	"unpair":    {},
	"cancel":    {},
	"queue":     {},
	"status":    {},
	"pending":   {},
	"approvals": {},
	"grant":     {},
	"revoke":    {},
	"approve":   {},
	"deny":      {},
	"help":      {},
}

var gatewayCommandAliasToCanonical = map[string]string{
	"approval": "approvals",
	"pendings": "pending",
	"abort":    "cancel",
	"stop":     "cancel",
	"q":        "queue",
	"s":        "status",
	"h":        "help",
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
	if mapped, exists := gatewayCommandAliasToCanonical[head]; exists {
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
