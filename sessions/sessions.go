package sessions

import (
	"database/sql"
	"encoding/json"
	"log"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	CurrentSessionID     int
	CurrentSessionName   string
	terminalWidth        int
	ArrayOfProcessResult []ProcessResult
	ArrayOfMessages      []MessageToSend
	CurrentAnswer        string
	IsFocused            bool
	sessionService       *SessionService
	AllSessions          []Session
	table                table.Model // table for sessions edit view
	textInput            textinput.Model
	isEdittingRow        bool
}

func New(db *sql.DB) Model {
	ss := NewSessionService(db)

	return Model{
		ArrayOfProcessResult: []ProcessResult{},
		sessionService:       ss,
	}
}

type LoadDataFromDB struct {
	session     Session
	allSessions []Session
	listTable   table.Model
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

		allSessions, err := m.sessionService.GetAllSessions()
		if err != nil {
			// TODO: better error handling
			log.Println("error", err)
			panic(err)
		}

		log.Println("init")

		return LoadDataFromDB{
			session:     session,
			allSessions: allSessions,
			listTable:   initEditListViewTable(allSessions),
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case LoadDataFromDB:
		log.Println("LoadDataFromDB", msg.session.ID)
		m.CurrentSessionID = msg.session.ID
		m.CurrentSessionName = msg.session.SessionName
		m.ArrayOfMessages = msg.session.Messages
		m.AllSessions = msg.allSessions
		m.table = msg.listTable
		m.isEdittingRow = false
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

					m.sessionService.UpdateSessionMessages(m.CurrentSessionID, m.ArrayOfMessages)
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
		log.Println("window size", msg.Width)
		m.terminalWidth = msg.Width
		return m, nil

	case tea.KeyMsg:
		if m.isEdittingRow {
			m.textInput, cmd = m.textInput.Update(msg)
			if msg.String() == "enter" {
				m.isEdittingRow = false
				return m, cmd
			}
		} else {
			// Check if the 'r' key was pressed
			if msg.String() == "r" {
				log.Println("The 'r' key was pressed!")
				m.isEdittingRow = true
				ti := textinput.New()
				ti.Width = 0
				m.textInput = ti
				m.textInput.SetValue(m.table.SelectedRow()[1])
				m.textInput.Focus()
				m.textInput.CharLimit = 100
			}
		}

	}

	if m.IsFocused && !m.isEdittingRow {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	listView := m.normaListView()

	if m.IsFocused {
		listView = m.editListView()
	}

	editForm := ""
	if m.isEdittingRow {
		editForm = m.textInput.View()
	}
	return m.container().Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader("Sessions"),
			listView,
			editForm,
		),
	)
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

func (m Model) insertRandomSession() {
	newSession := Session{
		// Initialize your session fields as needed
		// ID will be set by the database if using auto-increment
		SessionName: "Random session",  // Set a default or generate a name
		Messages:    []MessageToSend{}, // Assuming Messages is a slice of Message
	}
	// Insert the new session into the database
	// Insert the new session into the database
	messagesJSON, err := json.Marshal(newSession.Messages)
	if err != nil {
		// TODO: better error handling
		log.Println("error", err)
		panic(err)
	}
	m.sessionService.InsertNewSession(newSession.SessionName, messagesJSON)
}
