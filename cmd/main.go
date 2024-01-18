package main

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	textInput textinput.Model
	done      bool
	err       error
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type here..."
	ti.Focus()

	return model{
		textInput: ti,
		done:      false,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.done = false
			// log.Printf("You wrote: " + m.textInput.Value())
			return m, tea.Quit
		}

	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	tea.Printf("You wrote: " + m.textInput.Value())
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf("Hello \n %s", m.textInput.View())
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		log.Fatal(err)
	}
}
