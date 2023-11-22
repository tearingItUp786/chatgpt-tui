package main

import (
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/tearingItUp786/golang-tui/sessions"
	"github.com/tearingItUp786/golang-tui/settings"
)

// fake enum to keep tab of the currently focused pane
const (
	settingsType = iota
	sessionsType
	promptType
)

type model struct {
	focused         int
	promptContainer lipgloss.Style
	viewport        viewport.Model
	promptInput     textinput.Model
	settingsModel   settings.Model
	sessionModel    sessions.Model
	msgChan         chan tea.Msg

	currentSessionID string
	terminalWidth    int
	terminalHeight   int
}

func initialModal() model {
	ti := textinput.New()
	ti.Placeholder = "Ask ChatGPT"
	ti.Focus()

	si := settings.New()
	sm := sessions.New()

	return model{
		focused:          sessionsType,
		promptInput:      ti,
		settingsModel:    si,
		currentSessionID: "",
		sessionModel:     sm,
		msgChan:          make(chan tea.Msg),
		promptContainer: lipgloss.NewStyle().
			AlignVertical(lipgloss.Bottom).
			BorderStyle(lipgloss.NormalBorder()).
			MarginTop(1),
	}
}

func (m model) Init() tea.Cmd {
	return m.promptInput.Cursor.BlinkCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			m.focused = (m.focused + 1) % 3
			m.viewport.SetContent(m.focusedPaneName())
			return m, cmd
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// Start CallChatGpt on Enter key
			return m, tea.Batch(
				sessions.CallChatGpt(),
			)
		}

	case sessions.ArrayProccessResult:
		log.Println("Got result from CallChatGpt")

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.promptContainer = m.promptContainer.Copy().MaxWidth(m.terminalWidth).
			Width(m.terminalWidth - 3)

		height := (m.terminalHeight - m.promptContainer.GetHeight() - 5)
		width := (m.terminalWidth / 3 * 2)
		m.viewport = viewport.New(width, height)
		content := "FUCK"
		m.viewport.SetContent(content)
		yolo, cmd := m.settingsModel.Update(msg)
		another, cmd := m.sessionModel.Update(msg)
		m.settingsModel = yolo
		m.sessionModel = another
		return m, cmd
	}

	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var val string
	settingsStuff := lipgloss.JoinVertical(
		lipgloss.Left,
		m.settingsModel.View(),
		m.sessionModel.View(),
	)
	val = lipgloss.NewStyle().
		Align(lipgloss.Right, lipgloss.Right).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					Height(m.terminalHeight-m.promptContainer.GetHeight()-5).
					Width(m.terminalWidth/3*2).
					// this is where we want to render all the messages
					Render(
						wordwrap.String(
							m.viewport.View(),
							50,
						)),
				settingsStuff,
			),
		)

	return lipgloss.NewStyle().Render(
		val,
		m.promptContainer.Render(
			m.promptInput.View(),
		),
	)
}

func (m model) focusedPaneName() string {
	if m.focused == sessionsType {
		return "SESSSION"
	}

	if m.focused == settingsType {
		return "SETTINGS"
	}

	return "PROMPT"
}

func (m model) renderMessages() string {
	return ""
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	p := tea.NewProgram(initialModal(), tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		log.Fatal(err)
	}
}
