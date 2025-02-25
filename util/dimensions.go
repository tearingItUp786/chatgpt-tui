package util

import "math"

// Defaults
const (
	DefaultTerminalWidth  = 120
	DefaultTerminalHeight = 80

	DefaultElementsPadding = 2
)

// Panes
const (
	PromptPaneHeight    = 5
	PromptPanePadding   = 2
	PromptPaneMarginTop = 1

	ChatPaneMarginRight = 1
	SidePaneLeftPadding = 5

	// A 'counterweight' is a sum of other elements' margins and paggings
	// The counterweight needs to be subtracted when calculating pane sizes
	// in order to properly align elements
	SettingsPaneHeightCounterweight = 3
	SessionsPaneHeightCounterweight = 5
)

// UI elements
const (
	ListRightShiftedItemPadding = -2

	ListItemMarginLeft  = 2
	ListItemPaddingLeft = 2

	WidthMinScalingLimit = 120

	ListItemTrimThreshold  = 10
	ListItemTrimCharAmount = 14
)

/*
Pane sizes are calculated with proportions:
- Prompt pane:
  - Width: full termial witdh minus paddings
  - Height: a constant for height and a constant for top margin

- Chat pane:
  - Width: takes 2/3 of the terminal width
  - Height: full terminal height minus the prompt pane height

- Settings pane:
  - Width: takes 1/3 of the terminal width, minus paddings
  - Height: takes 1/3 of the chat pane height, minus paddings

- Sessions pane:
  - Width: takes 1/3 of the terminal width, minus paddings
  - Height: takes 2/3 of the chat pane height, minus paddings
*/

func twoThirds(reference int) int {
	return int(math.Round(float64(reference) * (2.0 / 3.0)))
}

func oneThird(reference int) int {
	return int(math.Round(float64(reference) / 3.0))
}

func ensureNonNegative(number int) int {
	if number < 0 {
		return 0
	}
	return number
}

func CalcPromptPaneSize(tw, th int) (w, h int) {
	return tw - PromptPanePadding, PromptPaneHeight
}

func CalcChatPaneSize(tw, th int, isZenMode bool) (w, h int) {
	if tw < WidthMinScalingLimit {
		isZenMode = true
	}

	paneWidth := twoThirds(tw)
	if isZenMode {
		paneWidth = tw - DefaultElementsPadding
	}

	paneHeight := th - PromptPaneHeight
	return paneWidth, paneHeight
}

func CalcSettingsPaneSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	_, chatPaneHeight := CalcChatPaneSize(tw, th, false)
	settingsPaneWidth := oneThird(tw) - SidePaneLeftPadding
	settingsPaneHeight := oneThird(chatPaneHeight) - SettingsPaneHeightCounterweight

	settingsPaneWidth = ensureNonNegative(settingsPaneWidth)
	settingsPaneHeight = ensureNonNegative(settingsPaneHeight)

	return settingsPaneWidth, settingsPaneHeight
}

func CalcModelsListSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	settingsPaneWidth, settingsPaneHeight := CalcSettingsPaneSize(tw, th)
	modelsListWidth := settingsPaneWidth - DefaultElementsPadding
	modelsListHeight := settingsPaneHeight + 1

	modelsListWidth = ensureNonNegative(modelsListWidth)
	modelsListHeight = ensureNonNegative(modelsListHeight)

	return modelsListWidth, modelsListHeight
}

func CalcSessionsPaneSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	_, chatPaneHeight := CalcChatPaneSize(tw, th, false)
	sessionsPaneWidth := oneThird(tw) - SidePaneLeftPadding
	sessionsPaneHeight := twoThirds(chatPaneHeight) - SessionsPaneHeightCounterweight

	sessionsPaneWidth = ensureNonNegative(sessionsPaneWidth)
	sessionsPaneHeight = ensureNonNegative(sessionsPaneHeight)

	return sessionsPaneWidth, sessionsPaneHeight
}

func CalcSessionsListSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	_, chatPaneHeight := CalcChatPaneSize(tw, th, false)
	sessionsPaneListWidth := oneThird(tw) - SidePaneLeftPadding
	sessionsPaneListHeight := twoThirds(chatPaneHeight) - SessionsPaneHeightCounterweight

	sessionsPaneListWidth = ensureNonNegative(sessionsPaneListWidth)
	sessionsPaneListHeight = ensureNonNegative(sessionsPaneListHeight)

	return sessionsPaneListWidth, sessionsPaneListHeight
}

func TrimListItem(value string, listWidth int) string {
	threshold := ListItemTrimThreshold
	if listWidth-threshold > 0 {
		trimTo := listWidth - ListItemTrimCharAmount
		if listWidth-threshold < len(value) {
			value = value[0:trimTo] + "..."
		}
	}

	return value
}
