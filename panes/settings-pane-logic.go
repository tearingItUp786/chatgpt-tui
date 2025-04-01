package panes

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/settings"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

func (p *SettingsPane) handleModelMode(msg tea.KeyMsg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if p.modelPicker.IsFiltering() {
		return tea.Batch(cmds...)
	}

	switch msg.Type {
	case tea.KeyEsc:
		p.mode = viewMode
		return cmd

	case tea.KeyEnter:
		i, ok := p.modelPicker.GetSelectedItem()
		if ok {
			p.settings.Model = string(i.Text)
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

	return tea.Batch(cmds...)
}

func (p *SettingsPane) handleViewMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd

	switch {

	case key.Matches(msg, p.keyMap.changeModel):
		p.loading = true
		return tea.Batch(
			func() tea.Msg { return p.loadModels(p.config.Provider, p.config.ChatGPTApiUrl) },
			p.spinner.Tick)

	case key.Matches(msg, p.keyMap.reset):
		var updErr error
		p.settings, updErr = p.settingsService.ResetToDefault(p.settings)
		if updErr != nil {
			return util.MakeErrorMsg(updErr.Error())
		}
		cmd = settings.MakeSettingsUpdateMsg(p.settings, nil)

	case key.Matches(msg, p.keyMap.editSysPrompt):
		content := ""
		if p.settings.SystemPrompt != nil {
			content = *p.settings.SystemPrompt
		}
		cmd = util.SwitchToEditor(content, util.SystemMessageEditing)

	case key.Matches(msg, p.keyMap.editFrequency):
		p.configureInput("Enter Frequency Number", util.FrequencyValidator, frequencyMode)
	case key.Matches(msg, p.keyMap.editTemp):
		p.configureInput("Enter Temperature Number", util.TemperatureValidator, tempMode)
	case key.Matches(msg, p.keyMap.editTopP):
		p.configureInput("Enter TopP Number", util.TopPValidator, nucleusSamplingMode)
	case key.Matches(msg, p.keyMap.editMaxTokens):
		p.configureInput("Enter Max Tokens", util.MaxTokensValidator, maxTokensMode)
	}

	return cmd
}

func (p *SettingsPane) configureInput(title string, validator func(str string) error, mode int) {
	ti := textinput.New()
	ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(util.DefaultElementsPadding)
	p.textInput = ti
	p.textInput.Focus()
	p.textInput.Placeholder = title
	p.textInput.Validate = validator
	p.mode = mode
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
			value, err := strconv.ParseFloat(inputValue, 8)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid frequency")
			}
			newFreq := float32(value)
			p.settings.Frequency = &newFreq

		case maxTokensMode:
			newTokens, err := strconv.Atoi(inputValue)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid Tokens")
			}
			p.settings.MaxTokens = newTokens

		case tempMode:
			value, err := strconv.ParseFloat(inputValue, 8)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid Temperature")
			}
			temp := float32(value)
			p.settings.Temperature = &temp

		case nucleusSamplingMode:
			value, err := strconv.ParseFloat(inputValue, 8)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid TopP")
			}
			topp := float32(value)
			p.settings.TopP = &topp
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

func (p SettingsPane) loadModels(providerType string, apiUrl string) tea.Msg {
	availableModels, err := p.settingsService.GetProviderModels(providerType, apiUrl)

	if err != nil {
		return util.ErrorEvent{Message: err.Error()}
	}

	return util.ModelsLoaded{Models: availableModels}
}

func (p *SettingsPane) updateModelsList(models []string) {
	var modelsList []list.Item
	for _, model := range models {
		modelsList = append(modelsList, components.ModelsListItem{Text: model})
	}

	w, h := util.CalcModelsListSize(p.terminalWidth, p.terminalHeight)
	p.modelPicker = components.NewModelsList(modelsList, w, h, p.colors)
}
