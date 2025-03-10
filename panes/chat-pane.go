package panes

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type displayMode int

const (
	normalMode displayMode = iota
	selectionMode
)

type ChatPane struct {
	isChatPaneReady        bool
	chatViewReady          bool
	displayMode            displayMode
	chatContent            string
	renderedContent        string
	isChatContainerFocused bool
	msgChan                chan clients.ProcessApiCompletionResponse
	viewMode               util.ViewMode

	terminalWidth  int
	terminalHeight int

	colors        util.SchemeColors
	chatContainer lipgloss.Style
	chatView      viewport.Model
	selectionView components.TextSelector
}

var chatContainerStyle = lipgloss.NewStyle().
	Border(lipgloss.ThickBorder()).
	MarginRight(util.ChatPaneMarginRight)

func NewChatPane(ctx context.Context, w, h int) ChatPane {
	chatView := viewport.New(w, h)
	chatView.SetContent(util.MotivationalMessage)
	msgChan := make(chan clients.ProcessApiCompletionResponse)

	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}
	colors := config.ColorScheme.GetColors()

	chatContainerStyle = chatContainerStyle.
		Copy().
		Width(w).
		Height(h).
		BorderForeground(colors.NormalTabBorderColor)

	return ChatPane{
		viewMode:               util.NormalMode,
		colors:                 colors,
		chatContainer:          chatContainerStyle,
		chatView:               chatView,
		chatViewReady:          false,
		chatContent:            util.MotivationalMessage,
		renderedContent:        util.MotivationalMessage,
		isChatContainerFocused: false,
		msgChan:                msgChan,
		terminalWidth:          util.DefaultTerminalWidth,
		terminalHeight:         util.DefaultTerminalHeight,
		displayMode:            normalMode,
	}
}

func waitForActivity(sub chan clients.ProcessApiCompletionResponse) tea.Cmd {
	return func() tea.Msg {
		someMessage := <-sub
		return someMessage
	}
}

func (p ChatPane) Init() tea.Cmd {
	return nil
}

func (p ChatPane) Update(msg tea.Msg) (ChatPane, tea.Cmd) {
	var (
		cmd                    tea.Cmd
		cmds                   []tea.Cmd
		enableUpdateOfViewport = true
	)

	if p.IsSelectionMode() {
		p.selectionView, cmd = p.selectionView.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case util.ViewModeChanged:
		p.viewMode = msg.Mode
		w, h := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, p.viewMode)
		p.chatView.Height = h
		p.chatView.Width = w
		p.chatContainer = p.chatContainer.
			Width(w).
			Height(h)

	case util.FocusEvent:
		p.isChatContainerFocused = msg.IsFocused

		if p.isChatContainerFocused {
			p.chatContainer.BorderForeground(p.colors.ActiveTabBorderColor)
		} else {
			p.chatContainer.BorderForeground(p.colors.NormalTabBorderColor)
		}
		return p, nil

	case sessions.LoadDataFromDB:
		return p.initializePane(msg.Session)

	case sessions.UpdateCurrentSession:
		return p.initializePane(msg.Session)

	case sessions.ResponseChunkProcessed:
		paneWidth := p.chatContainer.GetWidth()

		oldContent := util.GetMessagesAsPrettyString(msg.PreviousMsgArray, paneWidth, p.colors)
		styledBufferMessage := util.RenderBotMessage(msg.ChunkMessage, paneWidth, p.colors, false)

		if styledBufferMessage != "" {
			styledBufferMessage = "\n" + styledBufferMessage
		}

		rendered := wrap.String(oldContent+styledBufferMessage, paneWidth)
		p.renderedContent = rendered
		p.chatView.SetContent(rendered)
		p.chatView.GotoBottom()

		cmds = append(cmds, waitForActivity(p.msgChan))

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height
		w, h := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, p.viewMode)
		p.chatView.Height = h
		p.chatView.Width = w
		p.chatContainer = p.chatContainer.
			Width(w).
			Height(h)

	case tea.KeyMsg:
		if !p.isChatContainerFocused {
			enableUpdateOfViewport = false
		}

		if p.IsSelectionMode() {
			switch msg.Type {
			case tea.KeyEsc:
				p.displayMode = normalMode
				p.chatContainer.BorderForeground(p.colors.ActiveTabBorderColor)
			}
		}

		if p.IsSelectionMode() {
			break
		}

		switch keypress := msg.String(); keypress {
		case "v":
			if !p.isChatContainerFocused {
				break
			}
			p.displayMode = selectionMode
			enableUpdateOfViewport = false
			p.chatContainer.BorderForeground(p.colors.AccentColor)
			p.selectionView = components.NewTextSelector(
				p.terminalWidth,
				p.terminalHeight,
				p.chatView.YOffset,
				p.renderedContent,
				p.colors)
			p.selectionView.AdjustScroll()

		case "y":
			if p.isChatContainerFocused {
				copyLast := func() tea.Msg {
					return util.SendCopyLastMsg()
				}
				cmds = append(cmds, copyLast)
			}

		case "Y":
			if p.isChatContainerFocused {
				copyAll := func() tea.Msg {
					return util.SendCopyAllMsgs()
				}
				cmds = append(cmds, copyAll)
			}
		}
	}

	if enableUpdateOfViewport {
		p.chatView, cmd = p.chatView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return p, tea.Batch(cmds...)
}

func (p ChatPane) IsSelectionMode() bool {
	return p.displayMode == selectionMode
}

func (p ChatPane) DisplayCompletion(ctx context.Context, orchestrator sessions.Orchestrator) tea.Cmd {
	return tea.Batch(
		orchestrator.GetCompletion(ctx, p.msgChan),
		waitForActivity(p.msgChan),
	)
}

func (p ChatPane) View() string {
	if p.IsSelectionMode() {
		return p.chatContainer.Render(p.selectionView.View())
	}

	viewportContent := p.chatView.View()
	return p.chatContainer.Render(viewportContent)
}

func (p ChatPane) DisplayError(error string) string {
	return p.chatContainer.Render(util.RenderErrorMessage(error, p.chatContainer.GetWidth(), p.colors))
}

func (p ChatPane) SetPaneWitdth(w int) {
	p.chatContainer.Width(w)
}

func (p ChatPane) SetPaneHeight(h int) {
	p.chatContainer.Height(h)
}

func (p ChatPane) GetWidth() int {
	return p.chatContainer.GetWidth()
}

func (p ChatPane) initializePane(session sessions.Session) (ChatPane, tea.Cmd) {
	paneWidth, paneHeight := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, p.viewMode)
	if !p.isChatPaneReady {
		p.chatView = viewport.New(paneWidth, paneHeight)
		p.isChatPaneReady = true
	}

	oldContent := util.GetMessagesAsPrettyString(session.Messages, paneWidth, p.colors)
	if oldContent == "" {
		oldContent = util.MotivationalMessage
	}
	rendered := util.GetVisualModeView(session.Messages, paneWidth, p.colors)
	p.renderedContent = wrap.String(rendered, paneWidth)
	p.chatView.SetContent(wrap.String(oldContent, paneWidth))
	p.chatView.GotoBottom()
	return p, nil
}
