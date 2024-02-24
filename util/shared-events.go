package util

import tea "github.com/charmbracelet/bubbletea"

// fake enum to keep tab of the currently focused pane
const (
	settingsType = iota
	sessionsType
	promptType
	chatMessagesType
)

type FocusEvent struct {
	IsFocused bool
}

func MakeFocusMsg(v bool) tea.Msg {
	return FocusEvent{IsFocused: v}
}

type ErrorEvent struct {
	Message string
}

func MakeErrorMsg(v string) tea.Cmd {
	return func() tea.Msg {
		return ErrorEvent{Message: v}
	}
}
