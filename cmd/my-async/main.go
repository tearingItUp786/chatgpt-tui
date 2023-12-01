package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	SessionID     string
	SessionName   string
	terminalWidth int
	sub           chan struct{}
}

func New() Model {
	return Model{
		SessionName: "yolo",
		sub:         make(chan struct{}),
	}
}

func listItem(heading string, value string) string {
	headingEl := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("#FFC0CB")).
		Render
	spanEl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fff")).
		Render

	return headingEl(" "+heading, spanEl(value))
}

func (m Model) View() string {
	width := (m.terminalWidth / 3) - 5
	list := lipgloss.NewStyle().
		AlignVertical(lipgloss.Top).
		Border(lipgloss.NormalBorder(), true).
		Height(8).
		Width(width)

	listHeader := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		MarginLeft(2).
		Render

	return list.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader("Session"),
			listItem("2323lkjsdfsd", m.SessionName),
		),
	)
}

//	func callChatGpt(sub chan ProcessResult) tea.Cmd {
//		return func() tea.Msg {
//			for {
//				time.Sleep(time.Millisecond * time.Duration(rand.Int63n(900)+100)) // nolint:gosec
//				sub <- ProcessResult{}
//			}
//		}
//	}
//
// // A command that waits for the activity on a channel.
//
//	func waitForActivity(sub chan ProcessResult) tea.Cmd {
//		return func() tea.Msg {
//			return ProcessResult(<-sub)
//		}
//	}
//
// A message used to indicate that activity has occurred. In the real world (for
// example, chat) this would contain actual data.
type responseMsg struct{}

// Simulate a process that sends events at an irregular interval in real time.
// In this case, we'll send events on the channel at a random interval between
// 100 to 1000 milliseconds. As a command, Bubble Tea will run this
// asynchronously.
func listenForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		for {
			time.Sleep(time.Millisecond * time.Duration(rand.Int63n(900)+100)) // nolint:gosec
			sub <- struct{}{}
		}
	}
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		return responseMsg(<-sub)
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		listenForActivity(m.sub), // generate activity
		waitForActivity(m.sub),   // wait for activity
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// Start CallChatGpt on Enter key
			m.SessionName = "session"
			log.Println("Enter: session")
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		return m, nil
	case responseMsg:
		m.SessionName = "Response"
		return m, waitForActivity(m.sub) // wait for next event
	}
	return m, nil
}

func main() {
	p := tea.NewProgram(New())

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
