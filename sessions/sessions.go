package sessions

import (
	"database/sql"
	"fmt"
	"log"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	SessionID            int
	SessionName          string
	terminalWidth        int
	ArrayOfProcessResult []ProcessResult
	ArrayOfMessages      []MessageToSend
	CurrentAnswer        string
	IsFocused            bool
	sessionService       *SessionService
}

func New(db *sql.DB) Model {
	ss := NewSessionService(db)

	return Model{
		ArrayOfProcessResult: []ProcessResult{},
		sessionService:       ss,
	}
}

type LoadDataFromDB struct {
	session Session
}

func (m Model) Init() tea.Cmd {
	// Need to load the latest session as the current session  (select recently created)
	// OR we need to create a brand new session for the user with a random name (insert new and return)
	return func() tea.Msg {
		session, err := m.sessionService.GetLatestSession()
		if err != nil {
			// TODO: better error handling
			log.Println("error", err)
			panic(err)
		}

		log.Println("init")
		return LoadDataFromDB{
			session: session,
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case LoadDataFromDB:
		log.Println("LoadDataFromDB", msg.session.ID)
		m.SessionID = msg.session.ID
		m.SessionName = msg.session.SessionName
		m.ArrayOfMessages = msg.session.Messages
		return m, cmd

	case ProcessResult:
		// add the latest message to the array of messages
		m.ArrayOfProcessResult = append(m.ArrayOfProcessResult, msg)
		m.CurrentAnswer = ""

		// we need to sort on ID here because go routines are done in different threads
		// and the order in which our channel receives messages is not guaranteed.
		// TODO: look into a better way to insert (can I Insert in order)
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

					m.sessionService.UpdateSessionMessages(m.SessionID, m.ArrayOfMessages)
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
	borderColor := lipgloss.Color("#bbb")
	if m.IsFocused {
		borderColor = lipgloss.Color("#d70073")
	}

	list := lipgloss.NewStyle().
		AlignVertical(lipgloss.Top).
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(borderColor).
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
			listItem(fmt.Sprint(m.SessionID), m.SessionName),
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

// Converts the array of json messages into a single Message
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

func renderUserMessage(msg string, width int) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeft(true).
		Foreground(lipgloss.Color("12")).
		Width(width - 2).
		Render("💁 " + msg)
}

func RenderBotMessage(msg string, width int) string {
	if msg == "" {
		return ""
	}

	return lipgloss.NewStyle().
		Align(lipgloss.Left).
		BorderStyle(lipgloss.RoundedBorder()).
		Foreground(lipgloss.Color("#FAFAFA")).
		BorderLeft(true).
		BorderLeftForeground(lipgloss.Color("214")).
		Width(width - 5).
		Render(
			"🤖 " + msg,
		)
}

func (m Model) GetMessagesAsString() string {
	var messages string
	for _, message := range m.ArrayOfMessages {
		messageToUse := message.Content

		if message.Role == "user" {
			messageToUse = renderUserMessage(messageToUse, m.terminalWidth/3*2)
		}

		if message.Role == "assistant" {
			messageToUse = RenderBotMessage(messageToUse, m.terminalWidth/3*2)
		}

		if messages == "" {
			messages = messageToUse
			continue
		}
		messages = messages + "\n" + messageToUse
	}

	return messages
}
