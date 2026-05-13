package api

import (
	"os"
	"strings"
	"testing"
)

func TestContractSpecVersionSync(t *testing.T) {
	bs, err := os.ReadFile("../../docs/api/ui-chat-contract.openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi doc: %v", err)
	}
	doc := string(bs)
	if !strings.Contains(doc, `enum: [v1]`) {
		t.Fatalf("openapi missing api version enum v1")
	}
	if !strings.Contains(doc, `enum: ["`+uiCompat+`"]`) {
		t.Fatalf("openapi missing compat enum %s", uiCompat)
	}
}
