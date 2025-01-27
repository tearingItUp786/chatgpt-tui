package components

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type ModelsList struct {
	list     list.Model
	choice   string
	quitting bool
}

var listItemSpan = lipgloss.NewStyle().
	PaddingLeft(2).
	Foreground(lipgloss.Color(util.White))

var listItemSpanSelected = lipgloss.NewStyle().
	PaddingLeft(2).
	Foreground(lipgloss.Color(util.Pink200))

type ModelsListItem string

func (i ModelsListItem) FilterValue() string { return "" }

type modelItemDelegate struct{}

func (d modelItemDelegate) Height() int                             { return 1 }
func (d modelItemDelegate) Spacing() int                            { return 0 }
func (d modelItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d modelItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(ModelsListItem)
	if !ok {
		return
	}
	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := listItemSpan.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return listItemSpanSelected.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (l *ModelsList) View() string {
	return l.list.View()
}

func (l *ModelsList) GetSelectedItem() (ModelsListItem, bool) {
	item, ok := l.list.SelectedItem().(ModelsListItem)
	return item, ok
}

func (l ModelsList) Update(msg tea.Msg) (ModelsList, tea.Cmd) {
	var cmd tea.Cmd
	l.list, cmd = l.list.Update(msg)
	return l, cmd
}

func (l *ModelsList) SetItems(items []list.Item) {
	l.list.SetItems(items)
}

func NewModelsList(items []list.Item) ModelsList {
	newList := list.New(items, modelItemDelegate{}, 10, 8)
	newList.SetStatusBarItemName("model detected", "models detected")
	newList.SetShowTitle(false)
	newList.SetShowHelp(false)
	newList.SetFilteringEnabled(false)

	return ModelsList{
		list: newList,
	}
}
