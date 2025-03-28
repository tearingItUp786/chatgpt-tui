package panes

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	tempMode
	nucleusSamplingMode //top_p
	systemPromptMode
)

type settingsKeyMap struct {
	editTemp      key.Binding
	editFrequency key.Binding
	editTopP      key.Binding
	editSysPrompt key.Binding
	editMaxTokens key.Binding
	changeModel   key.Binding
	reset         key.Binding
}

var defaultSettingsKeyMap = settingsKeyMap{
	editTemp:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "change temperature")),
	editFrequency: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "change frequency")),
	editTopP:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "change top_p")),
	editSysPrompt: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "change system prompt")),
	editMaxTokens: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "change max_tokens")),
	changeModel:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "change current model")),
	reset:         key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "reset settings to default")),
}

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
	keyMap          settingsKeyMap

	modelPicker components.ModelsList

	container lipgloss.Style

	initMode  bool
	config    *config.Config
	llmClient util.LlmClient
	settings  util.Settings
}

var settingsService *settings.SettingsService

var settingsListHeader = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	Bold(true).
	MarginLeft(util.ListItemMarginLeft)

var commandTips = lipgloss.NewStyle()
var listItemHeading = lipgloss.NewStyle().
	PaddingLeft(util.ListItemPaddingLeft)

var listItemSpan = lipgloss.NewStyle()
var spinnerStyle = lipgloss.NewStyle()

func (p SettingsPane) listItemRenderer(heading string, value string) string {
	headingEl := listItemHeading.Render
	spanEl := listItemSpan.Copy().Foreground(p.colors.DefaultTextColor).Render

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
	llmClient := clients.ResolveLlmClient(config.Provider, config.ChatGPTApiUrl, config.SystemMessage)

	colors := config.ColorScheme.GetColors()
	listItemSpan = listItemSpan.Copy().Foreground(colors.DefaultTextColor)
	listItemHeading = listItemHeading.Copy().Foreground(colors.MainColor)
	settingsListHeader = settingsListHeader.Copy().Foreground(colors.DefaultTextColor)
	commandTips = list.DefaultStyles().NoItems.Copy().MarginLeft(util.ListItemMarginLeft)
	spinnerStyle = spinnerStyle.Copy().Foreground(colors.AccentColor)
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), true).
		BorderForeground(colors.NormalTabBorderColor)

	spinner := initSpinner()

	return SettingsPane{
		keyMap:          defaultSettingsKeyMap,
		colors:          colors,
		terminalWidth:   util.DefaultTerminalWidth,
		mode:            viewMode,
		container:       containerStyle,
		config:          config,
		llmClient:       llmClient,
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
			models := []list.Item{components.ModelsListItem{Text: msg.Settings.Model}}

			w, h := util.CalcModelsListSize(p.terminalWidth, p.terminalHeight)
			p.modelPicker = components.NewModelsList(models, w, h, p.colors)
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

	if !p.initMode && p.mode == modelMode {
		p.modelPicker, cmd = p.modelPicker.Update(msg)
		cmds = append(cmds, cmd)
	}

	return p, tea.Batch(cmds...)
}

func (p SettingsPane) View() string {
	editForm := ""
	tips := "'ctrl+r' reset to defaults\n's' edit system prompt"
	if p.mode == modelMode {
		return p.container.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				settingsListHeader.Render("Settings"),
				p.modelPicker.View(),
			),
		)
	}

	if p.mode != viewMode && p.mode != modelMode {
		tips = ""
		editForm = p.textInput.View()
	}

	if p.terminalHeight < util.HeightMinScalingLimit {
		tips = ""
	}

	modelName := util.TrimListItem(
		p.settings.Model,
		util.CalcMaxSettingValueSize(p.container.GetWidth()))
	modelRowContent := p.listItemRenderer("(m) model", modelName)
	if p.loading {
		modelRowContent = p.listItemRenderer(p.spinner.View(), "")
	}

	var (
		temp  = "not set"
		top_p = "not set"
	)

	if p.settings.Temperature != nil {
		temp = fmt.Sprint(*p.settings.Temperature)
	}
	if p.settings.TopP != nil {
		top_p = fmt.Sprint(*p.settings.TopP)
	}

	_, h := util.CalcSettingsPaneSize(p.terminalWidth, p.terminalHeight)
	tipsHiehgt := len(strings.Split(tips, "\n"))

	listItemsHight := h - tipsHiehgt

	lowerRows := commandTips.Render(tips) + "\n" + editForm
	if p.terminalHeight < util.HeightMinScalingLimit || p.mode != viewMode {
		lowerRows = editForm
		listItemsHight = h
	}

	return p.container.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			settingsListHeader.Render("Settings"),
			lipgloss.NewStyle().Height(listItemsHight).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					modelRowContent,
					p.listItemRenderer("(f) frequency", fmt.Sprint(p.settings.Frequency)),
					p.listItemRenderer("(t) max_tokens", fmt.Sprint((p.settings.MaxTokens))),
					p.listItemRenderer("(e) temperature", temp),
					p.listItemRenderer("(p) top_p", top_p),
				),
			),
			lowerRows,
		),
	)
}

func (p SettingsPane) AllowFocusChange() bool {
	return p.mode == viewMode
}
