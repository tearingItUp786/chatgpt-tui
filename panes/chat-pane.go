package panes

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type ChatPane struct {
	isChatPaneReady        bool
	chatViewReady          bool
	chatContent            string
	isChatContainerFocused bool
	msgChan                chan clients.ProcessApiCompletionResponse

	terminalWidth  int
	terminalHeight int

	chatContainer lipgloss.Style
	chatView      viewport.Model
}

var chatContainerStyle = lipgloss.NewStyle().
	Border(lipgloss.ThickBorder()).
	BorderForeground(util.NormalTabBorderColor).
	MarginRight(util.ChatPaneMarginRight)

func NewChatPane(w, h int) ChatPane {
	chatContainerStyle = chatContainerStyle.Copy().Width(w).Height(h)
	chatView := viewport.New(w, h)
	chatView.SetContent(util.MotivationalMessage)
	msgChan := make(chan clients.ProcessApiCompletionResponse)
	return ChatPane{
		chatContainer:          chatContainerStyle,
		chatView:               chatView,
		chatViewReady:          false,
		chatContent:            util.MotivationalMessage,
		isChatContainerFocused: false,
		msgChan:                msgChan,
		terminalWidth:          util.DefaultTerminalWidth,
		terminalHeight:         util.DefaultTerminalHeight,
	}
}

func waitForActivity(sub chan clients.ProcessApiCompletionResponse) tea.Cmd {
	return func() tea.Msg {
		someMessage := <-sub
		return someMessage
	}
}

func (p ChatPane) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(p.msgChan),
	)
}

func (p ChatPane) Update(msg tea.Msg) (ChatPane, tea.Cmd) {
	var (
		cmd                    tea.Cmd
		cmds                   []tea.Cmd
		enableUpdateOfViewport = true
	)

	switch msg := msg.(type) {
	case util.FocusEvent:
		p.isChatContainerFocused = msg.IsFocused

		if p.isChatContainerFocused {
			p.chatContainer.BorderForeground(util.ActiveTabBorderColor)
		} else {
			p.chatContainer.BorderForeground(util.NormalTabBorderColor)
		}
		return p, nil

	case sessions.LoadDataFromDB:
		return p.initializePane(msg.Session)

	case sessions.UpdateCurrentSession:
		return p.initializePane(msg.Session)

	case sessions.ResponseChunkProcessed:
		paneWidth := p.chatContainer.GetWidth()

		oldContent := util.GetMessagesAsPrettyString(msg.PreviousMsgArray, paneWidth)
		styledBufferMessage := util.RenderBotMessage(msg.ChunkMessage, paneWidth)

		if styledBufferMessage != "" {
			styledBufferMessage = "\n" + styledBufferMessage
		}
		p.chatView.SetContent(wrap.String(oldContent+styledBufferMessage, paneWidth))
		p.chatView.GotoBottom()

		cmds = append(cmds, waitForActivity(p.msgChan))

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height
		_, paneHeight := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, false)
		p.chatContainer = p.chatContainer.Height(paneHeight)
		p.chatView.Height = p.chatContainer.GetHeight()

	case tea.KeyMsg:
		if !p.isChatContainerFocused {
			enableUpdateOfViewport = false
		}

		switch keypress := msg.String(); keypress {
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

func (p ChatPane) DisplayCompletion(session sessions.Model) tea.Cmd {
	return tea.Batch(
		session.GetCompletion(p.msgChan),
		waitForActivity(p.msgChan),
	)
}

func (p ChatPane) View() string {
	viewportContent := p.chatView.View()
	return p.chatContainer.Render(viewportContent)
}

func (p ChatPane) DisplayError(error string) string {
	return p.chatContainer.Render(error)
}

func (p ChatPane) SwitchToZenMode() {
	paneWidth, _ := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, true)
	p.chatContainer.
		BorderForeground(util.NormalTabBorderColor).
		Width(paneWidth)
}

func (p ChatPane) SwitchToNormalMode() {
	paneWidth, _ := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, false)
	p.chatContainer.
		BorderForeground(util.NormalTabBorderColor).
		Width(paneWidth)
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
	paneWidth, paneHeight := util.CalcChatPaneSize(p.terminalWidth, p.terminalHeight, false)
	if !p.isChatPaneReady {
		p.chatView = viewport.New(paneWidth, paneHeight)
		p.isChatPaneReady = true
	}

	oldContent := util.GetMessagesAsPrettyString(session.Messages, paneWidth)
	if oldContent == "" {
		oldContent = util.MotivationalMessage
	}
	p.chatView.SetContent(wrap.String(oldContent, paneWidth))
	p.chatView.GotoBottom()
	return p, nil
}
