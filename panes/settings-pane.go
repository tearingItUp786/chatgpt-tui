package panes

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/settings"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const (
	viewMode  = -1
	modelMode = iota
	maxTokensMode
	frequencyMode
)

const (
	ModelPickerKey = "m"
	FrequencyKey   = "f"
	MaxTokensKey   = "t"
)

type SettingsPane struct {
	terminalWidth   int
	terminalHeight  int
	isFocused       bool
	mode            int
	textInput       textinput.Model
	settingsService *settings.SettingsService
	spinner         spinner.Model
	loading         bool
	colors          util.SchemeColors

	modelPicker components.ModelsList

	container lipgloss.Style

	initMode     bool
	config       *config.Config
	openAiClient *clients.OpenAiClient
	settings     util.Settings
}

var settingsService *settings.SettingsService

var settingsListHeader = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	Bold(true).
	MarginLeft(util.ListItemMarginLeft)

var listItemHeading = lipgloss.NewStyle().
	PaddingLeft(util.ListItemPaddingLeft)

var listItemSpan = lipgloss.NewStyle()
var spinnerStyle = lipgloss.NewStyle()

func listItemRenderer(heading string, value string) string {
	headingEl := listItemHeading.Render
	spanEl := listItemSpan.Render

	return headingEl("■ "+heading, spanEl(value))
}

func initSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = spinnerStyle

	return s
}

func NewSettingsPane(db *sql.DB, ctx context.Context) SettingsPane {
	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}

	settingsService = settings.NewSettingsService(db)
	openAiClient := clients.NewOpenAiClient(config.ChatGPTApiUrl, config.SystemMessage)

	colors := config.ColorScheme.GetColors()
	listItemSpan = listItemSpan.Copy().Foreground(colors.DefaultTextColor)
	listItemHeading = listItemHeading.Copy().Foreground(colors.MainColor)
	settingsListHeader = settingsListHeader.Copy().Foreground(colors.DefaultTextColor)
	spinnerStyle = spinnerStyle.Copy().Foreground(colors.AccentColor)
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), true).
		BorderForeground(colors.NormalTabBorderColor)

	spinner := initSpinner()

	return SettingsPane{
		colors:          colors,
		terminalWidth:   util.DefaultTerminalWidth,
		mode:            viewMode,
		container:       containerStyle,
		config:          config,
		openAiClient:    openAiClient,
		settingsService: settingsService,
		spinner:         spinner,
		initMode:        true,
		loading:         true,
	}
}

func (p *SettingsPane) Init() tea.Cmd {
	settingsLoader := func() tea.Msg { return p.settingsService.GetSettings(nil, *p.config) }
	return tea.Batch(p.spinner.Tick, settingsLoader)
}

func (p SettingsPane) Update(msg tea.Msg) (SettingsPane, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case util.FocusEvent:
		p.isFocused = msg.IsFocused
		p.mode = viewMode

		borderColor := p.colors.NormalTabBorderColor
		if p.isFocused {
			borderColor = p.colors.ActiveTabBorderColor
		}
		p.container.BorderForeground(borderColor)

		return p, nil

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height
		w, h := util.CalcSettingsPaneSize(p.terminalWidth, p.terminalHeight)
		p.container.Width(w).Height(h)

	case spinner.TickMsg:
		p.spinner, cmd = p.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case util.ErrorEvent:
		p.loading = false
		p.mode = viewMode

	case settings.UpdateSettingsEvent:
		if p.initMode {
			p.settings = msg.Settings
			w, h := util.CalcModelsListSize(p.terminalWidth, p.terminalHeight)
			p.modelPicker = components.NewModelsList([]list.Item{components.ModelsListItem(msg.Settings.Model)}, w, h, p.colors)
			p.initMode = false
			p.loading = false

			cmds = append(cmds, util.SendAsyncDependencyReadyMsg(util.SettingsPaneModule))
		}

	case util.ModelsLoaded:
		p.loading = false
		p.mode = modelMode
		p.updateModelsList(msg.Models)

	case tea.KeyMsg:
		if p.initMode {
			break
		}

		if p.isFocused {
			if p.mode == viewMode {
				cmd = p.handleViewMode(msg)
				cmds = append(cmds, cmd)
			} else if p.mode == modelMode {
				cmd = p.handleModelMode(msg)
				cmds = append(cmds, cmd)
			} else {
				cmd = p.handleEditMode(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return p, tea.Batch(cmds...)
}

func (p SettingsPane) View() string {
	editForm := ""
	if p.mode == modelMode {
		return p.container.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				settingsListHeader.Render("Settings"),
				p.modelPicker.View(),
			),
		)
	}

	if p.mode != viewMode && p.mode != modelMode {
		editForm = p.textInput.View()
	}

	modelRowContent := listItemRenderer("model", p.settings.Model)
	if p.loading {
		modelRowContent = listItemRenderer(p.spinner.View(), "")
	}

	_, h := util.CalcSettingsPaneSize(p.terminalWidth, p.terminalHeight)
	return p.container.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			settingsListHeader.Render("Settings"),
			lipgloss.NewStyle().Height(h).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					modelRowContent,
					listItemRenderer("frequency", fmt.Sprint(p.settings.Frequency)),
					listItemRenderer("max_tokens", fmt.Sprint((p.settings.MaxTokens))),
				),
			),
			editForm,
		),
	)
}

func (p *SettingsPane) handleModelMode(msg tea.KeyMsg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg.Type {
	case tea.KeyEsc:
		p.mode = viewMode
		return cmd

	case tea.KeyEnter:
		i, ok := p.modelPicker.GetSelectedItem()
		if ok {
			p.settings.Model = string(i)
			p.mode = viewMode

			var updateError error
			p.settings, updateError = settingsService.UpdateSettings(p.settings)
			if updateError != nil {
				return util.MakeErrorMsg(updateError.Error())
			}

			cmd = settings.MakeSettingsUpdateMsg(p.settings, nil)
			cmds = append(cmds, cmd)
		}
	}

	p.modelPicker, cmd = p.modelPicker.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (p *SettingsPane) handleViewMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch msg.Type {
	case tea.KeyRunes:
		key := string(msg.Runes)

		if key == ModelPickerKey || key == FrequencyKey || key == MaxTokensKey {
			ti := textinput.New()
			ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(util.DefaultElementsPadding)
			p.textInput = ti

			switch key {
			case ModelPickerKey:
				p.loading = true
				return tea.Batch(
					func() tea.Msg { return p.loadModels(p.config.ChatGPTApiUrl) },
					p.spinner.Tick)

			case FrequencyKey:
				p.mode = frequencyMode
				p.textInput.Placeholder = "Enter Frequency Number"
				p.textInput.Validate = func(str string) error {
					if _, err := strconv.ParseFloat(str, 64); err == nil {
						log.Printf("'%s' is a floating-point number.\n", str)
					} else {
						log.Printf("'%s' is not a floating-point number.\n", str)
						return err
					}

					return nil
				}
			case MaxTokensKey:
				p.textInput.Placeholder = "Enter Max Tokens"
				p.mode = maxTokensMode
			}

			p.textInput.Focus()
		}
	}

	return cmd
}

func (p *SettingsPane) handleEditMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	p.textInput, cmd = p.textInput.Update(msg)

	switch msg.Type {
	case tea.KeyEsc:
		p.mode = viewMode
		return cmd

	case tea.KeyEnter:
		inputValue := p.textInput.Value()

		if inputValue == "" {
			return cmd
		}

		switch p.mode {
		case frequencyMode:
			newFreq, err := strconv.Atoi(inputValue)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid frequency")
			}
			p.settings.Frequency = newFreq

		case maxTokensMode:
			newTokens, err := strconv.Atoi(inputValue)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid Tokens")
			}
			p.settings.MaxTokens = newTokens
		}

		newSettings, err := settingsService.UpdateSettings(p.settings)
		if err != nil {
			return util.MakeErrorMsg(err.Error())
		}

		p.settings = newSettings
		p.mode = viewMode
		cmd = settings.MakeSettingsUpdateMsg(p.settings, nil)
	}

	return cmd
}

func (p SettingsPane) loadModels(apiUrl string) tea.Msg {
	availableModels, err := p.settingsService.GetProviderModels(apiUrl)

	if err != nil {
		return util.ErrorEvent{Message: err.Error()}
	}

	return util.ModelsLoaded{Models: availableModels}
}

func (p *SettingsPane) updateModelsList(models []string) {
	var modelsList []list.Item
	for _, model := range models {
		modelsList = append(modelsList, components.ModelsListItem(model))
	}

	w, h := util.CalcModelsListSize(p.terminalWidth, p.terminalHeight)
	p.modelPicker = components.NewModelsList(modelsList, w, h, p.colors)
}
