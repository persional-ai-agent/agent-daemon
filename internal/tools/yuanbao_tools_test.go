package tools

import (
	"strings"
	"testing"
)

func TestYuanbaoSchemasDocumentDefaults(t *testing.T) {
	search := ybSearchStickerParams()
	searchProps, _ := search["properties"].(map[string]any)
	limit, _ := searchProps["limit"].(map[string]any)
	if desc, _ := limit["description"].(string); !strings.Contains(desc, "default 10") {
		t.Fatalf("yb_search_sticker.limit description=%q", desc)
	}

	members := ybQueryGroupMembersParams()
	props, _ := members["properties"].(map[string]any)
	offset, _ := props["offset"].(map[string]any)
	mlimit, _ := props["limit"].(map[string]any)
	if desc, _ := offset["description"].(string); !strings.Contains(desc, "default 0") {
		t.Fatalf("yb_query_group_members.offset description=%q", desc)
	}
	if desc, _ := mlimit["description"].(string); !strings.Contains(desc, "default 200") {
		t.Fatalf("yb_query_group_members.limit description=%q", desc)
	}
}
