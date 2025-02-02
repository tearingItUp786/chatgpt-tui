package settings

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/util"
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
	terminalWidth  int
	terminalHeight int
	isFocused      bool
	mode           int
	textInput      textinput.Model

	modelPicker components.ModelsList

	list lipgloss.Style

	config       *config.Config
	openAiClient *clients.OpenAiClient
	settings     util.Settings
}

var settingsService *SettingsService

var listHeader = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	MarginLeft(util.ListMarginLeft)

var listItemHeading = lipgloss.NewStyle().
	PaddingLeft(util.ListPaddingLeft).
	Foreground(lipgloss.Color(util.Pink100))

var listItemSpan = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#fff"))

var listItemSpanSelected = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#fffaaa"))

func (m *Model) Init() tea.Cmd {
	return nil
}

func listItemRenderer(heading string, value string) string {
	headingEl := listItemHeading.Render
	spanEl := listItemSpan.Render

	return headingEl(" "+heading, spanEl(value))
}

func (m Model) View() string {
	editForm := ""
	if m.mode == modelMode {
		editForm = m.modelPicker.View()
		return m.list.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				listHeader.Render("Settings"),
				editForm,
			),
		)
	}
	if m.mode != viewMode && m.mode != modelMode {
		editForm = m.textInput.View()
	}

	return m.list.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader.Render("Settings"),
			lipgloss.NewStyle().Height(util.SettingsPaneListHeight).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					listItemRenderer("model", m.settings.Model),
					listItemRenderer("frequency", fmt.Sprint(m.settings.Frequency)),
					listItemRenderer("max_tokens", fmt.Sprint((m.settings.MaxTokens))),
				),
			),
			editForm,
		),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case util.OurWindowResize:
		util.Log("our Window resized", msg.Width)
		width, _ := util.CalcSettingsListSize(m.terminalWidth, m.terminalHeight)
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
		m.terminalHeight = msg.Height

	case tea.KeyMsg:
		// in order to do proper event bubbling, we don't actually want to handle
		// any keyboard events, unless we're the focused pane.
		if m.isFocused {
			if m.mode == viewMode {
				cmd = m.handleViewMode(msg)
				cmds = append(cmds, cmd)
			} else if m.mode == modelMode {
				cmd = m.handleModelMode(msg)
				cmds = append(cmds, cmd)
			} else {
				cmd = m.handleEditMode(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleModelMode(msg tea.KeyMsg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	var settingsChanged UpdateSettingsEvent
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = viewMode
		return cmd

	case tea.KeyEnter:
		i, ok := m.modelPicker.GetSelectedItem()
		if ok {
			m.settings.Model = string(i)
			m.mode = viewMode
			settingsChanged.Settings = m.settings

			newSettings, err := settingsService.UpdateSettings(m.settings)
			if err != nil {
				return util.MakeErrorMsg(err.Error())
			}

			m.settings = newSettings
			m.mode = viewMode
			cmd = MakeSettingsUpdateMsg(m.settings)
			cmds = append(cmds, cmd)
		}
	}

	m.modelPicker, cmd = m.modelPicker.Update(msg)
	cmds = append(cmds, cmd)
	return tea.Batch(cmd, func() tea.Msg { return settingsChanged })
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
				modelsResponse := m.openAiClient.RequestModelsList()
				m.updateModelsList(modelsResponse)

			case "f":
				m.mode = frequencyMode
				m.textInput.Placeholder = "Enter Frequency Number"
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
				m.textInput.Placeholder = "Enter Max Tokens"
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

	switch msg.Type {
	case tea.KeyEsc:
		m.mode = viewMode
		return cmd

	case tea.KeyEnter:
		inputValue := m.textInput.Value()

		if inputValue == "" {
			return cmd
		}

		switch m.mode {
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

func (m *Model) updateModelsList(models clients.ProcessModelsResponse) {
	var modelsList []list.Item
	filteredModels := util.GetFilteredModelList(m.config.ChatGPTApiUrl, models.Result.GetModelNames())
	for _, model := range filteredModels {
		modelsList = append(modelsList, components.ModelsListItem(model))
	}

	m.modelPicker.SetItems(modelsList)
}

func New(db *sql.DB, ctx context.Context) Model {
	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}

	settingsService = NewSettingsService(db)
	settings, err := settingsService.GetSettings(ctx, *config)
	if err != nil {
		panic(err)
	}

	listStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), true).
		BorderForeground(util.NormalTabBorderColor).
		Height(util.SettingsPaneHeight)

	modelPicker := components.NewModelsList([]list.Item{components.ModelsListItem(settings.Model)})
	openAiClient := clients.NewOpenAiClient(config.ChatGPTApiUrl, config.SystemMessage)

	return Model{
		terminalWidth: util.DefaultSettingsPaneWidth,
		mode:          viewMode,
		settings:      settings,
		list:          listStyle,
		modelPicker:   modelPicker,
		config:        config,
		openAiClient:  openAiClient,
	}
}
