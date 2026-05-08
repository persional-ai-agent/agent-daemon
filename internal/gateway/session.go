package gateway

import "fmt"

type SessionSource struct {
	Platform string `json:"platform"`
	ChatID   string `json:"chat_id"`
	ChatType string `json:"chat_type"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	ThreadID string `json:"thread_id,omitempty"`
}

func BuildSessionKey(platform, chatType, chatID string) string {
	return fmt.Sprintf("agent:main:%s:%s:%s", platform, chatType, chatID)
}
