package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type UpdateSettingsEvent struct {
	Settings util.Settings
}

func MakeSettingsUpdateMsg(s util.Settings) tea.Cmd {
	return func() tea.Msg {
		return UpdateSettingsEvent{Settings: s}
	}
}
