package views

import (
	"context"
	"database/sql"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/panes"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

var asyncDeps = []util.AsyncDependency{util.SettingsPaneModule, util.Orchestrator}

type MainView struct {
	viewReady        bool
	focused          util.Pane
	viewMode         util.ViewMode
	error            util.ErrorEvent
	currentSessionID string

	chatPane     panes.ChatPane
	promptPane   panes.PromptPane
	sessionsPane panes.SessionsPane
	settingsPane panes.SettingsPane
	infoPane     panes.InfoPane
	loadedDeps   []util.AsyncDependency

	sessionOrchestrator sessions.Orchestrator
	context             context.Context
	completionContext   context.Context
	cancelInference     context.CancelFunc

	terminalWidth  int
	terminalHeight int
}

func NewMainView(db *sql.DB, ctx context.Context) MainView {
	promptPane := panes.NewPromptPane(ctx)
	sessionsPane := panes.NewSessionsPane(db, ctx)
	settingsPane := panes.NewSettingsPane(db, ctx)
	statusBarPane := panes.NewInfoPane(db, ctx)

	w, h := util.CalcChatPaneSize(util.DefaultTerminalWidth, util.DefaultTerminalHeight, util.NormalMode)
	chatPane := panes.NewChatPane(ctx, w, h)

	orchestrator := sessions.NewOrchestrator(db, ctx)

	return MainView{
		viewMode:            util.NormalMode,
		focused:             util.PromptPane,
		currentSessionID:    "",
		sessionOrchestrator: orchestrator,
		promptPane:          promptPane,
		sessionsPane:        sessionsPane,
		settingsPane:        settingsPane,
		infoPane:            statusBarPane,
		chatPane:            chatPane,
		context:             ctx,
	}
}

func (m MainView) Init() tea.Cmd {
	return tea.Batch(
		m.sessionOrchestrator.Init(),
		m.promptPane.Init(),
		m.sessionsPane.Init(),
		m.chatPane.Init(),
		m.settingsPane.Init(),
		m.sessionsPane.Init(),
	)
}

func (m MainView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.sessionOrchestrator, cmd = m.sessionOrchestrator.Update(msg)
	cmds = append(cmds, cmd)

	m.infoPane, cmd = m.infoPane.Update(msg)
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
		for _, dependency := range asyncDeps {
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
		m.viewReady = true
		cmds = append(cmds, util.SendProcessingStateChangedMsg(false))

	case util.PromptReady:
		m.error = util.ErrorEvent{}
		m.sessionOrchestrator.ArrayOfMessages = append(m.sessionOrchestrator.ArrayOfMessages, clients.ConstructUserMessage(msg.Prompt))
		m.sessionOrchestrator.ProcessingMode = sessions.PROCESSING
		m.viewMode = util.NormalMode

		completionContext, cancelInference := context.WithCancel(m.context)
		m.completionContext = completionContext
		m.cancelInference = cancelInference
		return m, tea.Batch(
			util.SendProcessingStateChangedMsg(true),
			m.chatPane.DisplayCompletion(m.completionContext, m.sessionOrchestrator),
			util.SendViewModeChangedMsg(m.viewMode))

	case tea.KeyMsg:
		if !m.viewReady {
			break
		}
		switch keypress := msg.String(); keypress {

		case "ctrl+b":
			m.cancelInference()

		case "ctrl+o":
			m.focused = util.PromptPane
			m.sessionsPane, _ = m.sessionsPane.Update(util.MakeFocusMsg(m.focused == util.SessionsPane))
			m.settingsPane, _ = m.settingsPane.Update(util.MakeFocusMsg(m.focused == util.SettingsPane))

			cmds = append(cmds, cmd)

			switch m.viewMode {
			case util.NormalMode:
				m.viewMode = util.ZenMode
			case util.ZenMode:
				m.viewMode = util.NormalMode
			}

			cmds = append(cmds, util.SendViewModeChangedMsg(m.viewMode))
		case "ctrl+f":
			if m.focused != util.PromptPane {
				break
			}

			switch m.viewMode {
			case util.NormalMode:
				m.viewMode = util.TextEditMode
			case util.ZenMode:
				m.viewMode = util.TextEditMode
			case util.TextEditMode:
				m.viewMode = util.NormalMode
			}

			cmds = append(cmds, util.SendViewModeChangedMsg(m.viewMode))
		}

		switch msg.Type {
		case tea.KeyTab:
			if m.promptPane.IsTypingInProcess() || m.chatPane.IsVisualMode || !m.viewReady {
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

func (m MainView) View() string {
	var windowViews string

	settingsAndSessionPanes := lipgloss.JoinVertical(
		lipgloss.Left,
		m.settingsPane.View(),
		m.sessionsPane.View(),
		m.infoPane.View(),
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
		lipgloss.JoinVertical(
			lipgloss.Left,
			windowViews,
			promptView,
		),
	)
}
