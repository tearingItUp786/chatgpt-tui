package util

import "github.com/charmbracelet/lipgloss"

const DefaultSettingsId = 0
const DefaultRequestTimeOutSec = 5
const ChunkIndexStart = 1

var DefaultMessage = lipgloss.NewStyle().
	PaddingLeft(1).
	Render("There's something scary about a blank canvas...that's why I'm here 😄!")
