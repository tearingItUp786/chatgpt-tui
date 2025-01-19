package sessions

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/settings"
	"github.com/tearingItUp786/chatgpt-tui/user"
	"github.com/tearingItUp786/chatgpt-tui/util"
	"golang.org/x/net/context"
)

const (
	IDLE       = "idle"
	PROCESSING = "processing"
	ERROR      = "error"
)

type Model struct {
	textInput         textinput.Model
	sessionsList      components.SessionsList
	isFocused         bool
	currentEditID     int
	sessionService    *SessionService
	userService       *user.UserService
	settingsContainer lipgloss.Style
	config            config.Config

	OpenAiClient         *clients.OpenAiClient
	Settings             util.Settings
	CurrentSessionID     int
	CurrentSessionName   string
	terminalWidth        int
	terminalHeight       int
	ArrayOfProcessResult []clients.ProcessApiCompletionResponse
	ArrayOfMessages      []util.MessageToSend
	CurrentAnswer        string
	AllSessions          []Session
	ProcessingMode       string
}

func New(db *sql.DB, ctx context.Context) Model {
	ss := NewSessionService(db)
	us := user.NewUserService(db)

	// default --> get some default settings
	defaultSettings := util.Settings{
		Model:     "gpt-3.5-turbo",
		MaxTokens: 300,
		Frequency: 0,
	}

	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}

	if len(config.DefaultModel) > 0 {
		defaultSettings.Model = config.DefaultModel
	}

	openAiClient := clients.NewOpenAiClient(config.ChatGPTApiUrl, config.SystemMessage)

	return Model{
		config:               *config,
		ArrayOfProcessResult: []clients.ProcessApiCompletionResponse{},
		sessionService:       ss,
		userService:          us,
		Settings:             defaultSettings,
		OpenAiClient:         openAiClient,
		ProcessingMode:       IDLE,
		settingsContainer: lipgloss.NewStyle().
			AlignVertical(lipgloss.Top).
			Border(lipgloss.ThickBorder(), true).
			BorderForeground(util.NormalTabBorderColor),
	}
}

type LoadDataFromDB struct {
	Session Session

	allSessions            []Session
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

type UpdateCurrentSession struct {
	Session Session
}

func SendUpdateCurrentSessionMsg(session Session) tea.Cmd {
	return func() tea.Msg {
		return UpdateCurrentSession{
			Session: session,
		}
	}
}

type ResponseChunkProcessed struct {
	PreviousMsgArray []util.MessageToSend
	ChunkMessage     string
}

func SendResponseChunkProcessedMsg(msg string, previousMsgs []util.MessageToSend) tea.Cmd {
	return func() tea.Msg {
		return ResponseChunkProcessed{
			PreviousMsgArray: previousMsgs,
			ChunkMessage:     msg,
		}
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
			Session:                mostRecentSession,
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

	case util.CopyLastMsg:
		latestBotMessage, err := m.GetLatestBotMessage()
		if err == nil {
			clipboard.WriteAll(latestBotMessage)
		}

	case util.CopyAllMsgs:
		clipboard.WriteAll(m.GetMessagesAsString())

	case LoadDataFromDB:
		m.CurrentSessionID = msg.currentActiveSessionID
		m.CurrentSessionName = msg.Session.SessionName
		m.ArrayOfMessages = msg.Session.Messages
		m.AllSessions = msg.allSessions

		listItems := constructSessionsListItems(m.AllSessions, m.currentEditID)
		m.sessionsList = components.NewSessionsList(listItems)
		m.currentEditID = -1

	case settings.UpdateSettingsEvent:
		m.Settings = msg.Settings

	case util.FocusEvent:
		m.isFocused = msg.IsFocused
		m.currentEditID = -1

	case util.OurWindowResize:
		width := m.terminalWidth - msg.Width - 5
		m.settingsContainer = m.settingsContainer.Width(width)

	case clients.ProcessApiCompletionResponse:
		// add the latest message to the array of messages
		m.handleMsgProcessing(msg)
		cmds = append(cmds, SendResponseChunkProcessedMsg(m.CurrentAnswer, m.ArrayOfMessages))

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
		m.sessionsList, cmd = m.sessionsList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	listView := m.normalListView()

	if m.isFocused {
		listView = m.sessionsList.EditListView(m.terminalHeight)
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

func (m Model) GetCompletion(resp chan clients.ProcessApiCompletionResponse) tea.Cmd {
	return m.OpenAiClient.RequestCompletion(m.ArrayOfMessages, m.Settings, resp)
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

// MIGHT BE WORTH TO MOVE TO A SEP FILE
func (m *Model) appendAndOrderProcessResults(msg clients.ProcessApiCompletionResponse) {
	m.ArrayOfProcessResult = append(m.ArrayOfProcessResult, msg)
	m.CurrentAnswer = ""

	// we need to sort on ID here because go routines are done in different threads
	// and the order in which our channel receives messages is not guaranteed.
	// TODO: look into a better way to insert (can I Insert in order)
	sort.SliceStable(m.ArrayOfProcessResult, func(i, j int) bool {
		return m.ArrayOfProcessResult[i].ID < m.ArrayOfProcessResult[j].ID
	})
}

func (m *Model) assertChoiceContentString(choice clients.Choice) (string, tea.Cmd) {
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

func constructJsonMessage(arrayOfProcessResult []clients.ProcessApiCompletionResponse) (util.MessageToSend, error) {
	newMessage := util.MessageToSend{Role: "assistant", Content: ""}

	for _, aMessage := range arrayOfProcessResult {
		if aMessage.Final {
			util.Log("Hit final in construct", aMessage.Result)
			break
		}

		if len(aMessage.Result.Choices) > 0 {
			choice := aMessage.Result.Choices[0]
			// TODO: we need a helper for this
			if choice.FinishReason == "stop" || choice.FinishReason == "length" {
				util.Log("Hit stop or length in construct")
				break
			}

			if content, ok := choice.Delta["content"].(string); ok {
				newMessage.Content += content
			} else {
				// Handle the case where the type assertion fails, e.g., log an error or return
				util.Log("type assertion to string failed for choice.Delta[\"content\"]")
				formattedError := fmt.Errorf("type assertion to string failed for choice.Delta[\"content\"]")
				return util.MessageToSend{}, formattedError
			}

		}
	}
	return newMessage, nil
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
	m.ArrayOfProcessResult = []clients.ProcessApiCompletionResponse{}

	if err != nil {
		util.Log("Error updating session messages", err)
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

func getArrayOfIDs(arr []clients.ProcessApiCompletionResponse) []int {
	ids := []int{}
	for _, msg := range arr {
		ids = append(ids, msg.ID)
	}
	return ids
}

// updates the current view with the messages coming in
func (m *Model) handleMsgProcessing(msg clients.ProcessApiCompletionResponse) tea.Cmd {
	m.appendAndOrderProcessResults(msg)
	areIdsAllThere := areIDsInOrderAndComplete(getArrayOfIDs(m.ArrayOfProcessResult))
	m.ProcessingMode = PROCESSING

	for _, msg := range m.ArrayOfProcessResult {
		if msg.Final && areIdsAllThere {
			util.Log("-----Final message found-----")
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
	m.ArrayOfProcessResult = []clients.ProcessApiCompletionResponse{}
	m.CurrentAnswer = ""
	return util.MakeErrorMsg(errMsg)
}

func (m *Model) handleCurrentEditID(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if msg.String() == "enter" {
		if m.textInput.Value() != "" {
			m.sessionService.UpdateSessionName(m.currentEditID, m.textInput.Value())
			m.updateSessionsList()
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
	listItems := constructSessionsListItems(m.AllSessions, m.CurrentSessionID)

	m.sessionsList.SetItems(listItems)

	return SendUpdateCurrentSessionMsg(session)
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
		newSession, _ := m.sessionService.InsertNewSession(defaultSessionName, []util.MessageToSend{})

		cmd = m.handleUpdateCurrentSession(newSession)
		m.updateSessionsList()

	case "enter":
		i, ok := m.sessionsList.GetSelectedItem()
		if ok {
			session, err := m.sessionService.GetSession(i.Id)
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}

			cmd = m.handleUpdateCurrentSession(session)
		}

	case "e":
		ti := textinput.New()
		ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(2)
		m.textInput = ti
		i, ok := m.sessionsList.GetSelectedItem()
		if ok {
			m.currentEditID = i.Id
			m.textInput.Placeholder = "New Session Name"
		}
		m.textInput.Focus()
		m.textInput.CharLimit = 100

	case "d":
		i, ok := m.sessionsList.GetSelectedItem()
		if ok {
			// delete this one if it's not the active one
			if i.Id != m.CurrentSessionID {
				m.sessionService.DeleteSession(i.Id)
				m.updateSessionsList()
			}
		}

	}

	return cmd
}

func constructSessionsListItems(sessions []Session, currentSessionId int) []list.Item {
	items := []list.Item{}

	for _, session := range sessions {
		anItem := components.SessionListItem{
			Id:       session.ID,
			Text:     session.SessionName,
			IsActive: session.ID == currentSessionId,
		}
		items = append(items, anItem)
	}

	return items
}

func (m *Model) updateSessionsList() {
	m.AllSessions, _ = m.sessionService.GetAllSessions()
	items := constructSessionsListItems(m.AllSessions, m.CurrentSessionID)
	m.sessionsList.SetItems(items)
}
