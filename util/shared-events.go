package util

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fake enum to keep tab of the currently focused pane
const (
	SettingsType = iota
	SessionsType
	PromptType
	ChatMessagesType
)

var MotivationalMessage = lipgloss.NewStyle().
	PaddingLeft(1).
	Render("There's something scary about a blank canvas...that's why I'm here 😼!")

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
