package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joho/godotenv"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/migrations"
	"github.com/tearingItUp786/chatgpt-tui/panes"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

var AsyncDeps = []util.AsyncDependency{util.SettingsPaneModule, util.Orchestrator}

type model struct {
	viewReady        bool
	focused          util.Pane
	viewMode         util.ViewMode
	error            util.ErrorEvent
	currentSessionID string

	chatPane     panes.ChatPane
	promptPane   panes.PromptPane
	sessionsPane panes.SessionsPane
	settingsPane panes.SettingsPane
	loadedDeps   []util.AsyncDependency

	sessionOrchestrator sessions.Orchestrator

	terminalWidth  int
	terminalHeight int
}

func initialModal(db *sql.DB, ctx context.Context) model {
	promptPane := panes.NewPromptPane()
	sessionsPane := panes.NewSessionsPane(db, ctx)
	settingsPane := panes.NewSettingsPane(db, ctx)

	w, h := util.CalcChatPaneSize(util.DefaultTerminalWidth, util.DefaultTerminalHeight, false)
	chatPane := panes.NewChatPane(w, h)

	orchestrator := sessions.NewOrchestrator(db, ctx)

	return model{
		viewMode:            util.NormalMode,
		focused:             util.PromptPane,
		currentSessionID:    "",
		sessionOrchestrator: orchestrator,
		promptPane:          promptPane,
		sessionsPane:        sessionsPane,
		settingsPane:        settingsPane,
		chatPane:            chatPane,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.sessionOrchestrator.Init(),
		m.promptPane.Init(),
		m.sessionsPane.Init(),
		m.chatPane.Init(),
		m.settingsPane.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.sessionOrchestrator, cmd = m.sessionOrchestrator.Update(msg)
	cmds = append(cmds, cmd)

	m.promptPane, cmd = m.promptPane.Update(msg)
	cmds = append(cmds, cmd)

	if m.sessionOrchestrator.ProcessingMode == sessions.IDLE {
		m.sessionsPane, cmd = m.sessionsPane.Update(msg)
		cmds = append(cmds, cmd)
		m.settingsPane, cmd = m.settingsPane.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case util.AsyncDependencyReady:
		m.loadedDeps = append(m.loadedDeps, msg.Dependency)
		for _, dependency := range AsyncDeps {
			if !slices.Contains(m.loadedDeps, dependency) {
				continue
			}
			m.viewReady = true
		}
		m.promptPane = m.promptPane.Enable()

	case util.ErrorEvent:
		util.Log("Error: ", msg.Message)
		m.sessionOrchestrator.ProcessingMode = sessions.IDLE
		m.error = msg
		cmds = append(cmds, util.SendProcessingStateChangedMsg(false))

	case util.PromptReady:
		m.error = util.ErrorEvent{}
		m.sessionOrchestrator.ArrayOfMessages = append(m.sessionOrchestrator.ArrayOfMessages, clients.ConstructUserMessage(msg.Prompt))
		m.sessionOrchestrator.ProcessingMode = sessions.PROCESSING

		return m, tea.Batch(
			util.SendProcessingStateChangedMsg(true),
			m.chatPane.DisplayCompletion(m.sessionOrchestrator))

	case tea.KeyMsg:
		if !m.viewReady {
			break
		}
		switch keypress := msg.String(); keypress {

		case "ctrl+o":
			m.focused = util.PromptPane
			m.sessionsPane, _ = m.sessionsPane.Update(util.MakeFocusMsg(m.focused == util.SessionsPane))
			m.settingsPane, _ = m.settingsPane.Update(util.MakeFocusMsg(m.focused == util.SettingsPane))

			cmds = append(cmds, cmd)

			switch m.viewMode {
			case util.NormalMode:
				m.viewMode = util.ZenMode
				m.chatPane.SwitchToZenMode()
			case util.ZenMode:
				m.viewMode = util.NormalMode
				m.chatPane.SwitchToNormalMode()
			}
		}

		switch msg.Type {
		case tea.KeyTab:
			if m.promptPane.IsTypingInProcess() || !m.viewReady {
				break
			}

			m.focused = util.GetNewFocusMode(m.viewMode, m.focused, m.terminalWidth)

			m.sessionsPane, _ = m.sessionsPane.Update(util.MakeFocusMsg(m.focused == util.SessionsPane))
			m.settingsPane, _ = m.settingsPane.Update(util.MakeFocusMsg(m.focused == util.SettingsPane))
			m.chatPane, _ = m.chatPane.Update(util.MakeFocusMsg(m.focused == util.ChatPane))
			m.promptPane, _ = m.promptPane.Update(util.MakeFocusMsg(m.focused == util.PromptPane))

		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		isZenMode := m.viewMode == util.ZenMode
		chatPaneWidth, chatPaneHeight := util.CalcChatPaneSize(m.terminalWidth, m.terminalHeight, isZenMode)

		m.chatPane.SetPaneWitdth(chatPaneWidth)
		m.chatPane.SetPaneHeight(chatPaneHeight)

		m.chatPane, cmd = m.chatPane.Update(msg)
		cmds = append(cmds, cmd)
		m.settingsPane, cmd = m.settingsPane.Update(msg)
		cmds = append(cmds, cmd)
		m.sessionsPane, cmd = m.sessionsPane.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.chatPane, cmd = m.chatPane.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var windowViews string

	settingsAndSessionPanes := lipgloss.JoinVertical(
		lipgloss.Left,
		m.settingsPane.View(),
		m.sessionsPane.View(),
	)

	mainView := m.chatPane.View()
	if m.error.Message != "" {
		mainView = m.chatPane.DisplayError(m.error.Message)
	}

	secondaryScreen := ""
	if m.viewMode == util.NormalMode {
		secondaryScreen = settingsAndSessionPanes
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
