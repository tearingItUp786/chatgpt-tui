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
	"github.com/tearingItUp786/golang-tui/other"
	"github.com/tearingItUp786/golang-tui/sessions"
	"github.com/tearingItUp786/golang-tui/settings"
)

// fake enum to keep tab of the currently focused pane
const (
	settingsType = iota
	sessionsType
	promptType
)

type model struct {
	ready           bool
	focused         int
	promptContainer lipgloss.Style
	viewport        viewport.Model
	promptInput     textinput.Model
	settingsModel   settings.Model
	sessionModel    sessions.Model
	msgChan         chan sessions.ProcessResult

	currentSessionID string
	terminalWidth    int
	terminalHeight   int
}

func initialModal(db *sql.DB) model {
	ti := textinput.New()
	ti.Placeholder = "Ask ChatGPT a question!"
	ti.Focus()

	si := settings.New()
	sm := sessions.New(db)

	msgChan := make(chan sessions.ProcessResult)

	return model{
		focused:          sessionsType,
		promptInput:      ti,
		settingsModel:    si,
		currentSessionID: "",
		sessionModel:     sm,
		msgChan:          msgChan,
		promptContainer: lipgloss.NewStyle().
			AlignVertical(lipgloss.Bottom).
			BorderStyle(lipgloss.NormalBorder()).
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
	log.Println("init main")
	return tea.Batch(
		m.promptInput.Cursor.BlinkCmd(),
		waitForActivity(m.msgChan),
		m.sessionModel.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	// each time we get a new message coming in from the model
	// lets handle it and pass it to the lower model
	case sessions.LoadDataFromDB:
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		oldContent := m.sessionModel.GetMessagesAsString()
		if oldContent == "" {
			oldContent = "Everyone starts somewhere. You can do it!"
		}
		m.viewport.SetContent(wrap.String(oldContent, m.terminalWidth/3*2))
		return m, cmd

	case sessions.ProcessResult:
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		oldContent := m.sessionModel.GetMessagesAsString()
		styledBufferMessage := sessions.RenderBotMessage(m.sessionModel.CurrentAnswer, m.terminalWidth/3*2)
		m.viewport.SetContent(wrap.String(oldContent+"\n"+styledBufferMessage, m.terminalWidth/3*2))

		return m, waitForActivity(m.msgChan)

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyTab:
			m.focused = (m.focused + 1) % 3
			m.viewport.SetContent(m.focusedPaneName())
			return m, cmd

		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			// Start CallChatGpt on Enter key
			m.sessionModel.ArrayOfMessages = append(m.sessionModel.ArrayOfMessages, sessions.ConstructUserMessage(m.promptInput.Value()))
			content := m.sessionModel.GetMessagesAsString()
			m.promptInput.SetValue("")
			// TODO: add a loading indicator / icon when we are waiting for chat gpt to return with a response.
			m.viewport.SetContent(wrap.String(content, m.terminalWidth/3*2))

			return m, m.sessionModel.CallChatGpt(m.msgChan)
		}

	case tea.WindowSizeMsg:
		log.Printf("width : %v", msg.Width)
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		m.promptContainer = m.promptContainer.Copy().MaxWidth(m.terminalWidth).
			Width(m.terminalWidth - 3)

		// TODO: get rid of this magic number
		prompContinerHeight := m.promptContainer.GetHeight() + 5

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-prompContinerHeight) // need to get rid of this magic number
			m.viewport.Style.MaxHeight(msg.Height)
			m.ready = true

		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - prompContinerHeight
		}

		updatedSettingsModel, _ := m.settingsModel.Update(msg)
		updatedSessionModel, _ := m.sessionModel.Update(msg)

		m.settingsModel = updatedSettingsModel
		m.sessionModel = updatedSessionModel

		return m, cmd
	}

	m.promptInput, cmd = m.promptInput.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var val string

	settingsStuff := lipgloss.JoinVertical(
		lipgloss.Left,
		m.settingsModel.View(),
		m.sessionModel.View(),
	)

	val = lipgloss.NewStyle().
		Align(lipgloss.Right, lipgloss.Right).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					Width(m.terminalWidth/3*2).
					// this is where we want to render all the messages
					Render(
						m.viewport.View(),
					),
				settingsStuff,
			),
		)

	return lipgloss.NewStyle().Render(
		val,
		m.promptContainer.Render(
			m.promptInput.View(),
		),
	)
}

func (m model) focusedPaneName() string {
	if m.focused == sessionsType {
		return "SESSSION"
	}

	if m.focused == settingsType {
		return "SETTINGS"
	}

	return "PROMPT"
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
	db := other.InitDb()
	err = other.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	p := tea.NewProgram(
		initialModal(db),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	_, err = p.Run()
	if err != nil {
		log.Fatal(err)
	}
}
