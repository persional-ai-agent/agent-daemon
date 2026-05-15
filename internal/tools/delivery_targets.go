package tools

import (
	"sort"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

func BuildDeliveryTargets(workdir, filterPlatform string) (platforms []string, targets []map[string]any, err error) {
	filterPlatform = strings.ToLower(strings.TrimSpace(filterPlatform))
	names := platform.Names()
	sort.Strings(names)
	items := make([]map[string]any, 0, len(names))
	seen := map[string]bool{}

	for _, name := range names {
		if filterPlatform != "" && name != filterPlatform {
			continue
		}
		home := ResolveHomeTarget(workdir, name)
		target := name
		if strings.TrimSpace(home) != "" {
			target = name + ":" + home
		}
		items = append(items, map[string]any{
			"platform":    name,
			"connected":   true,
			"home_target": home,
			"target":      target,
		})
		seen[name+":"] = true
	}

	if strings.TrimSpace(workdir) != "" {
		rows, listErr := ListChannelDirectory(workdir)
		if listErr != nil {
			return nil, nil, listErr
		}
		for _, row := range rows {
			if filterPlatform != "" && row.Platform != filterPlatform {
				continue
			}
			key := row.Platform + ":" + row.ChatID
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, map[string]any{
				"platform":     row.Platform,
				"chat_id":      row.ChatID,
				"chat_type":    row.ChatType,
				"user_id":      row.UserID,
				"user_name":    row.UserName,
				"global_id":    row.GlobalID,
				"home_target":  row.HomeTarget,
				"target":       row.Platform + ":" + row.ChatID,
				"connected":    false,
				"last_seen_at": row.LastSeenAt,
			})
		}
	}

	outPlatforms := names
	if filterPlatform != "" {
		outPlatforms = []string{}
		for _, name := range names {
			if name == filterPlatform {
				outPlatforms = append(outPlatforms, name)
			}
		}
	}
	return outPlatforms, items, nil
}
