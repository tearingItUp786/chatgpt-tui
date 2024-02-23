package sessions

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(-2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(-2).Foreground(lipgloss.Color("170"))
	activeItemStyle   = itemStyle.Copy().Foreground(lipgloss.Color("200"))
)

type item struct {
	id       int
	text     string
	isActive bool
}

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	var cmds []tea.Cmd
	// var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			log.Println("fuck")
		}
	}

	return tea.Batch(cmds...)
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		log.Println("not okay")
		return
	}

	str := fmt.Sprintf("%s", i.text)

	fn := itemStyle.Render
	selectedRender := selectedItemStyle.Render

	if i.isActive {
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

func ConstructListItems(sessions []Session, currentSessionId int) []list.Item {
	items := []list.Item{}

	for _, session := range sessions {
		anItem := item{
			id:       session.ID,
			text:     session.SessionName,
			isActive: session.ID == currentSessionId,
		}
		items = append(items, anItem)
	}

	return items
}

func initEditListViewTable(sessions []Session, currentSessionId int) list.Model {
	defaultWidth := 20
	listHeight := 5
	items := ConstructListItems(sessions, currentSessionId)

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	return l
}

func (m *Model) editListView() string {
	m.list.SetHeight(m.terminalHeight - 18)
	return lipgloss.NewStyle().MaxHeight(m.terminalHeight - 18).PaddingLeft(2).Render(m.list.View())
}
