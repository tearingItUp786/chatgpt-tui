package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joho/godotenv"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/migrations"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/settings"
	"github.com/tearingItUp786/chatgpt-tui/util"
	"github.com/tearingItUp786/chatgpt-tui/views"
)

type model struct {
	ready            bool
	focused          util.FocusPane
	viewMode         util.ViewMode
	error            util.ErrorEvent
	currentSessionID string

	chatPane       views.ChatPane
	promptPane     views.PromptPane
	settingsModel  settings.Model
	sessionModel   sessions.Model
	terminalWidth  int
	terminalHeight int
}

func initialModal(db *sql.DB, ctx context.Context) model {
	promptPane := views.NewPromptPane()

	si := settings.New(db, ctx)
	sm := sessions.New(db, ctx)

	return model{
		ready:            false,
		viewMode:         util.NormalMode,
		focused:          util.PromptType,
		settingsModel:    si,
		currentSessionID: "",
		sessionModel:     sm,
		promptPane:       promptPane,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.promptPane.Init(),
		m.sessionModel.Init(),
		m.chatPane.Init(),
		m.settingsModel.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// the settings model is actually an input into the session model
	m.sessionModel, cmd = m.sessionModel.Update(msg)
	cmds = append(cmds, cmd)

	m.chatPane, cmd = m.chatPane.Update(msg)
	cmds = append(cmds, cmd)

	if m.sessionModel.ProcessingMode == sessions.IDLE {
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case util.ErrorEvent:
		util.Log("Error: ", msg.Message)
		m.sessionModel.ProcessingMode = sessions.IDLE
		m.error = msg
		cmds = append(cmds, util.SendProcessingStateChangedMsg(false))

	case util.PromptReady:
		m.error = util.ErrorEvent{}
		m.sessionModel.ArrayOfMessages = append(m.sessionModel.ArrayOfMessages, clients.ConstructUserMessage(msg.Prompt))
		m.sessionModel.ProcessingMode = sessions.PROCESSING

		return m, tea.Batch(
			util.SendProcessingStateChangedMsg(true),
			// use current session for requests to OpenAI API
			m.chatPane.DisplayCompletion(m.sessionModel))

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {

		case "ctrl+o":
			m.focused = util.PromptType
			m.sessionModel, _ = m.sessionModel.Update(util.MakeFocusMsg(m.focused == util.SessionsType))
			m.settingsModel, _ = m.settingsModel.Update(util.MakeFocusMsg(m.focused == util.SettingsType))

			cmds = append(cmds, cmd)

			switch m.viewMode {
			case util.NormalMode:
				m.viewMode = util.ZenMode
				m.chatPane.SwitchToZenMode()
			case util.ZenMode:
				m.viewMode = util.NormalMode
				m.chatPane.SwitchToNormalMode()
			}

			chatContainerWidth := m.chatPane.GetWidth()
			m.settingsModel, cmd = m.settingsModel.Update(util.MakeWindowResizeMsg(chatContainerWidth))
			cmds = append(cmds, cmd)
			m.sessionModel, cmd = m.sessionModel.Update(util.MakeWindowResizeMsg(chatContainerWidth))
			cmds = append(cmds, cmd)
		}

		switch msg.Type {

		case tea.KeyTab:
			if m.promptPane.IsTypingInProcess() {
				break
			}

			m.focused = util.GetNewFocusMode(m.viewMode, m.focused, m.terminalWidth)

			m.sessionModel, _ = m.sessionModel.Update(util.MakeFocusMsg(m.focused == util.SessionsType))
			m.settingsModel, _ = m.settingsModel.Update(util.MakeFocusMsg(m.focused == util.SettingsType))
			m.chatPane, _ = m.chatPane.Update(util.MakeFocusMsg(m.focused == util.ChatMessagesType))
			m.promptPane, _ = m.promptPane.Update(util.MakeFocusMsg(m.focused == util.PromptType))

		case tea.KeyCtrlC:
			return m, tea.Quit

		}

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		chatPaneWidth, chatPaneHeight := util.CalcChatPaneSize(m.terminalWidth, m.terminalHeight, false)

		util.Log("viewMode:", m.viewMode)
		if m.viewMode == util.ZenMode {
			chatPaneWidthZen, _ := util.CalcChatPaneSize(m.terminalWidth, m.terminalHeight, true)
			m.chatPane.SetPaneWitdth(chatPaneWidthZen)
		}

		if !m.ready {
			m.chatPane = views.NewChatPane(chatPaneWidth, chatPaneHeight)
			m.ready = true
		} else {
			m.chatPane.SetPaneWitdth(chatPaneWidth)
			m.chatPane.SetPaneHeight(chatPaneHeight)
		}

		m.settingsModel, cmd = m.settingsModel.Update(util.MakeWindowResizeMsg(m.chatPane.GetWidth()))
		cmds = append(cmds, cmd)
		m.sessionModel, cmd = m.sessionModel.Update(util.MakeWindowResizeMsg(m.chatPane.GetWidth()))
		cmds = append(cmds, cmd)
		m.promptPane, cmd = m.promptPane.Update(util.MakeWindowResizeMsg(msg.Width))
		cmds = append(cmds, cmd)
	}

	m.chatPane, cmd = m.chatPane.Update(msg)
	cmds = append(cmds, cmd)
	m.promptPane, cmd = m.promptPane.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var windowViews string

	settingsAndSessionViews := lipgloss.JoinVertical(
		lipgloss.Left,
		m.settingsModel.View(),
		m.sessionModel.View(),
	)

	mainView := m.chatPane.View()
	if m.error.Message != "" {
		mainView = m.chatPane.DisplayError(m.error.Message)
	}

	secondaryScreen := ""
	if m.viewMode == util.NormalMode {
		secondaryScreen = settingsAndSessionViews
	}

	windowViews = lipgloss.NewStyle().
		Align(lipgloss.Right, lipgloss.Right).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				mainView,
				secondaryScreen,
			),
		)

	promptView := m.promptPane.View()

	return lipgloss.NewStyle().Render(
		windowViews,
		promptView,
	)
}

func main() {
	env := os.Getenv("FOO_ENV")
	if "" == env {
		env = "development"
	}

	godotenv.Load(".env." + env + ".local")
	if "test" != env {
		godotenv.Load(".env.local")
	}
	godotenv.Load(".env." + env)
	godotenv.Load() // The Original .env

	appPath, err := util.GetAppDataPath()
	f, err := tea.LogToFile(filepath.Join(appPath, "debug.log"), "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if "" == apiKey {
		fmt.Println("OPENAI_API_KEY not set; set it in your profile")
		fmt.Printf("export OPENAI_API_KEY=your_key in the config for :%v \n", os.Getenv("SHELL"))
		fmt.Println("Exiting...")
		os.Exit(1)
	}

	// delete files if in dev mode
	util.DeleteFilesIfDevMode()
	// validate config
	configToUse := config.CreateAndValidateConfig()

	// run migrations for our database
	db := util.InitDb()
	err = util.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		log.Println("Error: ", err)
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	ctxWithConfig := config.WithConfig(ctx, &configToUse)

	p := tea.NewProgram(
		initialModal(db, ctxWithConfig),
		tea.WithAltScreen(),
		// tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	_, err = p.Run()
	if err != nil {
		log.Fatal(err)
	}
}
