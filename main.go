package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"github.com/muesli/reflow/wrap"
	"github.com/tearingItUp786/golang-tui/migrations"
	"github.com/tearingItUp786/golang-tui/sessions"
	"github.com/tearingItUp786/golang-tui/settings"
	"github.com/tearingItUp786/golang-tui/util"
)

type model struct {
	ready            bool
	focused          util.FocusPane
	viewMode         util.ViewMode
	msgChan          chan sessions.ProcessResult
	error            util.ErrorEvent
	currentSessionID string

	promptContainer lipgloss.Style
	viewport        viewport.Model
	promptInput     textinput.Model
	settingsModel   settings.Model
	sessionModel    sessions.Model
	terminalWidth   int
	terminalHeight  int
}

func initialModal(db *sql.DB) model {
	ti := textinput.New()
	ti.Placeholder = "Ask ChatGPT a question!"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(util.ActiveTabBorderColor))
	ti.Focus()

	si := settings.New(db)
	sm := sessions.New(db)

	msgChan := make(chan sessions.ProcessResult)

	return model{
		viewMode:         util.NormalMode,
		focused:          util.PromptType,
		promptInput:      ti,
		settingsModel:    si,
		currentSessionID: "",
		sessionModel:     sm,
		msgChan:          msgChan,
		promptContainer: lipgloss.NewStyle().
			AlignVertical(lipgloss.Bottom).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(util.ActiveTabBorderColor).
			MaxHeight(4).
			MarginTop(1),
	}
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan sessions.ProcessResult) tea.Cmd {
	return func() tea.Msg {
		someMessage := <-sub
		return someMessage
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.promptInput.Cursor.BlinkCmd(),
		waitForActivity(m.msgChan),
		m.sessionModel.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd                    tea.Cmd
		cmds                   []tea.Cmd
		enableUpdateOfViewport = true
	)

	isPromptFocused := m.focused == util.PromptType
	isChatMessagesFocused := m.focused == util.ChatMessagesType

	// the settings model is actually an input into the session model
	if m.viewMode == util.NormalMode {
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		cmds = append(cmds, cmd)
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) { // each time we get a new message coming in from the model
	// lets handle it and pass it to the lower model
	case sessions.LoadDataFromDB:
		oldContent := m.sessionModel.GetMessagesAsString()
		if oldContent == "" {
			oldContent = util.MotivationalMessage
		}
		m.viewport.SetContent(wrap.String(oldContent, m.terminalWidth/3*2))
		return m, cmd

	case sessions.UpdateCurrentSession:
		oldContent := m.sessionModel.GetMessagesAsString()
		if oldContent == "" {
			oldContent = util.MotivationalMessage
		}
		m.viewport.SetContent(wrap.String(oldContent, m.terminalWidth/3*2))
		return m, cmd

	case sessions.ProcessResult:
		oldContent := m.sessionModel.GetMessagesAsString()
		styledBufferMessage := sessions.RenderBotMessage(m.sessionModel.CurrentAnswer, m.terminalWidth/3*2)
		m.viewport.SetContent(wrap.String(oldContent+"\n"+styledBufferMessage, m.terminalWidth/3*2))

		m.viewport.GotoBottom()

		return m, waitForActivity(m.msgChan)

	case util.ErrorEvent:
		m.error = msg

	case tea.KeyMsg:

		if !isChatMessagesFocused {
			enableUpdateOfViewport = false
		}

		switch keypress := msg.String(); keypress {
		case "ctrl+o":
			switch m.viewMode {
			case util.NormalMode:
				m.viewMode = util.ZenMode
			case util.ZenMode:
				m.viewMode = util.NormalMode
			}
		}

		switch msg.Type {

		case tea.KeyTab:
			m.focused = util.GetNewFocusMode(m.viewMode, m.focused)
			if m.viewMode == util.NormalMode {
				m.sessionModel, _ = m.sessionModel.Update(util.MakeFocusMsg(m.focused == util.SessionsType))
				m.settingsModel, _ = m.settingsModel.Update(util.MakeFocusMsg(m.focused == util.SettingsType))
			}

			if m.focused == util.PromptType {
				borderColor := util.ActiveTabBorderColor
				m.promptContainer = m.promptContainer.Copy().BorderForeground(borderColor)
				m.promptInput.PromptStyle = m.promptInput.PromptStyle.Copy().Foreground(lipgloss.Color(util.ActiveTabBorderColor))
				m.promptInput.Focus()
			} else {
				borderColor := util.NormalTabBorderColor
				m.promptContainer = m.promptContainer.Copy().BorderForeground(borderColor)
				m.promptInput.PromptStyle = m.promptInput.PromptStyle.Copy().Foreground(lipgloss.Color(util.NormalTabBorderColor))
				m.promptInput.Blur()
			}
			return m, cmd

		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if isPromptFocused {
				// Start CallChatGpt on Enter key
				m.error = util.ErrorEvent{}
				m.sessionModel.ArrayOfMessages = append(m.sessionModel.ArrayOfMessages, sessions.ConstructUserMessage(m.promptInput.Value()))
				content := m.sessionModel.GetMessagesAsString()
				m.promptInput.SetValue("")
				// TODO: add a loading indicator / icon when we are waiting for chat gpt to return with a response.
				m.viewport.SetContent(wrap.String(content, m.terminalWidth/3*2))

				return m, m.sessionModel.CallChatGpt(m.msgChan)
			}
		}

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		m.promptContainer = m.promptContainer.Copy().MaxWidth(m.terminalWidth).
			Width(m.terminalWidth - 2)

		// TODO: get rid of this magic number
		prompContinerHeight := m.promptContainer.GetHeight() + 5

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-prompContinerHeight)
			m.viewport.Style.MaxHeight(msg.Height)
			m.ready = true
			m.promptInput.Width = msg.Width - 3
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - prompContinerHeight
			m.promptInput.Width = msg.Width - 3
		}

	}

	m.promptInput, cmd = m.promptInput.Update(msg)
	cmds = append(cmds, cmd)

	if enableUpdateOfViewport {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var windowViews string

	borderColor := util.NormalTabBorderColor
	if m.focused == util.ChatMessagesType {
		borderColor = util.ActiveTabBorderColor
	}

	chatMessagesViewRender := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Width(m.terminalWidth - 2).
		MarginRight(1)

	settingsAndSessionViews := lipgloss.JoinVertical(
		lipgloss.Left,
		m.settingsModel.View(),
		m.sessionModel.View(),
	)

	strToRender := m.viewport.View()
	if m.error.Message != "" {
		log.Println("error", m.error.Message)
		strToRender = m.error.Message
	}

	secondaryScreen := ""
	if m.viewMode == util.NormalMode {
		secondaryScreen = settingsAndSessionViews
		chatMessagesViewRender.Width(m.terminalWidth / 3 * 2)
	}

	mainView := chatMessagesViewRender.Render(strToRender)

	windowViews = lipgloss.NewStyle().
		Align(lipgloss.Right, lipgloss.Right).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				mainView,
				secondaryScreen,
			),
		)

	promptView := m.promptContainer.Render(
		m.promptInput.View(),
	)

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

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// run migrations for our database
	db := util.InitDb()
	err = util.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	p := tea.NewProgram(
		initialModal(db),
		tea.WithAltScreen(),
		// tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	_, err = p.Run()
	if err != nil {
		log.Fatal(err)
	}
}
