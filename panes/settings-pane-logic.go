package panes

import (
	"slices"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/settings"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

var editModeKeys = []string{
	ModelPickerKey, FrequencyKey, MaxTokensKey, TempKey, TopPKey,
}

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

	switch msg.Type {
	case tea.KeyRunes:
		key := string(msg.Runes)

		if slices.Contains(editModeKeys, key) {
			ti := textinput.New()
			ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(util.DefaultElementsPadding)
			p.textInput = ti

			switch key {
			case ModelPickerKey:
				p.loading = true
				return tea.Batch(
					func() tea.Msg { return p.loadModels(p.config.Provider, p.config.ChatGPTApiUrl) },
					p.spinner.Tick)

			case FrequencyKey:
				p.textInput.Placeholder = "Enter Frequency Number"
				p.textInput.Validate = util.FrequencyValidator
				p.mode = frequencyMode
			case TempKey:
				p.textInput.Placeholder = "Enter Temperature Number"
				p.textInput.Validate = util.TemperatureValidator
				p.mode = tempMode
			case TopPKey:
				p.textInput.Placeholder = "Enter TopP Number"
				p.textInput.Validate = util.TopPValidator
				p.mode = nucleusSamplingMode
			case MaxTokensKey:
				p.textInput.Placeholder = "Enter Max Tokens"
				p.textInput.Validate = util.MaxTokensValidator
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
			value, err := strconv.ParseFloat(inputValue, 8)
			if err != nil {
				cmd = util.MakeErrorMsg("Invalid frequency")
			}
			newFreq := float32(value)
			p.settings.Frequency = newFreq

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
