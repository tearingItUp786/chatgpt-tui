package views

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
	MarginRight(1)

func NewChatPane(w, h int) ChatPane {
	chatContainerStyle = chatContainerStyle.Copy().Width(w / 3 * 2).Height(h / 2)
	chatView := viewport.New(w/3*2, h/2)
	chatView.SetContent(util.MotivationalMessage)
	msgChan := make(chan clients.ProcessApiCompletionResponse)
	return ChatPane{
		chatContainer:          chatContainerStyle,
		chatView:               chatView,
		terminalWidth:          w,
		terminalHeight:         h,
		chatViewReady:          false,
		chatContent:            util.MotivationalMessage,
		isChatContainerFocused: false,
		msgChan:                msgChan,
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
	case sessions.LoadDataFromDB:
		if !p.isChatPaneReady {
			p.chatView = viewport.New(p.terminalWidth/3*2, p.terminalHeight/2)
			p.isChatPaneReady = true
		}

		oldContent := util.GetMessagesAsPrettyString(msg.Session.Messages, p.terminalWidth, p.terminalHeight)
		if oldContent == "" {
			oldContent = util.MotivationalMessage
		}
		p.chatContainer.Width(p.terminalWidth / 3 * 2)
		p.chatView.SetContent(wrap.String(oldContent, p.terminalWidth/3*2))
		p.chatView.GotoBottom()
		return p, cmd

	case sessions.UpdateCurrentSession:
		if !p.isChatPaneReady {
			p.chatView = viewport.New(p.terminalWidth/3*2, p.terminalHeight/2)
			p.isChatPaneReady = true
		}

		oldContent := util.GetMessagesAsPrettyString(msg.Session.Messages, p.terminalWidth, p.terminalHeight)
		if oldContent == "" {
			oldContent = util.MotivationalMessage
		}
		p.chatView.SetContent(wrap.String(oldContent, p.terminalWidth/3*2))
		p.chatView.GotoBottom()
		return p, cmd

	case sessions.ResponseChunkProcessed:
		oldContent := util.GetMessagesAsPrettyString(msg.PreviousMsgArray, p.terminalWidth, p.terminalHeight)
		styledBufferMessage := util.RenderBotMessage(msg.ChunkMessage, p.terminalWidth/3*2)

		if styledBufferMessage != "" {
			styledBufferMessage = "\n" + styledBufferMessage
		}
		p.chatView.SetContent(wrap.String(oldContent+styledBufferMessage, p.terminalWidth/3*2))
		p.chatView.GotoBottom()

		cmds = append(cmds, waitForActivity(p.msgChan))

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height

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
	p.chatContainer.
		BorderForeground(util.NormalTabBorderColor).
		Width(p.terminalWidth - 2)
}

func (p ChatPane) SwitchToNormalMode() {
	p.chatContainer.
		BorderForeground(util.NormalTabBorderColor).
		Width(p.terminalWidth / 3 * 2)
}

func (p ChatPane) SetPaneWitdth(w int) {
	p.chatContainer.Width(w)
}

func (p ChatPane) SetPaneHeight(h int) {
	p.chatContainer.Height(h)
}

func (p ChatPane) SetFocus(isFocused bool) ChatPane {
	p.isChatContainerFocused = isFocused
	if isFocused {
		p.chatContainer.BorderForeground(util.ActiveTabBorderColor)
	} else {
		p.chatContainer.BorderForeground(util.NormalTabBorderColor)
	}

	return p
}

func (p ChatPane) GetWidth() int {
	return p.chatContainer.GetWidth()
}
