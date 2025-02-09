package util

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PrompInputMode int

const (
	PromptInsertMode PrompInputMode = iota
	PromptNormalMode
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

func GetNewFocusMode(mode ViewMode, currentFocus FocusPane, tw int) FocusPane {
	var focusModes []FocusPane

	switch mode {
	case NormalMode:
		focusModes = NormalFocusModes

		if tw < WidthMinScalingLimit {
			focusModes = ZenFocusModes
		}
	case ZenMode:
		focusModes = ZenFocusModes
	default:
		Log("Invalid mode")
		return currentFocus
	}

	for i, v := range focusModes {
		if v == currentFocus {
			// this allows for correct wrapping over the array.
			// 3 + 1 = 4 / 4 = 0. (we're already at the last spot, so wrap around)
			return focusModes[(i+1)%len(focusModes)]
		}
	}

	Log("Current focus not found in mode", currentFocus)
	return currentFocus
}

var MotivationalMessage = lipgloss.NewStyle().
	PaddingLeft(1).
	Render("There's something scary about a blank canvas...that's why I'm here 😄!")

type ModelsLoaded struct {
	Models []string
}

type SettingsLoaded struct {
	Settings Settings
	Error    error
}

type ProcessingStateChanged struct {
	IsProcessing bool
}

func SendProcessingStateChangedMsg(isProcessing bool) tea.Cmd {
	return func() tea.Msg {
		return ProcessingStateChanged{IsProcessing: isProcessing}
	}
}

type PromptReady struct {
	Prompt string
}

func SendPromptReadyMsg(prompt string) tea.Cmd {
	return func() tea.Msg {
		return PromptReady{Prompt: prompt}
	}
}

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

type OurWindowResize struct {
	Width int
}

func MakeWindowResizeMsg(w int) tea.Msg {
	return OurWindowResize{Width: w}
}

type CopyLastMsg struct{}

func SendCopyLastMsg() tea.Msg {
	return CopyLastMsg{}
}

type CopyAllMsgs struct{}

func SendCopyAllMsgs() tea.Msg {
	return CopyAllMsgs{}
}
