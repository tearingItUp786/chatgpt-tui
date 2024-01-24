package sessions

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) container() lipgloss.Style {
	width := (m.terminalWidth / 3) - 5
	borderColor := lipgloss.Color("#bbb")

	if m.isFocused {
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

	return strings.Join(sessionListItems, "\n")
}

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(-2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(-2).Foreground(lipgloss.Color("170"))
)

type item struct {
	id   int
	text string
}

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s", i.text)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func initEditListViewTable(sessions []Session, currentSessionId int) list.Model {
	defaultWidth := 20
	listHeight := 5
	items := []list.Item{}

	for _, session := range sessions {
		anItem := item{
			id:   session.ID,
			text: session.SessionName,
		}
		items = append(items, anItem)
	}

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	return l
}

func (m *Model) editListView() string {
	return lipgloss.NewStyle().PaddingLeft(2).Render(m.list.View())
}
