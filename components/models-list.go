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
	list list.Model
}

var listItemSpan = lipgloss.NewStyle().
	PaddingLeft(util.ListItemPaddingLeft).
	Foreground(lipgloss.Color(util.White))

var listItemSpanSelected = lipgloss.NewStyle().
	PaddingLeft(util.ListItemPaddingLeft).
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
	str = util.TrimListItem(str, m.Width())

	fn := listItemSpan.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			row := "> " + strings.Join(s, " ")
			return listItemSpanSelected.Render(row)
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

func NewModelsList(items []list.Item, w, h int) ModelsList {
	newList := list.New(items, modelItemDelegate{}, w, h)

	newList.SetStatusBarItemName("model detected", "models detected")
	newList.SetShowTitle(false)
	newList.SetShowHelp(false)
	newList.SetFilteringEnabled(false)
	newList.DisableQuitKeybindings()

	newList.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.Color(util.Pink300)).Render("•")
	newList.Paginator.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.Color(util.White)).Render("•")

	return ModelsList{
		list: newList,
	}
}
