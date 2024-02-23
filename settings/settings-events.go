package settings

import tea "github.com/charmbracelet/bubbletea"

type UpdateSettingsEvent struct {
	Settings Settings
}

func MakeSettingsUpdateMsg(s Settings) tea.Cmd {
	return func() tea.Msg {
		return UpdateSettingsEvent{Settings: s}
	}
}
