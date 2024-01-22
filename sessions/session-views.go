package sessions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) container() lipgloss.Style {
	width := (m.terminalWidth / 3) - 5
	borderColor := lipgloss.Color("#bbb")

	if m.IsFocused {
		borderColor = lipgloss.Color("#d70073")
	}

	container := lipgloss.NewStyle().
		AlignVertical(lipgloss.Top).
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(borderColor).
		Height(8).
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

func listItem(heading string, value string) string {
	headingEl := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("#FFC0CB")).
		Render
	spanEl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fff")).
		Render

	return headingEl(" "+heading, spanEl(value))
}

func (m Model) normaListView() string {
	sessionListItems := []string{}
	for _, session := range m.AllSessions {
		sessionListItems = append(
			sessionListItems,
			listItem(fmt.Sprint(session.ID), session.SessionName),
		)
	}

	return strings.Join(sessionListItems, "\n")
}

func initEditListViewTable(sessions []Session) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 2},
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{}
	for _, session := range sessions {
		rows = append(rows, table.Row{
			fmt.Sprint(session.ID),
			session.SessionName,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

func (m *Model) editListView() string {
	return lipgloss.NewStyle().PaddingLeft(2).Render(m.table.View())
}
