package panes

import (
	"context"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/util"
	"golang.design/x/clipboard"
)

const ResponseWaitingMsg = "> Please wait ..."
const InitializingMsg = "Components initializing ..."
const PlaceholderMsg = "Prompts go here"

type PromptPane struct {
	input      textinput.Model
	textEditor textarea.Model
	container  lipgloss.Style
	inputMode  util.PrompInputMode
	colors     util.SchemeColors

	viewMode       util.ViewMode
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

	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	colors := config.ColorScheme.GetColors()

	input := textinput.New()
	input.Placeholder = InitializingMsg
	input.PromptStyle = lipgloss.NewStyle().Foreground(colors.ActiveTabBorderColor)
	input.CharLimit = 0
	input.Width = 0

	textEditor := textarea.New()
	textEditor.Placeholder = PlaceholderMsg
	textEditor.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(colors.ActiveTabBorderColor)
	textEditor.FocusedStyle.CursorLine.Background(lipgloss.NoColor{})
	textEditor.FocusedStyle.EndOfBuffer = lipgloss.NewStyle().Foreground(colors.ActiveTabBorderColor)
	textEditor.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(colors.AccentColor).Faint(true)

	textEditor.EndOfBufferCharacter = rune(' ')
	textEditor.ShowLineNumbers = true
	textEditor.CharLimit = 0
	textEditor.MaxHeight = 0
	textEditor.Blur()

	container := lipgloss.NewStyle().
		AlignVertical(lipgloss.Bottom).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(colors.ActiveTabBorderColor).
		MarginTop(util.PromptPaneMarginTop)

	return PromptPane{
		viewMode:       util.NormalMode,
		colors:         colors,
		input:          input,
		textEditor:     textEditor,
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
		switch p.viewMode {
		case util.TextEditMode:
			p.textEditor, cmd = p.textEditor.Update(msg)
		default:
			p.input, cmd = p.input.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case util.ViewModeChanged:
		p.viewMode = msg.Mode

		isTextEditMode := p.viewMode == util.TextEditMode
		w, h := util.CalcPromptPaneSize(p.terminalWidth, p.terminalHeight, isTextEditMode)
		if isTextEditMode {
			p.textEditor.SetHeight(h)
			p.textEditor.SetWidth(w)

			currentInput := p.input.Value()
			p.input.Blur()
			p.input.Reset()

			p.inputMode = util.PromptInsertMode

			p.textEditor.SetValue(currentInput)
			p.textEditor.Focus()
			cmds = append(cmds, p.textEditor.Cursor.BlinkCmd())
		} else {
			p.input.Width = w
			currentInput := p.textEditor.Value()
			p.textEditor.Blur()
			p.textEditor.Reset()

			p.input.SetValue(currentInput)
		}
		p.container = p.container.Copy().MaxWidth(p.terminalWidth).Width(w)

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

		isTextEditMode := p.viewMode == util.TextEditMode
		w, h := util.CalcPromptPaneSize(p.terminalWidth, p.terminalHeight, isTextEditMode)
		if isTextEditMode {
			p.textEditor.SetHeight(h)
			p.textEditor.SetWidth(w)
		} else {
			p.input.Width = w
		}
		p.container = p.container.Copy().MaxWidth(p.terminalWidth).Width(w)

	case tea.KeyMsg:
		if !p.ready {
			break
		}

		switch keypress := msg.String(); keypress {
		case "i":
			if p.isFocused && p.inputMode == util.PromptNormalMode {
				p.inputMode = util.PromptInsertMode
				switch p.viewMode {
				case util.TextEditMode:
					p.textEditor.Focus()
					cmds = append(cmds, p.textEditor.Cursor.BlinkCmd())
				default:
					p.input.Focus()
					cmds = append(cmds, p.input.Cursor.BlinkCmd())
				}
			}
		}

		switch msg.Type {

		case tea.KeyEscape:
			if p.isFocused {
				p.inputMode = util.PromptNormalMode

				switch p.viewMode {
				case util.TextEditMode:
					p.textEditor.Blur()
				default:
					p.input.Blur()
				}
			}

		case tea.KeyEnter:
			if p.isFocused && p.isSessionIdle {

				switch p.viewMode {
				case util.TextEditMode:
					if !p.textEditor.Focused() {
						promptText := p.textEditor.Value()
						log.Println("\n" + promptText)
						p.textEditor.SetValue("")
						p.textEditor.Blur()
						return p, tea.Batch(
							util.SendPromptReadyMsg(promptText),
							util.SendViewModeChangedMsg(util.NormalMode))
					}
				default:
					promptText := p.input.Value()
					p.input.SetValue("")
					p.input.Blur()

					p.inputMode = util.PromptNormalMode

					return p, util.SendPromptReadyMsg(promptText)
				}
			}
		case tea.KeyCtrlV:
			if p.isFocused && p.viewMode == util.TextEditMode && p.textEditor.Focused() {
				buffer := clipboard.Read(clipboard.FmtText)
				editorValue := p.textEditor.Value()
				p.textEditor.SetValue(editorValue + string(buffer))
				p.textEditor.SetCursor(0)
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
		content := p.input.View()
		if p.viewMode == util.TextEditMode {
			content = p.textEditor.View()
		}
		return p.container.Render(content)
	}

	return p.container.Render(ResponseWaitingMsg)
}
