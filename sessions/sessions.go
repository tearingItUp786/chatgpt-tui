package sessions

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	SessionID            string
	SessionName          string
	terminalWidth        int
	ArrayOfProcessResult []ProcessResult
	ArrayOfMessages      []MessageToSend
	CurrentAnswer        string
}

func New() Model {
	return Model{
		ArrayOfProcessResult: []ProcessResult{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case ProcessResult:
		m.ArrayOfProcessResult = append(m.ArrayOfProcessResult, msg)
		m.CurrentAnswer = ""
		sort.Slice(m.ArrayOfProcessResult, func(i, j int) bool {
			return m.ArrayOfProcessResult[i].ID < m.ArrayOfProcessResult[j].ID
		})
		for _, msg := range m.ArrayOfProcessResult {
			if len(msg.Result.Choices) > 0 {
				choice := msg.Result.Choices[0]
				// Now you can safely use 'choice' since you've confirmed there's at least one element.
				if choice.FinishReason == "stop" || msg.Final {
					// empty out the array bro
					m.ArrayOfMessages = append(m.ArrayOfMessages, constructJsonMessage(m.ArrayOfProcessResult))
					m.ArrayOfProcessResult = []ProcessResult{}
					break
				}
				choiceContent, ok := choice.Delta["content"]
				if !ok {
					// TODO: this should be an error
					continue
				}
				choiceString, ok := choiceContent.(string)
				if !ok {
					// TODO: this should be an error
					continue
				}
				m.CurrentAnswer = m.CurrentAnswer + choiceString
			}
		}

		return m, cmd
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		return m, nil
	}
	return m, nil
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
			listItem("2323lkjsdfsd", "Some Session"),
		),
	)
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

func constructJsonMessage(arrayOfProcessResult []ProcessResult) MessageToSend {
	newMessage := MessageToSend{Role: "assistant", Content: ""}
	for _, aMessage := range arrayOfProcessResult {
		if len(aMessage.Result.Choices) > 0 {
			choice := aMessage.Result.Choices[0]
			if choice.FinishReason == "stop" || aMessage.Final {
				break
			}

			newMessage.Content += choice.Delta["content"].(string)
		}
	}
	return newMessage
}

func (m Model) GetMessagesAsString() string {
	var messages string
	for _, message := range m.ArrayOfMessages {
		messages = messages + "\n" + message.Content
	}

	return messages
}
