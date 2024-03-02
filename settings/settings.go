package settings

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/golang-tui/util"
)

const (
	viewMode  = -1
	modelMode = iota
	maxTokensMode
	frequencyMode
)

// set up a text input model  that only renders if it is not viewMode
// based on the mode we are in, that is the column we will update
// in the sql database. After a successful save, we are going to
// go back to view mode.
type Model struct {
	terminalWidth int
	isFocused     bool
	mode          int
	settings      Settings
	textInput     textinput.Model

	list lipgloss.Style
}

var settingsService *SettingsService

func (m Model) Init() tea.Cmd {
	return nil
}

func listItem(heading string, value string) string {
	headingEl := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color(util.Pink100)).
		Render
	spanEl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fff")).
		Render

	return headingEl(" "+heading, spanEl(value))
}

func (m Model) View() string {
	listHeader := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		MarginLeft(2).
		Render

	editForm := ""
	if m.mode != viewMode {
		editForm = m.textInput.View()
	}
	return m.list.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader("Settings"),
			listItem("model", m.settings.Model),
			listItem("frequency", fmt.Sprint(m.settings.Frequency)),
			listItem("max_tokens", fmt.Sprint((m.settings.MaxTokens))),
			editForm,
		),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case util.OurWindowResize:
		width := m.terminalWidth - msg.Width - 5
		m.list.Width(width)

	case util.FocusEvent:
		m.isFocused = msg.IsFocused
		m.mode = viewMode

		borderColor := util.NormalTabBorderColor
		if m.isFocused {
			borderColor = util.ActiveTabBorderColor
		}
		m.list.BorderForeground(borderColor)

		return m, nil
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		return m, nil
	case tea.KeyMsg:
		// in order to do proper event bubbling, we don't actually want to handle
		// any keyboard events, unless we're the focused pane.
		if !m.isFocused {
			return m, nil
		}

		if m.mode == viewMode {
			cmd = m.handleViewMode(msg)
			cmds = append(cmds, cmd)
		} else {
			cmd = m.handleEditMode(msg)
			cmds = append(cmds, cmd)
		}

	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleViewMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch msg.Type {
	case tea.KeyRunes:
		key := string(msg.Runes)

		if key == "m" || key == "f" || key == "t" {
			ti := textinput.New()
			ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(2)
			m.textInput = ti

			switch key {
			case "m":
				m.mode = modelMode
				m.textInput.CharLimit = 100
			case "f":
				m.mode = frequencyMode
				// the validate function will not allow us to type in any characters
				// that don't pass validation. So here, we are ensuring that we do not allow
				// the user to type in any non numeric characters. We can extend this further,
				// maybe...
				m.textInput.Validate = func(str string) error {
					if _, err := strconv.ParseFloat(str, 64); err == nil {
						log.Printf("'%s' is a floating-point number.\n", str)
					} else {
						log.Printf("'%s' is not a floating-point number.\n", str)
						return err
					}

					return nil
				}
			case "t":
				m.mode = maxTokensMode
			}

			m.textInput.Focus()
		}
	}

	return cmd
}

func (m *Model) handleEditMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	if msg.String() == "enter" {
		inputValue := m.textInput.Value()
		if inputValue == "" {
			return cmd
		}

		switch m.mode {
		case modelMode:
			m.settings.Model = inputValue

		case frequencyMode:
			newFreq, err := strconv.Atoi(inputValue)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid frequency")
			}
			m.settings.Frequency = newFreq

		case maxTokensMode:
			newTokens, err := strconv.Atoi(inputValue)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid Tokens")
			}
			m.settings.MaxTokens = newTokens
		}

		newSettings, err := settingsService.UpdateSettings(m.settings)
		if err != nil {
			return util.MakeErrorMsg(err.Error())
		}

		m.settings = newSettings
		m.mode = viewMode
		cmd = MakeSettingsUpdateMsg(m.settings)
	}

	return cmd
}

func New(db *sql.DB) Model {
	settingsService = NewSettingsService(db)
	settings, err := settingsService.GetSettings()
	if err != nil {
		panic(err)
	}

	list := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), true).
		BorderForeground(util.NormalTabBorderColor).
		Height(8)

	return Model{
		terminalWidth: 20,
		mode:          viewMode,
		settings:      settings,
		list:          list,
	}
}
