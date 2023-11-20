package main

import (
	"log"
	"tea/sessions"
	"tea/settings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fake enum to keep tab of the currently focused pane
const (
	settingsType = iota
	sessionsType
	promptType
)

type model struct {
	focused          int
	promptContainer  lipgloss.Style
	promptInput      textinput.Model
	settingsModel    settings.Model
	sessionModel     sessions.Model
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
		promptContainer: lipgloss.NewStyle().
			AlignVertical(lipgloss.Bottom).
			BorderStyle(lipgloss.NormalBorder()).
			MarginTop(1).
			Padding(1),
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
			return m, nil
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.promptContainer = m.promptContainer.Copy().MaxWidth(m.terminalWidth).
			Width(m.terminalWidth - 3)

		yolo, cmd := m.settingsModel.Update(msg)
		another, cmd := m.sessionModel.Update(msg)
		m.settingsModel = yolo
		m.sessionModel = another
		return m, cmd
	}

	log.Printf("You wrote: %v", msg)
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
					Height(m.terminalHeight-m.promptContainer.GetHeight()-7).
					Width(m.terminalWidth/3*2).
					Render("FUCK"),
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
