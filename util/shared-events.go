package util

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FocusPane int

// fake enum to keep tab of the currently focused pane
const (
	SettingsType FocusPane = iota
	SessionsType
	PromptType
	ChatMessagesType
)

type ViewMode int

const (
	ZenMode ViewMode = iota
	NormalMode
)

var (
	NormalFocusModes = []FocusPane{SettingsType, SessionsType, PromptType, ChatMessagesType}
	ZenFocusModes    = []FocusPane{PromptType, ChatMessagesType}
)

func GetNewFocusMode(mode ViewMode, currentFocus FocusPane) FocusPane {
	var focusModes []FocusPane

	switch mode {
	case NormalMode:
		focusModes = NormalFocusModes
	case ZenMode:
		focusModes = ZenFocusModes
	default:
		log.Println("Invalid mode")
		return currentFocus
	}

	for i, v := range focusModes {
		if v == currentFocus {
			// this allows for correct wrapping over the array.
			// 3 + 1 = 4 / 4 = 0. (we're already at the last spot, so wrap around)
			return focusModes[(i+1)%len(focusModes)]
		}
	}

	log.Println("Current focus not found in mode")
	return currentFocus
}

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
