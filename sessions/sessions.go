package sessions

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/settings"
	"github.com/tearingItUp786/chatgpt-tui/user"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const (
	IDLE       = "idle"
	PROCESSING = "processing"
	ERROR      = "error"
)

type Model struct {
	textInput         textinput.Model
	list              list.Model
	isFocused         bool
	currentEditID     int
	sessionService    *SessionService
	userService       *user.UserService
	settingsContainer lipgloss.Style

	Settings             settings.Settings
	CurrentSessionID     int
	CurrentSessionName   string
	terminalWidth        int
	terminalHeight       int
	ArrayOfProcessResult []ProcessResult
	ArrayOfMessages      []MessageToSend
	CurrentAnswer        string
	AllSessions          []Session
	ProcessingMode       string
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
		ProcessingMode:       IDLE,
		settingsContainer: lipgloss.NewStyle().
			AlignVertical(lipgloss.Top).
			Border(lipgloss.ThickBorder(), true).
			BorderForeground(util.NormalTabBorderColor),
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

	case util.OurWindowResize:
		width := m.terminalWidth - msg.Width - 5
		m.settingsContainer = m.settingsContainer.Width(width)

	case ProcessResult:
		// add the latest message to the array of messages
		log.Println("Processing message: ")
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
	listView := m.normalListView()

	if m.isFocused {
		listView = m.editListView()
	}

	editForm := ""
	if m.currentEditID != -1 {
		editForm = m.textInput.View()
	}

	borderColor := util.NormalTabBorderColor

	if m.isFocused {
		borderColor = util.ActiveTabBorderColor
	}

	return m.settingsContainer.BorderForeground(borderColor).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader("Sessions"),
			listView,
			editForm,
		),
	)
}

func RenderUserMessage(msg string, width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(util.Pink100)).
		Render("💁 " + msg)
}

func RenderBotMessage(msg string, width int) string {
	if msg == "" {
		return ""
	}

	lol, _ := glamour.RenderWithEnvironmentConfig(msg)
	output := strings.TrimSpace(lol)
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeftForeground(lipgloss.Color(util.Pink300)).
		Render(output)

	// return lipgloss.NewStyle().
	// 	Align(lipgloss.Left).
	// 	BorderStyle(lipgloss.RoundedBorder()).
	// 	Foreground(lipgloss.Color("#FAFAFA")).
	// 	BorderLeft(true).
	// 	BorderLeftForeground(lipgloss.Color(util.Pink300)).
	// 	Width(width - 5).
	// 	Render(
	// 		"🤖 " + msg,
	// 	)
}

func (m Model) GetLatestBotMessage() (string, error) {
	// the last bot in the array is actually the blank message (the stop command)
	lastIndex := len(m.ArrayOfMessages) - 2
	// Check if lastIndex is within the bounds of the slice
	if lastIndex >= 0 && lastIndex < len(m.ArrayOfMessages) {
		return m.ArrayOfMessages[lastIndex].Content, nil
	}
	return "", fmt.Errorf(
		"No messages found in array of messages. Length: %v",
		len(m.ArrayOfMessages),
	)
}

func (m Model) GetMessagesAsString() string {
	var messages string
	for _, message := range m.ArrayOfMessages {
		messageToUse := message.Content

		if messages == "" {
			messages = messageToUse
			continue
		}

		messages = messages + "\n" + messageToUse
	}

	return messages
}

func (m Model) GetMessagesAsPrettyString() string {
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
	// log.Println("Appending and ordering process results", msg)
	m.ArrayOfProcessResult = append(m.ArrayOfProcessResult, msg)
	m.CurrentAnswer = ""

	// we need to sort on ID here because go routines are done in different threads
	// and the order in which our channel receives messages is not guaranteed.
	// TODO: look into a better way to insert (can I Insert in order)
	sort.SliceStable(m.ArrayOfProcessResult, func(i, j int) bool {
		return m.ArrayOfProcessResult[i].ID < m.ArrayOfProcessResult[j].ID
	})
}

func (m *Model) assertChoiceContentString(choice Choice) (string, tea.Cmd) {
	choiceContent, ok := choice.Delta["content"]

	if !ok {
		if choice.FinishReason == "stop" || choice.FinishReason == "length" {

			areIdsAllThere := areIDsInOrderAndComplete(getArrayOfIDs(m.ArrayOfProcessResult))
			var cmd tea.Cmd
			if areIdsAllThere && m.ProcessingMode == PROCESSING {
				cmd = m.handleFinalChoiceMessage()
			}
			return "", cmd
		}
		return "", m.resetStateAndCreateError("choice content not found")
	}
	choiceString, ok := choiceContent.(string)

	if !ok {
		return "", m.resetStateAndCreateError("choice string no good")
	}

	return choiceString, nil
}

func (m *Model) handleFinalChoiceMessage() tea.Cmd {
	// if the json for whatever reason is malformed, bail out
	jsonMessages, err := constructJsonMessage(m.ArrayOfProcessResult)

	m.ArrayOfMessages = append(
		m.ArrayOfMessages,
		jsonMessages,
	)

	/*
		Update the database session with the ArrayOfMessages
		And then reset the model that we use for the view to the default state
	*/
	err = m.sessionService.UpdateSessionMessages(m.CurrentSessionID, m.ArrayOfMessages)
	m.ProcessingMode = IDLE
	m.CurrentAnswer = ""
	m.ArrayOfProcessResult = []ProcessResult{}

	if err != nil {
		log.Println("Error updating session messages", err)
		return m.resetStateAndCreateError(err.Error())
	}

	return nil
}

func areIDsInOrderAndComplete(ids []int) bool {
	if len(ids) == 0 {
		return false // Assuming the list shouldn't be empty
	}

	for i := 0; i < len(ids)-1; i++ {
		if ids[i+1] != ids[i]+1 {
			return false
		}
	}

	return true
}

func getArrayOfIDs(arr []ProcessResult) []int {
	ids := []int{}
	for _, msg := range arr {
		ids = append(ids, msg.ID)
	}
	return ids
}

// updates the current view with the messages coming in
func (m *Model) handleMsgProcessing(msg ProcessResult) tea.Cmd {
	m.appendAndOrderProcessResults(msg)
	areIdsAllThere := areIDsInOrderAndComplete(getArrayOfIDs(m.ArrayOfProcessResult))
	m.ProcessingMode = PROCESSING

	for _, msg := range m.ArrayOfProcessResult {
		if msg.Final && areIdsAllThere {
			log.Println("-----Final message found-----")
			return m.handleFinalChoiceMessage()
		}

		if len(msg.Result.Choices) > 0 {
			// Now you can safely use 'choice' since you've confirmed there's at least one element.
			// this is when we're done with the stream
			choice := msg.Result.Choices[0]

			// we need to keep appending content to our current answer in this case
			choiceString, cmdToRun := m.assertChoiceContentString(choice)
			if cmdToRun != nil {
				return cmdToRun
			}

			m.CurrentAnswer = m.CurrentAnswer + choiceString
		}
	}

	return nil
}

func (m *Model) resetStateAndCreateError(errMsg string) tea.Cmd {
	m.ProcessingMode = ERROR
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

func (m *Model) handleUpdateCurrentSession(session Session) tea.Cmd {
	m.userService.UpdateUserCurrentActiveSession(1, session.ID)

	m.CurrentSessionID = session.ID
	m.CurrentSessionName = session.SessionName
	m.ArrayOfMessages = session.Messages
	m.list.SetItems(ConstructListItems(m.AllSessions, m.CurrentSessionID))

	return SendUpdateCurrentSessionMsg()
}

func (m *Model) handleCurrentNormalMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd

	// We don't want to do anything if we're processing
	if m.ProcessingMode != IDLE {
		return cmd
	}

	switch msg.String() {
	case "ctrl+n":

		currentTime := time.Now()
		formattedTime := currentTime.Format(time.ANSIC)
		defaultSessionName := fmt.Sprintf("%s", formattedTime)
		newSession, _ := m.sessionService.InsertNewSession(defaultSessionName, []MessageToSend{})

		cmd = m.handleUpdateCurrentSession(newSession)
		m.updateSessionList()

	case "enter":
		i, ok := m.list.SelectedItem().(item)
		if ok {
			session, err := m.sessionService.GetSession(i.id)
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}

			cmd = m.handleUpdateCurrentSession(session)
		}

	case "e":
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
