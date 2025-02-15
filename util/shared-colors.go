package util

import "github.com/charmbracelet/lipgloss"

type ColorScheme string

const (
	OriginalPink ColorScheme = "Pink"
	SmoothBlue   ColorScheme = "Blue"
)

const (
	GlamourDarkTheme    = "dark"
	GlamourDraculaTheme = "dracula"
)

var (
	pink100   = "#F2B3E8"
	pink200   = "#8C3A87"
	pink300   = "#BD54BF"
	purple    = "#432D59"
	red       = "#DE3163"
	white     = "#FFFFFF"
	lightGrey = "#bbbbbb"
)

var (
	smoothBlue = "#90a0d3"
	pinkYellow = "#e3b89f"
	cyan       = "#c3f7f5"
	lightGreen = "#a0d390"
	blue       = "#6b81c5"
)

type SchemeColors struct {
	MainColor            lipgloss.Color
	AccentColor          lipgloss.Color
	HighlightColor       lipgloss.Color
	DefaultTextColor     lipgloss.Color
	ErrorColor           lipgloss.Color
	NormalTabBorderColor lipgloss.Color
	ActiveTabBorderColor lipgloss.Color
	RendererTheme        string
}

func (s ColorScheme) GetColors() SchemeColors {
	defaultColors := SchemeColors{
		MainColor:            lipgloss.Color(pink100),
		AccentColor:          lipgloss.Color(pink200),
		HighlightColor:       lipgloss.Color(pink300),
		DefaultTextColor:     lipgloss.Color(white),
		ErrorColor:           lipgloss.Color(red),
		NormalTabBorderColor: lipgloss.Color(lightGrey),
		ActiveTabBorderColor: lipgloss.Color(pink300),
		RendererTheme:        GlamourDarkTheme,
	}

	switch s {
	case SmoothBlue:
		return SchemeColors{
			MainColor:            lipgloss.Color(pinkYellow),
			AccentColor:          lipgloss.Color(lightGreen),
			HighlightColor:       lipgloss.Color(blue),
			DefaultTextColor:     lipgloss.Color(white),
			ErrorColor:           lipgloss.Color(red),
			NormalTabBorderColor: lipgloss.Color(smoothBlue),
			ActiveTabBorderColor: lipgloss.Color(pinkYellow),
			RendererTheme:        GlamourDraculaTheme,
		}

	case OriginalPink:
		return defaultColors
	default:
		return defaultColors
	}
}
