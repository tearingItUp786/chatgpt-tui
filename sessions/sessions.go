package sessions

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tearingItUp786/chatgpt-tui/clients"
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

type Orchestrator struct {
	sessionService  *SessionService
	userService     *user.UserService
	settingsService *settings.SettingsService
	config          config.Config

	OpenAiClient         *clients.OpenAiClient
	Settings             util.Settings
	CurrentSessionID     int
	CurrentSessionName   string
	ArrayOfProcessResult []clients.ProcessApiCompletionResponse
	ArrayOfMessages      []util.MessageToSend
	CurrentAnswer        string
	AllSessions          []Session
	ProcessingMode       string
}

func NewOrchestrator(db *sql.DB, ctx context.Context) Orchestrator {
	ss := NewSessionService(db)
	us := user.NewUserService(db)

	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}

	settingsService := settings.NewSettingsService(db)
	openAiClient := clients.NewOpenAiClient(config.ChatGPTApiUrl, config.SystemMessage)

	return Orchestrator{
		config:               *config,
		ArrayOfProcessResult: []clients.ProcessApiCompletionResponse{},
		sessionService:       ss,
		userService:          us,
		settingsService:      settingsService,
		OpenAiClient:         openAiClient,
		ProcessingMode:       IDLE,
	}
}

func (m Orchestrator) Init() tea.Cmd {
	// Need to load the latest session as the current session  (select recently created)
	// OR we need to create a brand new session for the user with a random name (insert new and return)

	settingsData := func() tea.Msg { return m.settingsService.GetSettings(nil, m.config) }

	dbData := func() tea.Msg {
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

		dbLoadEvent := LoadDataFromDB{
			Session:                mostRecentSession,
			AllSessions:            allSessions,
			CurrentActiveSessionID: user.CurrentActiveSessionID,
		}
		return dbLoadEvent
	}

	return tea.Batch(settingsData, dbData)
}

func (m Orchestrator) Update(msg tea.Msg) (Orchestrator, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case util.CopyLastMsg:
		latestBotMessage, err := m.GetLatestBotMessage()
		if err == nil {
			clipboard.WriteAll(latestBotMessage)
		}

	case util.CopyAllMsgs:
		clipboard.WriteAll(m.GetMessagesAsString())

	case UpdateCurrentSession:
		m.CurrentSessionID = msg.Session.ID
		m.CurrentSessionName = msg.Session.SessionName
		m.ArrayOfMessages = msg.Session.Messages

	case LoadDataFromDB:
		m.CurrentSessionID = msg.CurrentActiveSessionID
		m.CurrentSessionName = msg.Session.SessionName
		m.ArrayOfMessages = msg.Session.Messages
		m.AllSessions = msg.AllSessions

	case settings.UpdateSettingsEvent:
		m.Settings = msg.Settings

	case clients.ProcessApiCompletionResponse:
		// add the latest message to the array of messages
		cmds = append(cmds, m.handleMsgProcessing(msg))
		cmds = append(cmds, SendResponseChunkProcessedMsg(m.CurrentAnswer, m.ArrayOfMessages))
	}

	return m, tea.Batch(cmds...)
}

func (m Orchestrator) GetCompletion(resp chan clients.ProcessApiCompletionResponse) tea.Cmd {
	return m.OpenAiClient.RequestCompletion(m.ArrayOfMessages, m.Settings, resp)
}

func (m Orchestrator) GetLatestBotMessage() (string, error) {
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

func (m Orchestrator) GetMessagesAsString() string {
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
func (m *Orchestrator) appendAndOrderProcessResults(msg clients.ProcessApiCompletionResponse) {
	m.ArrayOfProcessResult = append(m.ArrayOfProcessResult, msg)
	m.CurrentAnswer = ""

	// we need to sort on ID here because go routines are done in different threads
	// and the order in which our channel receives messages is not guaranteed.
	// TODO: look into a better way to insert (can I Insert in order)
	sort.SliceStable(m.ArrayOfProcessResult, func(i, j int) bool {
		return m.ArrayOfProcessResult[i].ID < m.ArrayOfProcessResult[j].ID
	})
}

func (m *Orchestrator) assertChoiceContentString(choice clients.Choice) (string, tea.Cmd) {
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

func (m *Orchestrator) handleFinalChoiceMessage() tea.Cmd {
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

	return util.SendProcessingStateChangedMsg(false)
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
func (m *Orchestrator) handleMsgProcessing(msg clients.ProcessApiCompletionResponse) tea.Cmd {
	m.appendAndOrderProcessResults(msg)
	areIdsAllThere := areIDsInOrderAndComplete(getArrayOfIDs(m.ArrayOfProcessResult))
	m.ProcessingMode = PROCESSING

	for _, msg := range m.ArrayOfProcessResult {
		if msg.Final && areIdsAllThere {
			util.Log("-----Final message found-----")
			return tea.Batch(m.handleFinalChoiceMessage(), util.SendProcessingStateChangedMsg(false))
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

func (m *Orchestrator) resetStateAndCreateError(errMsg string) tea.Cmd {
	m.ProcessingMode = ERROR
	m.ArrayOfProcessResult = []clients.ProcessApiCompletionResponse{}
	m.CurrentAnswer = ""
	return util.MakeErrorMsg(errMsg)
}
