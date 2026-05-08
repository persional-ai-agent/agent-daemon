package gateway

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type Runner struct {
	adapters     []PlatformAdapter
	engine       *agent.Engine
	allowedUsers string
	mu           sync.Mutex
	wg           sync.WaitGroup
}

func NewRunner(adapters []PlatformAdapter, engine *agent.Engine, allowedUsers string) *Runner {
	return &Runner{
		adapters:     adapters,
		engine:       engine,
		allowedUsers: allowedUsers,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	for _, a := range r.adapters {
		adapter := a
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			adapterCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			adapter.OnMessage(adapterCtx, func(msgCtx context.Context, event MessageEvent) {
				r.handleMessage(msgCtx, adapter, event)
			})

			if err := adapter.Connect(adapterCtx); err != nil {
				log.Printf("[gateway:%s] connect failed: %v", adapter.Name(), err)
				return
			}

			<-adapterCtx.Done()
			if err := adapter.Disconnect(context.Background()); err != nil {
				log.Printf("[gateway:%s] disconnect error: %v", adapter.Name(), err)
			}
		}()
	}
	return nil
}

func (r *Runner) Stop() {
	r.wg.Wait()
}

func (r *Runner) handleMessage(ctx context.Context, adapter PlatformAdapter, event MessageEvent) {
	if !CheckAuthorization(r.allowedUsers, event.UserID) {
		_, _ = adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
		return
	}

	sessionKey := BuildSessionKey(adapter.Name(), event.ChatType, event.ChatID)

	history, err := r.engine.SessionStore.LoadMessages(sessionKey, 500)
	if err != nil {
		log.Printf("[gateway:%s] load history: %v", adapter.Name(), err)
		history = nil
	}

	collector := NewStreamCollector()
	streamMsgID := ""

	eng := *r.engine
	eng.EventSink = func(evt core.AgentEvent) {
		collector.Ingest(evt)
		if collector.ShouldEdit() {
			content := collector.Content()
			if content == "" {
				return
			}
			if streamMsgID == "" {
				_ = adapter.SendTyping(context.Background(), event.ChatID)
				result, sendErr := adapter.Send(context.Background(), event.ChatID, escapeMarkdown(content), event.MessageID)
				if sendErr == nil {
					streamMsgID = result.MessageID
				}
			} else {
				_ = adapter.EditMessage(context.Background(), event.ChatID, streamMsgID, escapeMarkdown(content)+"…")
			}
		}
	}

	res, runErr := eng.Run(ctx, sessionKey, event.Text, agent.DefaultSystemPrompt(), history)

	finalContent := collector.Content()
	if finalContent == "" && res != nil {
		finalContent = res.FinalResponse
	}
	if finalContent == "" && runErr != nil {
		finalContent = "Error: " + runErr.Error()
	}

	if streamMsgID != "" {
		_ = adapter.EditMessage(context.Background(), event.ChatID, streamMsgID, escapeMarkdown(finalContent))
	} else {
		_, _ = adapter.Send(context.Background(), event.ChatID, escapeMarkdown(finalContent), event.MessageID)
	}
}

func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "~", "\\~")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, ">", "\\>")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "+", "\\+")
	s = strings.ReplaceAll(s, "-", "\\-")
	s = strings.ReplaceAll(s, "=", "\\=")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, ".", "\\.")
	s = strings.ReplaceAll(s, "!", "\\!")
	return s
}
