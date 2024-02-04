package sessions

import (
	"database/sql"
	"log"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/golang-tui/util"
)

type Model struct {
	textInput     textinput.Model
	list          list.Model
	isFocused     bool
	currentEditID int

	CurrentSessionID     int
	CurrentSessionName   string
	terminalWidth        int
	terminalHeight       int
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

type UpdateCurrentSession struct{}

func SendUpdateCurrentSessionMsg() tea.Cmd {
	return func() tea.Msg {
		return UpdateCurrentSession{}
	}
}

func (m Model) Init() tea.Cmd {
	// Need to load the latest session as the current session  (select recently created)
	// OR we need to create a brand new session for the user with a random name (insert new and return)
	session, _ := m.sessionService.GetLatestSession()
	y := []Session{}
	y = append(y, session)
	m.list = initEditListViewTable(y, m.CurrentSessionID)
	return func() tea.Msg {
		session, err := m.sessionService.GetLatestSession()
		if err != nil {
			// TODO: better error handling
			log.Println("error", err)
			panic(err)
		}

		// m.insertRandomSession()
		allSessions, err := m.sessionService.GetAllSessions()
		if err != nil {
			// TODO: better error handling
			log.Println("error", err)
			panic(err)
		}

		return LoadDataFromDB{
			session:     session,
			allSessions: allSessions,
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	dKeyPressed := false
	switch msg := msg.(type) {

	case LoadDataFromDB:
		m.CurrentSessionID = msg.session.ID
		m.CurrentSessionName = msg.session.SessionName
		m.ArrayOfMessages = msg.session.Messages
		m.AllSessions = msg.allSessions

		m.list = initEditListViewTable(m.AllSessions, m.CurrentSessionID)
		m.currentEditID = -1
		return m, cmd

	case util.FocusEvent:
		m.isFocused = msg.IsFocused
		m.currentEditID = -1
		return m, nil

	case ProcessResult:
		// add the latest message to the array of messages
		m.handleMsgProcessing(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		// m.list.SetHeight(msg.Height - 18)

	// TODO: clean this up and make it more neat
	case tea.KeyMsg:
		if m.isFocused {
			dKeyPressed = msg.String() == "d"
			if m.currentEditID != -1 {
				cmd = m.handleCurrentEditID(msg)
				cmds = append(cmds, cmd)
			} else {
				cmd := m.handleCurrentNormalMode(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	if m.isFocused && m.currentEditID == -1 && !dKeyPressed {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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

	return m.settingsContainer().Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader("Sessions"),
			listView,
			editForm,
		),
	)
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

// updates the current view with the messages coming in
func (m *Model) handleMsgProcessing(msg ProcessResult) {
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
				m.ArrayOfMessages = append(
					m.ArrayOfMessages,
					constructJsonMessage(m.ArrayOfProcessResult),
				)

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
}

func (m *Model) handleCurrentEditID(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if msg.String() == "enter" {
		if m.textInput.Value() != "" {
			m.sessionService.UpdateSessionName(m.currentEditID, m.textInput.Value())
			m.updateSessionList()
			m.currentEditID = -1
		}
	}
	return cmd
}

func (m *Model) handleCurrentNormalMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+n":
		m.sessionService.InsertNewSession("New Session", []MessageToSend{})
		m.updateSessionList()

	case "enter":
		i, ok := m.list.SelectedItem().(item)
		if ok {
			session, err := m.sessionService.GetSession(i.id)
			if err != nil {
				log.Println("error", err)
				panic(err)
			}

			m.CurrentSessionID = session.ID
			m.CurrentSessionName = session.SessionName
			m.ArrayOfMessages = session.Messages
			m.list.SetItems(ConstructListItems(m.AllSessions, m.CurrentSessionID))

			log.Println("enter taran", session)

			cmd = SendUpdateCurrentSessionMsg()
		}

	case "r":
		ti := textinput.New()
		ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(2)
		m.textInput = ti
		i, ok := m.list.SelectedItem().(item)
		if ok {
			m.currentEditID = i.id
			m.textInput.Placeholder = "New Session Name"
		}
		m.textInput.Focus()
		m.textInput.CharLimit = 100

	case "d":
		i, ok := m.list.SelectedItem().(item)
		if ok {
			// delete this one if it's not the active one
			if i.id != m.CurrentSessionID {
				m.sessionService.DeleteSession(i.id)
				m.updateSessionList()
			}
		}

	}

	return cmd
}

func (m *Model) updateSessionList() {
	m.AllSessions, _ = m.sessionService.GetAllSessions()
	items := []list.Item{}

	for _, session := range m.AllSessions {
		anItem := item{
			id:       session.ID,
			text:     session.SessionName,
			isActive: session.ID == m.CurrentSessionID,
		}
		items = append(items, anItem)
	}
	m.list.SetItems(items)
}
