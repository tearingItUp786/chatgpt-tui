package util

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PrompInputMode int

const (
	PromptInsertMode PrompInputMode = iota
	PromptNormalMode
)

type Pane int
type AsyncDependency int
type Notification int

// fake enum to keep tab of the currently focused pane
const (
	SettingsPane Pane = iota
	SessionsPane
	PromptPane
	ChatPane
)

const (
	SettingsPaneModule AsyncDependency = iota
	Orchestrator
)

const (
	CopiedNotification Notification = iota
	CancelledNotification
)

type ViewMode int

const (
	ZenMode ViewMode = iota
	TextEditMode
	NormalMode
)

var (
	NormalFocusPanes = []Pane{SettingsPane, SessionsPane, PromptPane, ChatPane}
	ZenFocusPanes    = []Pane{PromptPane, ChatPane}
)

func IsFocusAllowed(mode ViewMode, pane Pane, tw int) bool {
	focusPanes := getFocuesPanes(mode, pane, tw)

	if slices.Contains(focusPanes, pane) {
		return true
	}

	return false
}

func GetNewFocusMode(mode ViewMode, currentFocus Pane, tw int) Pane {
	focusPanes := getFocuesPanes(mode, currentFocus, tw)

	for i, v := range focusPanes {
		if v == currentFocus {
			// this allows for correct wrapping over the array.
			// 3 + 1 = 4 / 4 = 0. (we're already at the last spot, so wrap around)
			return focusPanes[(i+1)%len(focusPanes)]
		}
	}

	Log("Current focus not found in mode", currentFocus)
	return currentFocus
}

func getFocuesPanes(mode ViewMode, pane Pane, tw int) []Pane {
	var focusPanes []Pane

	switch mode {
	case NormalMode:
		focusPanes = NormalFocusPanes
		if tw < WidthMinScalingLimit {
			focusPanes = ZenFocusPanes
		}
	case ZenMode:
		focusPanes = ZenFocusPanes
	case TextEditMode:
		focusPanes = ZenFocusPanes
	default:
		focusPanes = []Pane{pane}
	}

	return focusPanes
}

var MotivationalMessage = lipgloss.NewStyle().
	PaddingLeft(1).
	Render("There's something scary about a blank canvas...that's why I'm here 😄!")

type ModelsLoaded struct {
	Models []string
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

type AsyncDependencyReady struct {
	Dependency AsyncDependency
}

func SendAsyncDependencyReadyMsg(dependency AsyncDependency) tea.Cmd {
	return func() tea.Msg {
		return AsyncDependencyReady{Dependency: dependency}
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

type NotificationMsg struct {
	Notification Notification
}

func SendNotificationMsg(notification Notification) tea.Cmd {
	return func() tea.Msg {
		return NotificationMsg{Notification: notification}
	}
}

type CopyLastMsg struct{}

func SendCopyLastMsg() tea.Msg {
	return CopyLastMsg{}
}

type CopyAllMsgs struct{}

func SendCopyAllMsgs() tea.Msg {
	return CopyAllMsgs{}
}

type ViewModeChanged struct {
	Mode ViewMode
}

func SendViewModeChangedMsg(mode ViewMode) tea.Cmd {
	return func() tea.Msg {
		return ViewModeChanged{Mode: mode}
	}
}
