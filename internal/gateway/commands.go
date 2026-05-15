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

func IsBuiltInGatewayCommand(name string) bool {
	_, ok := builtInGatewayCommandSet[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func BuiltInGatewaySlashCommands() []string {
	out := make([]string, 0, len(builtInGatewayCommandSet))
	for name := range builtInGatewayCommandSet {
		out = append(out, "/"+name)
	}
	sort.Strings(out)
	return out
}
