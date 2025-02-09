package panes

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const ResponseWaitingMsg = "> Please wait ..."

type PromptPane struct {
	input     textinput.Model
	container lipgloss.Style
	inputMode util.PrompInputMode

	isSessionIdle  bool
	isFocused      bool
	terminalWidth  int
	terminalHeight int
}

func NewPromptPane() PromptPane {
	input := textinput.New()
	input.Placeholder = "Prompts go here"
	input.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(util.ActiveTabBorderColor))

	container := lipgloss.NewStyle().
		AlignVertical(lipgloss.Bottom).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(util.ActiveTabBorderColor).
		MaxHeight(util.PromptPaneHeight).
		MarginTop(util.PromptPaneMarginTop)

	return PromptPane{
		input:          input,
		container:      container,
		inputMode:      util.PromptNormalMode,
		isSessionIdle:  true,
		isFocused:      true,
		terminalWidth:  util.DefaultTerminalWidth,
		terminalHeight: util.DefaultTerminalHeight,
	}
}

func (p PromptPane) Init() tea.Cmd {
	return p.input.Cursor.BlinkCmd()
}

func (p PromptPane) Update(msg tea.Msg) (PromptPane, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if p.isFocused && p.inputMode == util.PromptInsertMode && p.isSessionIdle {
		p.input, cmd = p.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {

	case util.ProcessingStateChanged:
		p.isSessionIdle = msg.IsProcessing == false

	case util.FocusEvent:
		p.isFocused = msg.IsFocused

		if p.isFocused {
			p.inputMode = util.PromptNormalMode
			p.container = p.container.BorderForeground(util.ActiveTabBorderColor)
			p.input.PromptStyle = p.input.PromptStyle.Copy().Foreground(lipgloss.Color(util.ActiveTabBorderColor))
		} else {
			p.inputMode = util.PromptNormalMode
			p.container = p.container.BorderForeground(util.NormalTabBorderColor)
			p.input.PromptStyle = p.input.PromptStyle.Foreground(lipgloss.Color(util.NormalTabBorderColor))
			p.input.Blur()
		}
		return p, nil

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height

		w, _ := util.CalcPromptPaneSize(p.terminalWidth, p.terminalHeight)
		p.container = p.container.Copy().MaxWidth(p.terminalWidth).Width(w)
		p.input.Width = w

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "i":
			if p.isFocused && p.inputMode == util.PromptNormalMode {
				p.inputMode = util.PromptInsertMode
				p.input.Focus()
				cmds = append(cmds, p.input.Cursor.BlinkCmd())
			}
		}

		switch msg.Type {

		case tea.KeyEscape:
			if p.isFocused {
				p.inputMode = util.PromptNormalMode
				p.input.Blur()
			}

		case tea.KeyEnter:
			if p.isFocused && p.isSessionIdle {
				promptText := p.input.Value()
				p.input.SetValue("")
				p.input.Focus()

				p.inputMode = util.PromptInsertMode

				return p, util.SendPromptReadyMsg(promptText)
			}
		}
	}

	return p, tea.Batch(cmds...)
}

func (p PromptPane) IsTypingInProcess() bool {
	return p.isFocused && p.inputMode == util.PromptInsertMode
}

func (p PromptPane) View() string {
	if p.isSessionIdle {
		return p.container.Render(p.input.View())
	}

	return p.container.Render(ResponseWaitingMsg)
}
