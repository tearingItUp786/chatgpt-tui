package panes

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const ResponseWaitingMsg = "> Please wait ..."
const InitializingMsg = "Components initializing ..."
const PlaceholderMsg = "Prompts go here"

type PromptPane struct {
	input     textinput.Model
	container lipgloss.Style
	inputMode util.PrompInputMode
	colors    util.SchemeColors

	isSessionIdle  bool
	isFocused      bool
	terminalWidth  int
	terminalHeight int
	ready          bool
}

func NewPromptPane(ctx context.Context) PromptPane {
	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}
	colors := config.ColorScheme.GetColors()

	input := textinput.New()
	input.Placeholder = InitializingMsg
	input.PromptStyle = lipgloss.NewStyle().Foreground(colors.ActiveTabBorderColor)

	container := lipgloss.NewStyle().
		AlignVertical(lipgloss.Bottom).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(colors.ActiveTabBorderColor).
		MaxHeight(util.PromptPaneHeight).
		MarginTop(util.PromptPaneMarginTop)

	return PromptPane{
		colors:         colors,
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
			p.container = p.container.BorderForeground(p.colors.ActiveTabBorderColor)
			p.input.PromptStyle = p.input.PromptStyle.Copy().Foreground(p.colors.ActiveTabBorderColor)
		} else {
			p.inputMode = util.PromptNormalMode
			p.container = p.container.BorderForeground(p.colors.NormalTabBorderColor)
			p.input.PromptStyle = p.input.PromptStyle.Foreground(p.colors.NormalTabBorderColor)
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
		if !p.ready {
			break
		}

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

func (p PromptPane) Enable() PromptPane {
	p.input.Placeholder = PlaceholderMsg
	p.ready = true
	return p
}

func (p PromptPane) View() string {
	if p.isSessionIdle {
		return p.container.Render(p.input.View())
	}

	return p.container.Render(ResponseWaitingMsg)
}
