package tools

func BuildUIResultEnvelope(result any) map[string]any {
	return map[string]any{
		"ok":     true,
		"result": result,
	}
}

