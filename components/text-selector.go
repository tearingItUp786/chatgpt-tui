package components

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/util"
	"golang.design/x/clipboard"
)

const (
	HighlightPrefix = " +"
	CursorSymbol    = "~"
)

type KeyMap struct {
	VisualLineMode key.Binding
	Up             key.Binding
	Down           key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	Copy           key.Binding
	Bottom         key.Binding
	Top            key.Binding
}

var DefaultKeyMap = KeyMap{
	VisualLineMode: key.NewBinding(key.WithKeys("V", "v", tea.KeySpace.String()), key.WithHelp("V, v, <space>", "visual line mode")),
	Up:             key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	Down:           key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	PageUp:         key.NewBinding(key.WithKeys("ctrl+u", "u"), key.WithHelp("ctrl+u", "move up a page")),
	PageDown:       key.NewBinding(key.WithKeys("ctrl+d", "d"), key.WithHelp("ctrl+d", "move down a page")),
	Copy:           key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy selection")),
	Bottom:         key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "go to bottom")),
	Top:            key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "go to top")),
}

type Cursor struct {
	line int
}

type Selection struct {
	Active bool
	anchor Cursor
}

type TextSelector struct {
	Selection    Selection
	lines        []string
	cursor       Cursor
	scrollOffset int
	paneHeight   int
	paneWidth    int
	keys         KeyMap
	renderedText string
	colors       util.SchemeColors

	numberLines int
}

func (s TextSelector) Init() tea.Cmd {
	return nil
}

func (s TextSelector) Update(msg tea.Msg) (TextSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		paneWidth, paneHeight := util.CalcVisualModeViewSize(msg.Width, msg.Height)
		s.paneHeight = paneHeight
		s.paneWidth = paneWidth
		s.AdjustScroll()
	case tea.KeyMsg:

		keypress := msg.String()
		if number, err := strconv.Atoi(keypress); err == nil {
			return s.handleLineJumps(keypress, number), nil
		}

		switch {

		case key.Matches(msg, s.keys.PageUp):
			upLines := s.paneHeight / 2
			s.cursor.line = max(s.cursor.line-upLines, s.firstLinePosition())
			s.AdjustScroll()

		case key.Matches(msg, s.keys.Up):
			s = s.handleKeyUp()

		case key.Matches(msg, s.keys.Down):
			s = s.handleKeyDown()

		case key.Matches(msg, s.keys.PageDown):
			downLines := s.paneHeight / 2
			s.cursor.line = min(s.cursor.line+downLines, s.lastLinePosition())
			s.AdjustScroll()

		case key.Matches(msg, s.keys.Bottom):
			s.cursor.line = s.lastLinePosition()
			s.AdjustScroll()

		case key.Matches(msg, s.keys.Top):
			s.cursor.line = s.firstLinePosition()
			s.AdjustScroll()

		case key.Matches(msg, s.keys.VisualLineMode):
			if s.Selection.Active {
				s.Selection.Active = false
			} else {
				s.Selection.Active = true
				s.Selection.anchor = s.cursor
			}

		case key.Matches(msg, s.keys.Copy):
			if s.Selection.Active {
				s.copySelectedLinesToClipboard()
				s.Selection.Active = false
			}
		}
	}

	return s, nil
}

func (s *TextSelector) AdjustScroll() {
	if s.cursor.line < s.scrollOffset {
		s.scrollOffset = s.cursor.line - 1
	} else if s.cursor.line >= s.scrollOffset+s.paneHeight {
		s.scrollOffset = s.cursor.line - s.paneHeight + 1
	}
}

func (s TextSelector) View() string {
	return s.renderLines()
}

func (s TextSelector) renderLines() string {
	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF")).
		Background(s.colors.HighlightColor)

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF")).
		Background(s.colors.AccentColor)

	output := ""
	start := s.scrollOffset
	end := min(start+s.paneHeight, len(s.lines))
	for i := start; i < end; i++ {
		line := s.lines[i]
		if s.Selection.Active {
			startLine := s.Selection.anchor.line
			endLine := s.cursor.line

			if startLine > endLine {
				startLine, endLine = endLine, startLine
			}

			if i >= startLine && i <= endLine {
				output += highlightStyle.Render(HighlightPrefix) + fmt.Sprintf("%s\n", line)
			} else {
				output += line + "\n"
			}

		} else if i == s.cursor.line {
			output += cursorStyle.Render(CursorSymbol) + fmt.Sprintf("%s\n", line)

		} else {
			output += line + "\n"
		}

	}
	return output
}

func (s TextSelector) lastLinePosition() int {
	return len(s.lines) - 1
}

func (s TextSelector) IsSelecting() bool {
	return s.Selection.Active
}

func (s TextSelector) firstLinePosition() int {
	return 1
}

func (s TextSelector) handleKeyUp() TextSelector {
	firstLinePosition := s.firstLinePosition()
	if s.cursor.line > firstLinePosition {
		projectedPosition := s.cursor.line - s.numberLines
		if projectedPosition < firstLinePosition {
			projectedPosition = firstLinePosition
		}

		if s.numberLines > 0 {
			s.cursor.line = projectedPosition
			s.numberLines = 0
		} else {
			s.cursor.line--
		}
	}
	s.AdjustScroll()
	return s
}

func (s TextSelector) handleKeyDown() TextSelector {
	lastLinePosition := s.lastLinePosition()
	if s.cursor.line < lastLinePosition {
		projectedPosition := s.cursor.line + s.numberLines
		if projectedPosition > lastLinePosition {
			projectedPosition = lastLinePosition
		}

		if s.numberLines > 0 {
			s.cursor.line = projectedPosition
			s.numberLines = 0
		} else {
			s.cursor.line++
		}
	}
	s.AdjustScroll()
	return s
}

func (s TextSelector) handleLineJumps(keypress string, parsedNumber int) TextSelector {
	if s.numberLines > 0 {
		prevNumber := strconv.Itoa(s.numberLines)
		combinedNumber, err := strconv.Atoi(prevNumber + keypress)
		if err == nil {
			s.numberLines = combinedNumber
		}
	} else {
		s.numberLines = parsedNumber
	}
	return s
}

func stripAnsiCodes(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mG]`)
	return ansiRegex.ReplaceAllString(str, "")
}

func (s TextSelector) copySelectedLinesToClipboard() {
	var selectedLines []string
	if s.Selection.Active {
		startLine := s.Selection.anchor.line
		endLine := s.cursor.line
		if startLine > endLine {
			startLine, endLine = endLine, startLine
		}
		for i := startLine; i <= endLine; i++ {
			filteredLine := filterLine(s.lines[i])
			selectedLines = append(selectedLines, filteredLine)
		}

	}
	linesToCopy := stripAnsiCodes(strings.Join(selectedLines, "\n"))
	clipboard.Write(clipboard.FmtText, []byte(linesToCopy))
}

func filterLine(line string) string {
	line = strings.Replace(line, "🤖", "", -1)
	line = strings.Replace(line, "💁", "", -1)
	return line
}

func NewTextSelector(w, h int, scrollPos int, sessionData string, colors util.SchemeColors) TextSelector {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	lines := strings.Split(sessionData, "\n")

	viewWidth, viewHeight := util.CalcVisualModeViewSize(w, h)
	pos := scrollPos + viewHeight/2
	if pos < 1 {
		pos = 1
	}

	if pos > len(lines) {
		pos = len(lines) - 1
	}

	state := TextSelector{
		lines:        lines,
		cursor:       Cursor{line: pos},
		Selection:    Selection{Active: false},
		scrollOffset: scrollPos,
		paneHeight:   viewHeight,
		paneWidth:    viewWidth,
		keys:         DefaultKeyMap,
		renderedText: sessionData,
		numberLines:  0,
		colors:       colors,
	}

	return state
}
