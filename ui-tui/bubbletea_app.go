package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type turnDoneMsg struct {
	err  error
	quit bool
}

type doctorDoneMsg struct {
	items []doctorItem
	ok    bool
}

type tuiModel struct {
	state      *appState
	inputValue string
	compBase   string
	compItems  []string
	compIndex  int
	viewport   viewport.Model
	width      int
	height     int
	processing bool
	turnStream chan tea.Msg
	runDoctor  bool

	runtime *terminalRuntime
}

const streamRenderInterval = 80 * time.Millisecond

func newTUIModel(state *appState, noDoctor bool) tuiModel {
	_ = noDoctor
	vp := viewport.New(80, 20)
	vp.SetContent("欢迎使用 ui-tui（Bubble Tea 模式）\n")
	runtime := newTerminalRuntime(vp.Width)
	runtime.addSystemText("欢迎使用 ui-tui（Bubble Tea 模式）")
	return tuiModel{
		state:     state,
		viewport:  vp,
		runtime:   runtime,
		runDoctor: false,
	}
}

func (m tuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.WindowSize()}
	if m.runDoctor {
		cmds = append(cmds, startDoctorCmd(m.state))
	}
	return tea.Batch(cmds...)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputH := 3
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - inputH - 4
		if m.viewport.Height < 5 {
			m.viewport.Height = 5
		}
		m.runtime.setWidth(m.viewport.Width)
		m.syncViewport(true)
		return m, nil
	case turnDoneMsg:
		m.processing = false
		m.turnStream = nil
		m.runtime.endTurn()
		if msg.err != nil && !errors.Is(msg.err, errTurnCancelled) {
			m.runtime.addError(fmt.Sprintf("error: %v", msg.err))
			m.state.setErrStatus(msg.err)
		} else if errors.Is(msg.err, errTurnCancelled) {
			m.runtime.addSystemText("turn cancelled")
			m.state.setStatus(true, "cancelled", "turn cancelled")
		} else if msg.quit {
			return m, tea.Quit
		} else {
			m.state.setStatus(true, "ok", "turn completed")
		}
		m.syncViewport(false)
		return m, nil
	case doctorDoneMsg:
		if msg.ok {
			m.state.setStatus(true, "ok", "startup doctor passed")
		} else {
			m.state.setStatus(false, "doctor_failed", "startup doctor found failures")
		}
		return m, nil
	case turnLineMsg:
		if strings.TrimSpace(msg.line) != "" {
			m.runtime.publishLine(msg.line)
			m.runtime.consumePendingEvents()
		}
		m.syncViewport(false)
		if m.processing && m.turnStream != nil {
			return m, waitTurnStreamCmd(m.turnStream)
		}
		return m, nil
	case turnEventMsg:
		if msg.event != nil {
			m.runtime.publishTurnEvent(msg.event)
			m.runtime.consumePendingEvents()
			m.state.addEvent(msg.event)
		}
		m.syncViewport(false)
		if m.processing && m.turnStream != nil {
			return m, waitTurnStreamCmd(m.turnStream)
		}
		return m, nil
	case streamRenderTickMsg:
		if m.processing {
			m.runtime.consumePendingEvents()
			m.syncViewport(false)
			return m, streamRenderTickCmd(streamRenderInterval)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.processing {
				m.state.requestTurnCancel()
				m.runtime.addSystemText("cancelling current turn...")
				m.syncViewport(false)
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+t":
			expanded := m.runtime.toggleThinkingExpanded()
			if expanded {
				m.state.setStatus(true, "ok", "thinking expanded")
			} else {
				m.state.setStatus(true, "ok", "thinking collapsed")
			}
			m.syncViewport(true)
			return m, nil
		case "pgup":
			m.viewport.HalfViewUp()
			return m, nil
		case "pgdown":
			m.viewport.HalfViewDown()
			return m, nil
		case "up", "down":
			if m.processing {
				return m, nil
			}
		case "enter":
			if m.processing {
				return m, nil
			}
			raw := strings.TrimSpace(m.inputValue)
			m.inputValue = ""
			m.resetCompletion()
			if raw == "" {
				return m, nil
			}
			if raw == "/quit" || raw == "/exit" {
				return m, tea.Quit
			}
			if raw == "/clear" {
				m.runtime.resetContent("欢迎使用 ui-tui（Bubble Tea 模式）")
				m.syncViewport(true)
				return m, nil
			}

			m.runtime.startTurn(raw)
			m.processing = true
			m.turnStream = make(chan tea.Msg, 64)
			m.syncViewport(true)
			return m, tea.Batch(
				startTurnCmd(m.turnStream, m.state, raw),
				waitTurnStreamCmd(m.turnStream),
				streamRenderTickCmd(streamRenderInterval),
			)
		case "backspace", "ctrl+h":
			if !m.processing && m.inputValue != "" {
				runes := []rune(m.inputValue)
				m.inputValue = string(runes[:len(runes)-1])
				m.resetCompletion()
			}
			return m, nil
		case "tab":
			if m.processing {
				return m, nil
			}
			m.applyCompletion()
			return m, nil
		}
		if !m.processing && len(msg.Runes) > 0 {
			m.inputValue += string(msg.Runes)
			m.resetCompletion()
			return m, nil
		}
	}
	return m, nil
}

func (m tuiModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("session=%s status=%s/%s transport=%s", m.state.session, m.state.lastStatus, m.state.lastCode, m.state.activeTransport),
	)
	footer := "Enter 提交，PgUp/PgDn 翻页，Ctrl+T 折叠思考，Ctrl+C 退出"
	if m.processing {
		footer = "处理中...（Enter 提交，PgUp/PgDn 翻页，Ctrl+T 折叠思考，Ctrl+C 退出）"
	}
	inputLine := "› " + m.inputValue
	if strings.TrimSpace(m.inputValue) == "" {
		inputLine = "› 输入消息或命令（/help, /quit）"
	}
	if len(m.compItems) > 0 && m.compBase != "" {
		next := m.compItems[m.compIndex%len(m.compItems)]
		inputLine += "    [Tab补全: " + next + "]"
	}
	return title + "\n" + m.viewport.View() + "\n\n" + inputLine + "\n" + footer
}

func (m *tuiModel) resetCompletion() {
	m.compBase = ""
	m.compItems = nil
	m.compIndex = 0
}

func (m *tuiModel) applyCompletion() {
	base := strings.TrimSpace(m.inputValue)
	if base == "" || !strings.HasPrefix(base, "/") {
		m.resetCompletion()
		return
	}
	if m.compBase != base || len(m.compItems) == 0 {
		m.compBase = base
		m.compItems = slashCompletions(base)
		m.compIndex = 0
	}
	if len(m.compItems) == 0 {
		m.resetCompletion()
		return
	}
	m.inputValue = m.compItems[m.compIndex%len(m.compItems)]
	m.compIndex = (m.compIndex + 1) % len(m.compItems)
}

type turnLineMsg struct {
	line string
}

type turnEventMsg struct {
	event map[string]any
}

type streamRenderTickMsg struct{}

func startTurnCmd(stream chan tea.Msg, state *appState, text string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer close(stream)
			_, err, quit := handleTUICommand(state, text, func(evt map[string]any) {
				stream <- turnEventMsg{event: evt}
			}, func(line string) {
				stream <- turnLineMsg{line: line}
			})
			stream <- turnDoneMsg{err: err, quit: quit}
		}()
		return nil
	}
}

func waitTurnStreamCmd(stream chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return nil
		}
		return msg
	}
}

func streamRenderTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return streamRenderTickMsg{}
	})
}

func startDoctorCmd(state *appState) tea.Cmd {
	return func() tea.Msg {
		items, ok := state.runDoctor()
		return doctorDoneMsg{items: items, ok: ok}
	}
}

func (m *tuiModel) syncViewport(force bool) {
	content, changed := m.runtime.render(force)
	m.state.debugLogf("render", "force=%t changed=%t processing=%t content_len=%d", force, changed, m.processing, len(content))
	if !changed {
		return
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func runBubbleTeaUI(s *appState, noDoctor bool) error {
	termenv.SetDefaultOutput(termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor), termenv.WithTTY(true)))
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stdout, termenv.WithProfile(termenv.TrueColor), termenv.WithTTY(true)))
	lipgloss.SetColorProfile(termenv.TrueColor)
	model := newTUIModel(s, noDoctor)
	program := tea.NewProgram(model)
	_, err := program.Run()
	return err
}
