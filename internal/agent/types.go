package agent

import "encoding/json"

type Message struct {
Role       string     `json:"role"`
Content    string     `json:"content,omitempty"`
Name       string     `json:"name,omitempty"`
ToolCallID string     `json:"tool_call_id,omitempty"`
ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
ID       string       `json:"id"`
Type     string       `json:"type"`
Function ToolFunction `json:"function"`
}

type ToolFunction struct {
Name      string `json:"name"`
Arguments string `json:"arguments"`
}

type ToolSchema struct {
Type     string           `json:"type"`
Function ToolSchemaDetail `json:"function"`
}

type ToolSchemaDetail struct {
Name        string         `json:"name"`
Description string         `json:"description"`
Parameters  map[string]any `json:"parameters"`
}

type RunResult struct {
SessionID       string    `json:"session_id"`
FinalResponse   string    `json:"final_response"`
Messages        []Message `json:"messages"`
TurnsUsed       int       `json:"turns_used"`
FinishedNaturally bool    `json:"finished_naturally"`
}

func CloneMessages(in []Message) []Message {
b, _ := json.Marshal(in)
out := make([]Message, 0)
_ = json.Unmarshal(b, &out)
return out
}
