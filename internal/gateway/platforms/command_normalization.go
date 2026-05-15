package platforms

import (
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func normalizeInboundSlashText(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return text
	}
	return gateway.CanonicalizeGatewaySlashText(trimmed)
}

