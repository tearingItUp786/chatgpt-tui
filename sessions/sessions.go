package sessions

import (
	"database/sql"
	"encoding/json"
	"log"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/golang-tui/other"
)

type Model struct {
	textInput     textinput.Model
	list          list.Model
	isFocused     bool
	currentEditID int

	CurrentSessionID     int
	CurrentSessionName   string
	terminalWidth        int
	ArrayOfProcessResult []ProcessResult
	ArrayOfMessages      []MessageToSend
	CurrentAnswer        string
	sessionService       *SessionService
	AllSessions          []Session
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
	listTable   list.Model
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
			listTable:   initEditListViewTable(allSessions, session.ID),
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	log.Println("update", m.isFocused)

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case LoadDataFromDB:
		log.Println("LoadDataFromDB", msg.session.ID)
		m.CurrentSessionID = msg.session.ID
		m.CurrentSessionName = msg.session.SessionName
		m.ArrayOfMessages = msg.session.Messages
		m.AllSessions = msg.allSessions
		m.list = msg.listTable
		m.currentEditID = -1
		return m, cmd

	case other.FocusEvent:
		m.isFocused = msg.IsFocused
		m.currentEditID = -1
		return m, nil
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
		if m.isFocused {
			if m.currentEditID != -1 {
				m.textInput, cmd = m.textInput.Update(msg)
				if msg.String() == "enter" {
					if m.textInput.Value() != "" {
						m.sessionService.UpdateSessionName(m.currentEditID, m.textInput.Value())
						m.AllSessions, _ = m.sessionService.GetAllSessions()
						items := []list.Item{}

						for _, session := range m.AllSessions {
							anItem := item{
								id:   session.ID,
								text: session.SessionName,
							}
							items = append(items, anItem)
						}
						m.list.SetItems(items)
						m.currentEditID = -1
					}
				}
			} else {
				// Check if the 'r' key was pressed
				if msg.String() == "r" {
					log.Println("The 'r' key was pressed!")
					ti := textinput.New()
					ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(2)
					m.textInput = ti
					i, ok := m.list.SelectedItem().(item)
					if ok {
						m.currentEditID = i.id
						m.textInput.Placeholder = i.text
					}
					m.textInput.Focus()
					m.textInput.CharLimit = 100
				}
			}
		}

	}

	if m.isFocused && m.currentEditID == -1 {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	listView := m.normaListView()

	if m.isFocused {
		listView = m.editListView()
	}

	editForm := ""
	if m.currentEditID != -1 {
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

func RenderUserMessage(msg string, width int) string {
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
			messageToUse = RenderUserMessage(messageToUse, m.terminalWidth/3*2)
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
