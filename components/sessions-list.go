package components

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(-2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(-2).Foreground(lipgloss.Color(util.Pink200))
	activeItemStyle   = itemStyle.Copy().Foreground(lipgloss.Color(util.Pink300))
)

type SessionListItem struct {
	Id       int
	Text     string
	IsActive bool
}

type SessionsList struct {
	list     list.Model
	choice   string
	quitting bool
}

func (i SessionListItem) FilterValue() string { return "" }

type sessionItemDelegate struct{}

func (d sessionItemDelegate) Height() int  { return 1 }
func (d sessionItemDelegate) Spacing() int { return 0 }
func (d sessionItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	var cmds []tea.Cmd

	return tea.Batch(cmds...)
}

func (d sessionItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(SessionListItem)
	if !ok {
		log.Println("not okay")
		return
	}

	str := fmt.Sprintf("%s", i.Text)

	fn := itemStyle.Render
	selectedRender := selectedItemStyle.Render

	if i.IsActive {
		fn = activeItemStyle.Render
		selectedRender = activeItemStyle.Render
	}

	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedRender("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (l *SessionsList) GetSelectedItem() (SessionListItem, bool) {
	item, ok := l.list.SelectedItem().(SessionListItem)
	return item, ok
}

func (l *SessionsList) SetItems(items []list.Item) {
	l.list.SetItems(items)
}

func (l SessionsList) Update(msg tea.Msg) (SessionsList, tea.Cmd) {
	var cmd tea.Cmd
	l.list, cmd = l.list.Update(msg)
	return l, cmd
}

func NewSessionsList(items []list.Item) SessionsList {
	defaultWidth := 20
	listHeight := 5

	l := list.New(items, sessionItemDelegate{}, defaultWidth, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle

	return SessionsList{
		list: l,
	}
}

func (l *SessionsList) EditListView(terminalHeight int) string {
	l.list.SetHeight(terminalHeight - 22)
	return lipgloss.NewStyle().MaxHeight(terminalHeight - 22).PaddingLeft(2).Render(l.list.View())
}
