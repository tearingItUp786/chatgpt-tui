package sessions

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/golang-tui/settings"
	"github.com/tearingItUp786/golang-tui/user"
	"github.com/tearingItUp786/golang-tui/util"
)

type Model struct {
	textInput      textinput.Model
	list           list.Model
	isFocused      bool
	currentEditID  int
	sessionService *SessionService
	userService    *user.UserService

	Settings             settings.Settings
	CurrentSessionID     int
	CurrentSessionName   string
	terminalWidth        int
	terminalHeight       int
	ArrayOfProcessResult []ProcessResult
	ArrayOfMessages      []MessageToSend
	CurrentAnswer        string
	AllSessions          []Session
}

func New(db *sql.DB) Model {
	ss := NewSessionService(db)
	us := user.NewUserService(db)

	// default --> get some default settings
	defaultSettings := settings.Settings{
		Model:     "gpt-3.5-turbo",
		MaxTokens: 300,
		Frequency: 0,
	}

	return Model{
		ArrayOfProcessResult: []ProcessResult{},
		sessionService:       ss,
		userService:          us,
		Settings:             defaultSettings,
	}
}

type LoadDataFromDB struct {
	session                Session
	allSessions            []Session
	listTable              list.Model
	currentActiveSessionID int
}

// Final Message is the concatenated string from the chat gpt stream
type FinalProcessMessage struct {
	FinalMessage string
}

func SendFinalProcessMessage(msg string) tea.Cmd {
	return func() tea.Msg {
		return FinalProcessMessage{
			FinalMessage: msg,
		}
	}
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
	return func() tea.Msg {
		mostRecentSession, err := m.sessionService.GetMostRecessionSessionOrCreateOne()
		if err != nil {
			return util.MakeErrorMsg(err.Error())
		}

		user, err := m.userService.GetUser(1)
		if err != nil {
			if err == sql.ErrNoRows {
				user, err = m.userService.InsertNewUser(mostRecentSession.ID)
			} else {
				return util.MakeErrorMsg(err.Error())
			}
		}

		mostRecentSession, err = m.sessionService.GetSession(user.CurrentActiveSessionID)
		if err != nil {
			return util.MakeErrorMsg(err.Error())
		}

		allSessions, err := m.sessionService.GetAllSessions()
		if err != nil {
			return util.MakeErrorMsg(err.Error())
		}

		return LoadDataFromDB{
			session:                mostRecentSession,
			allSessions:            allSessions,
			currentActiveSessionID: user.CurrentActiveSessionID,
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	dKeyPressed := false
	switch msg := msg.(type) {

	case LoadDataFromDB:
		m.CurrentSessionID = msg.currentActiveSessionID
		m.CurrentSessionName = msg.session.SessionName
		m.ArrayOfMessages = msg.session.Messages
		m.AllSessions = msg.allSessions

		m.list = initEditListViewTable(m.AllSessions, m.CurrentSessionID)
		m.currentEditID = -1
		return m, cmd

	case settings.UpdateSettingsEvent:
		m.Settings = msg.Settings
		return m, nil

	case util.FocusEvent:
		m.isFocused = msg.IsFocused
		m.currentEditID = -1
		return m, nil

	case ProcessResult:
		// add the latest message to the array of messages
		cmd = m.handleMsgProcessing(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

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
		Foreground(lipgloss.Color(util.Pink100)).
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
		BorderLeftForeground(lipgloss.Color(util.Indigo)).
		Width(width - 5).
		Render(
			"🤖 " + msg,
		)
}

func (m Model) GetMessagesAsString() string {
	var messages string
	for _, message := range m.ArrayOfMessages {
		messageToUse := message.Content

		switch {
		case message.Role == "user":
			messageToUse = RenderUserMessage(messageToUse, m.terminalWidth/3*2)
		case message.Role == "assistant":
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

// MIGHT BE WORTH TO MOVE TO A SEP FILE
func (m *Model) appendAndOrderProcessResults(msg ProcessResult) {
	m.ArrayOfProcessResult = append(m.ArrayOfProcessResult, msg)
	m.CurrentAnswer = ""

	// we need to sort on ID here because go routines are done in different threads
	// and the order in which our channel receives messages is not guaranteed.
	// TODO: look into a better way to insert (can I Insert in order)
	sort.Slice(m.ArrayOfProcessResult, func(i, j int) bool {
		return m.ArrayOfProcessResult[i].ID < m.ArrayOfProcessResult[j].ID
	})
}

func (m *Model) assertChoiceContentString(choice Choice) (string, tea.Cmd) {
	choiceContent, ok := choice.Delta["content"]

	if !ok {
		return "", m.resetStateAndCreateError("choice content not found")
	}
	choiceString, ok := choiceContent.(string)

	if !ok {
		return "", m.resetStateAndCreateError("choice string no good")
	}

	return choiceString, nil
}

func (m *Model) handleFinalChoiceMessage(choice Choice) tea.Cmd {
	// if the json for whatever reason is malformed, bail out
	jsonMessages, err := constructJsonMessage(m.ArrayOfProcessResult)
	if err != nil {
		return m.resetStateAndCreateError(err.Error())
	}

	m.ArrayOfMessages = append(
		m.ArrayOfMessages,
		jsonMessages,
	)

	/*
		Update the database session with the ArrayOfMessages
		And then reset the model that we use for the view to the default state
	*/
	err = m.sessionService.UpdateSessionMessages(m.CurrentSessionID, m.ArrayOfMessages)
	if err != nil {
		return m.resetStateAndCreateError(err.Error())
	}

	oldMessage := m.CurrentAnswer
	m.ArrayOfProcessResult = []ProcessResult{}
	m.CurrentAnswer = ""
	return SendFinalProcessMessage(oldMessage)
}

// updates the current view with the messages coming in
func (m *Model) handleMsgProcessing(msg ProcessResult) tea.Cmd {
	m.appendAndOrderProcessResults(msg)

	for _, msg := range m.ArrayOfProcessResult {
		if len(msg.Result.Choices) > 0 {
			choice := msg.Result.Choices[0]
			// Now you can safely use 'choice' since you've confirmed there's at least one element.
			// this is when we're done with the stream
			if choice.FinishReason == "stop" || msg.Final {
				return m.handleFinalChoiceMessage(choice)
			}

			// we need to keep appending content to our current answer in this case
			choiceString, errCmd := m.assertChoiceContentString(choice)
			if errCmd != nil {
				return errCmd
			}

			m.CurrentAnswer = m.CurrentAnswer + choiceString
		}
	}

	return nil
}

func (m *Model) resetStateAndCreateError(errMsg string) tea.Cmd {
	m.ArrayOfProcessResult = []ProcessResult{}
	m.CurrentAnswer = ""
	return util.MakeErrorMsg(errMsg)
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

		currentTime := time.Now()
		formattedTime := currentTime.Format(time.ANSIC)
		defaultSessionName := fmt.Sprintf("%s", formattedTime)
		m.sessionService.InsertNewSession(defaultSessionName, []MessageToSend{})
		m.updateSessionList()

	case "enter":
		i, ok := m.list.SelectedItem().(item)
		if ok {
			session, err := m.sessionService.GetSession(i.id)
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}

			m.userService.UpdateUserCurrentActiveSession(1, session.ID)

			m.CurrentSessionID = session.ID
			m.CurrentSessionName = session.SessionName
			m.ArrayOfMessages = session.Messages
			m.list.SetItems(ConstructListItems(m.AllSessions, m.CurrentSessionID))

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
