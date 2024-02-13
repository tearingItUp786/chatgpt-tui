package sessions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) settingsContainer() lipgloss.Style {
	width := (m.terminalWidth / 3) - 5
	borderColor := lipgloss.Color("#bbb")

	if m.isFocused {
		borderColor = lipgloss.Color("#d70073")
	}

	container := lipgloss.NewStyle().
		AlignVertical(lipgloss.Top).
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(borderColor).
		Width(width)

	return container
}

func listHeader(str ...string) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		MarginLeft(2).
		Render(str...)
}

func listItem(heading string, value string, isActive bool) string {
	headingColor := "#FFC0CB"
	color := "#bbb"
	if isActive {
		const colorValue = "#E591A6"
		color = colorValue
		headingColor = colorValue
	}
	headingEl := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color(headingColor)).
		Render
	spanEl := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render

	return headingEl(" "+heading, spanEl(value))
}

func (m Model) normaListView() string {
	sessionListItems := []string{}
	for _, session := range m.AllSessions {
		isCurrentSession := m.CurrentSessionID == session.ID
		sessionListItems = append(
			sessionListItems,
			listItem(fmt.Sprint(session.ID), session.SessionName, isCurrentSession),
		)
	}

	return lipgloss.NewStyle().
		Height(m.terminalHeight - 18).
		MaxHeight(m.terminalHeight - 18).
		Render(strings.Join(sessionListItems, "\n"))
}
