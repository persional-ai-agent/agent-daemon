package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type turnDoneMsg struct {
	err   error
	quit  bool
}

type tuiModel struct {
	state      *appState
	input      textinput.Model
	viewport   viewport.Model
	lines      []string
	width      int
	height     int
	processing bool
	turnStream chan tea.Msg
}

func newTUIModel(state *appState) tuiModel {
	in := textinput.New()
	in.Placeholder = "输入消息或命令（/help, /quit）"
	in.Focus()
	in.CharLimit = 0
	in.Prompt = "› "

	vp := viewport.New(80, 20)
	vp.SetContent("欢迎使用 ui-tui（Bubble Tea 模式）\n")
	return tuiModel{
		state:    state,
		input:    in,
		viewport: vp,
		lines:    []string{"欢迎使用 ui-tui（Bubble Tea 模式）"},
	}
}

func (m tuiModel) Init() tea.Cmd { return textinput.Blink }

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
		m.syncViewport()
		return m, nil
	case turnDoneMsg:
		m.processing = false
		m.turnStream = nil
		if msg.err != nil {
			m.appendLine(fmt.Sprintf("error: %v", msg.err))
			m.state.setErrStatus(msg.err)
		} else if msg.quit {
			return m, tea.Quit
		} else {
			m.state.setStatus(true, "ok", "turn completed")
		}
		return m, nil
	case turnLineMsg:
		if strings.TrimSpace(msg.line) != "" {
			m.appendLine(msg.line)
		}
		if m.processing && m.turnStream != nil {
			return m, waitTurnStreamCmd(m.turnStream)
		}
		return m, nil
	case turnEventMsg:
		if msg.event != nil {
			m.state.addEvent(msg.event)
		}
		if m.processing && m.turnStream != nil {
			return m, waitTurnStreamCmd(m.turnStream)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "pgup":
			m.viewport.HalfViewUp()
			return m, nil
		case "pgdown":
			m.viewport.HalfViewDown()
			return m, nil
		case "up":
			if m.processing {
				return m, nil
			}
		case "down":
			if m.processing {
				return m, nil
			}
		case "enter":
			if m.processing {
				return m, nil
			}
			raw := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if raw == "" {
				return m, nil
			}
			if raw == "/quit" || raw == "/exit" {
				return m, tea.Quit
			}
			if raw == "/clear" {
				m.lines = nil
				m.syncViewport()
			}
			m.appendLine("› " + raw)
			m.processing = true
			m.turnStream = make(chan tea.Msg, 64)
			return m, tea.Batch(startTurnCmd(m.turnStream, m.state, raw), waitTurnStreamCmd(m.turnStream))
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("session=%s status=%s/%s transport=%s", m.state.session, m.state.lastStatus, m.state.lastCode, m.state.activeTransport),
	)
	footer := ""
	if m.processing {
		footer = "处理中...（Enter 提交，PgUp/PgDn 翻页，Ctrl+C 退出）"
	} else {
		footer = "Enter 提交，PgUp/PgDn 翻页，Ctrl+C 退出"
	}
	return title + "\n" + m.viewport.View() + "\n\n" + m.input.View() + "\n" + footer
}

type turnLineMsg struct {
	line string
}

type turnEventMsg struct {
	event map[string]any
}

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
			return turnDoneMsg{}
		}
		return msg
	}
}

func (m *tuiModel) appendLine(line string) {
	m.lines = append(m.lines, line)
	if len(m.lines) > m.state.chatMaxLines && m.state.chatMaxLines > 0 {
		m.lines = m.lines[len(m.lines)-m.state.chatMaxLines:]
	}
	m.syncViewport()
}

func (m *tuiModel) syncViewport() {
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	m.viewport.GotoBottom()
}

func runBubbleTeaUI(s *appState, noDoctor bool) error {
	if s.autoDoctor && !noDoctor {
		_, _ = s.runDoctor()
	}
	model := newTUIModel(s)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}
