package platform

import "context"

type MessageEvent struct {
	Text      string   `json:"text"`
	MessageID string   `json:"message_id"`
	ChatID    string   `json:"chat_id"`
	ChatType  string   `json:"chat_type"`
	UserID    string   `json:"user_id"`
	UserName  string   `json:"user_name"`
	MediaURLs []string `json:"media_urls,omitempty"`
	ReplyToID string   `json:"reply_to_id,omitempty"`
	ThreadID  string   `json:"thread_id,omitempty"`
	IsCommand bool     `json:"is_command"`
}

type SendResult struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type MessageHandler func(ctx context.Context, event MessageEvent)

type Adapter interface {
	Name() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Send(ctx context.Context, chatID, content, replyTo string) (SendResult, error)
	EditMessage(ctx context.Context, chatID, messageID, content string) error
	SendTyping(ctx context.Context, chatID string) error
	OnMessage(ctx context.Context, handler MessageHandler)
}

// RichTextSender is an optional extension for adapters that can attach
// platform-native metadata (for example Telegram inline approval buttons).
type RichTextSender interface {
	SendText(ctx context.Context, chatID, content, replyTo string, meta map[string]any) (SendResult, error)
}

// MediaSender is an optional extension for gateway adapters.
// If implemented, tools like send_message can deliver local files (e.g. TTS audio).
type MediaSender interface {
	SendMedia(ctx context.Context, chatID, path, caption, replyTo string) (SendResult, error)
}
