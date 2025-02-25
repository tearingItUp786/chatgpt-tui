package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type UpdateSettingsEvent struct {
	Settings util.Settings
	Err      error
}

func MakeSettingsUpdateMsg(s util.Settings, err error) tea.Cmd {
	return func() tea.Msg {
		return UpdateSettingsEvent{Settings: s, Err: err}
	}
}
