package components

import (
	"fmt"
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

type keyMap struct {
	visualLineMode key.Binding
	up             key.Binding
	down           key.Binding
	pageUp         key.Binding
	pageDown       key.Binding
	copy           key.Binding
	bottom         key.Binding
	top            key.Binding
}

var defaultKeyMap = keyMap{
	visualLineMode: key.NewBinding(key.WithKeys("V", "v", tea.KeySpace.String()), key.WithHelp("V, v, <space>", "visual line mode")),
	up:             key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	down:           key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	pageUp:         key.NewBinding(key.WithKeys("ctrl+u", "u"), key.WithHelp("ctrl+u", "move up a page")),
	pageDown:       key.NewBinding(key.WithKeys("ctrl+d", "d"), key.WithHelp("ctrl+d", "move down a page")),
	copy:           key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy selection")),
	bottom:         key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "go to bottom")),
	top:            key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "go to top")),
}

type cursor struct {
	line int
}

type selection struct {
	Active bool
	anchor cursor
}

type TextSelector struct {
	Selection    selection
	lines        []string
	cursor       cursor
	scrollOffset int
	paneHeight   int
	paneWidth    int
	keys         keyMap
	renderedText string
	colors       util.SchemeColors

	numberLines int
}

func (s TextSelector) Init() tea.Cmd {
	return nil
}

func (s TextSelector) Update(msg tea.Msg) (TextSelector, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

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

		case key.Matches(msg, s.keys.pageUp):
			upLines := s.paneHeight / 2
			s.cursor.line = max(s.cursor.line-upLines, s.firstLinePosition())
			s.AdjustScroll()

		case key.Matches(msg, s.keys.up):
			s = s.handleKeyUp()

		case key.Matches(msg, s.keys.down):
			s = s.handleKeyDown()

		case key.Matches(msg, s.keys.pageDown):
			downLines := s.paneHeight / 2
			s.cursor.line = min(s.cursor.line+downLines, s.lastLinePosition())
			s.AdjustScroll()

		case key.Matches(msg, s.keys.bottom):
			s.cursor.line = s.lastLinePosition()
			s.AdjustScroll()

		case key.Matches(msg, s.keys.top):
			s.cursor.line = s.firstLinePosition()
			s.AdjustScroll()

		case key.Matches(msg, s.keys.visualLineMode):
			if s.Selection.Active {
				s.Selection.Active = false
			} else {
				s.Selection.Active = true
				s.Selection.anchor = s.cursor
			}

		case key.Matches(msg, s.keys.copy):
			if s.Selection.Active {
				s.copySelectedLinesToClipboard()
				s.Selection.Active = false
				cmds = append(cmds, util.SendCopiedToBufferMsg())
			}
		}
	}

	return s, tea.Batch(cmds...)
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
		Foreground(s.colors.DefaultTextColor).
		Background(s.colors.HighlightColor)

	cursorStyle := lipgloss.NewStyle().
		Foreground(s.colors.DefaultTextColor).
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

func (s TextSelector) firstLinePosition() int {
	return 1
}

func (s TextSelector) handleKeyUp() TextSelector {
	firstLinePosition := s.firstLinePosition()
	if s.cursor.line > firstLinePosition {
		projectedPosition := s.cursor.line - s.numberLines
		projectedPosition = max(projectedPosition, firstLinePosition)

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
		projectedPosition = min(projectedPosition, lastLinePosition)

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

func (s TextSelector) copySelectedLinesToClipboard() {
	if !s.Selection.Active {
		return
	}

	var selectedLines []string
	startLine := s.Selection.anchor.line
	endLine := s.cursor.line
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}
	for i := startLine; i <= endLine; i++ {
		filteredLine := filterLine(s.lines[i])
		selectedLines = append(selectedLines, filteredLine)
	}

	linesToCopy := util.StripAnsiCodes(strings.Join(selectedLines, "\n"))
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
	pos = max(pos, 1)

	if pos > len(lines) {
		pos = len(lines) - 1
	}

	state := TextSelector{
		lines:        lines,
		cursor:       cursor{line: pos},
		Selection:    selection{Active: false},
		scrollOffset: scrollPos,
		paneHeight:   viewHeight,
		paneWidth:    viewWidth,
		keys:         defaultKeyMap,
		renderedText: sessionData,
		numberLines:  0,
		colors:       colors,
	}

	return state
}
