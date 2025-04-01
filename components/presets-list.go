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

type PresetsList struct {
	list list.Model
}

var presetItemSpan = lipgloss.NewStyle().
	PaddingLeft(util.ListItemPaddingLeft)

var presetItemSpanSelected = lipgloss.NewStyle().
	PaddingLeft(util.ListItemPaddingLeft)

type PresetsListItem struct {
	Id   int
	Text string
}

func (i PresetsListItem) FilterValue() string { return i.Text }

type presetsItemDelegate struct{}

func (d presetsItemDelegate) Height() int                             { return 1 }
func (d presetsItemDelegate) Spacing() int                            { return 0 }
func (d presetsItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d presetsItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(PresetsListItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Text)
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

func (l *PresetsList) View() string {
	if l.list.FilterState() == list.Filtering {
		l.list.SetShowStatusBar(false)
	} else {
		l.list.SetShowStatusBar(true)
	}
	return l.list.View()
}

func (l *PresetsList) GetSelectedItem() (PresetsListItem, bool) {
	item, ok := l.list.SelectedItem().(PresetsListItem)
	return item, ok
}

func (l PresetsList) IsFiltering() bool {
	return l.list.SettingFilter()
}

func (l PresetsList) Update(msg tea.Msg) (PresetsList, tea.Cmd) {
	var cmd tea.Cmd
	l.list, cmd = l.list.Update(msg)
	return l, cmd
}

func NewPresetsList(items []list.Item, w, h int, colors util.SchemeColors) PresetsList {
	l := list.New(items, presetsItemDelegate{}, w, h)

	l.SetStatusBarItemName("preset found", "presets found")
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	l.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(colors.HighlightColor).Render("■")
	l.Paginator.InactiveDot = lipgloss.NewStyle().Foreground(colors.DefaultTextColor).Render("•")
	listItemSpan = listItemSpan.Copy().Foreground(colors.DefaultTextColor)
	listItemSpanSelected = listItemSpanSelected.Copy().Foreground(colors.AccentColor)
	l.FilterInput.PromptStyle = l.FilterInput.PromptStyle.Copy().Foreground(colors.ActiveTabBorderColor).PaddingBottom(0).Margin(0)
	l.FilterInput.Cursor.Style = l.FilterInput.Cursor.Style.Copy().Foreground(colors.NormalTabBorderColor)

	return PresetsList{
		list: l,
	}
}
