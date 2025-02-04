package util

// Defaults
const (
	DefaultSettingsPaneWidth = 20
	DefaultSessionsPaneWidth = 20

	DefaultTerminalWidth  = 120
	DefaultTerminalHeight = 80

	DefaultModelsListWidth  = 10
	DefaultModelsListHeight = 8

	DefaultSessionsListWidth  = 20
	DefaultSessionsListHeight = 5

	DefaultElementsPadding = 2
)

// Panes
const (
	PromptPaneHeight    = 5
	PromptPanePadding   = 2
	PromptPaneMarginTop = 1

	SessionsPaneListTitleMarginLeft = -2

	SettingsPanePadding    = 5
	SettingsPaneHeight     = 12
	SettingsPaneListHeight = 5

	ChatPaneMarginRight = 1
)

// UI elements
const (
	ListRightShiftedItemPadding = -2

	ListItemMarginLeft  = 2
	ListItemPaddingLeft = 2

	WidthMinScalingLimit = 120
)

func CalcPromptPaneSize(tw, th int) (w, h int) {
	return tw - PromptPanePadding, PromptPaneHeight
}

func CalcChatPaneSize(tw, th int, isZenMode bool) (w, h int) {
	if tw < WidthMinScalingLimit {
		isZenMode = true
	}
	// two thirds of terminal width
	paneWidth := tw / 3 * 2

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
	chatPaneWidth, _ := CalcChatPaneSize(tw, th, false)
	settingsPaneWidth := tw - chatPaneWidth - SettingsPanePadding
	return settingsPaneWidth, SettingsPaneHeight
}

func CalcSettingsListSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	chatPaneWidth, _ := CalcChatPaneSize(tw, th, false)
	settingsPaneWidth := tw - chatPaneWidth - SettingsPanePadding
	return settingsPaneWidth, SettingsPaneListHeight
}

func CalcSessionsPaneSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	_, settingsPaneHeight := CalcSettingsPaneSize(tw, th)
	chatPaneWidth, chatPaneHeight := CalcChatPaneSize(tw, th, false)
	sessionsPaneHeight := chatPaneHeight - settingsPaneHeight - SettingsPanePadding
	sessionsPaneWidth := tw - chatPaneWidth - SettingsPanePadding
	return sessionsPaneWidth, sessionsPaneHeight
}

func CalcSessionsListSize(tw, th int) (w, h int) {
	if tw < WidthMinScalingLimit {
		return 0, 0
	}
	_, settingsPaneHeight := CalcSettingsPaneSize(tw, th)
	chatPaneWidth, chatPaneHeight := CalcChatPaneSize(tw, th, false)
	sessionsPaneListHeight := chatPaneHeight - settingsPaneHeight - SettingsPanePadding
	sessionsPaneListWidth := tw - chatPaneWidth - SettingsPanePadding
	return sessionsPaneListWidth, sessionsPaneListHeight
}
