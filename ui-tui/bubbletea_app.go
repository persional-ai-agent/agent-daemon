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
	lines []string
	err   error
}

type tuiModel struct {
	state      *appState
	input      textinput.Model
	viewport   viewport.Model
	lines      []string
	width      int
	height     int
	processing bool
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
		if msg.err != nil {
			m.appendLine(fmt.Sprintf("error: %v", msg.err))
			m.state.setErrStatus(msg.err)
		} else {
			for _, ln := range msg.lines {
				if strings.TrimSpace(ln) != "" {
					m.appendLine(ln)
				}
			}
			m.state.setStatus(true, "ok", "turn completed")
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
			m.appendLine("› " + raw)
			m.processing = true
			return m, m.runCommand(raw)
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

func (m *tuiModel) runCommand(text string) tea.Cmd {
	return func() tea.Msg {
		lines, err, quit := handleTUICommand(m.state, text, func(evt map[string]any) {
			m.state.addEvent(evt)
		})
		if quit {
			return tea.QuitMsg{}
		}
		if text == "/clear" {
			m.lines = nil
			m.syncViewport()
		}
		return turnDoneMsg{lines: lines, err: err}
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

func useBubbleTea() bool {
	v := strings.ToLower(strings.TrimSpace(getenvOr("AGENT_UI_TUI_BUBBLETEA", "true")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
