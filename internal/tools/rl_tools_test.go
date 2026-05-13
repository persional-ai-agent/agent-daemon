package tools

import (
	"strings"
	"testing"
)

func TestRLTestInferenceSchemaDocumentsTimeoutDefault(t *testing.T) {
	params := rlTestInferenceParams()
	props, _ := params["properties"].(map[string]any)
	timeout, _ := props["timeout"].(map[string]any)
	desc, _ := timeout["description"].(string)
	if !strings.Contains(desc, "default 180") {
		t.Fatalf("rl_test_inference.timeout description=%q, want default hint", desc)
	}
}
