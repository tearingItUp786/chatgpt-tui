package panes

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const NotificationDisplayDurationSec = 2

const (
	copiedLabelText     = "Copied"
	idleLabelText       = "IDLE"
	processingLabelText = "Processing"
)

var infoSpinnerStyle = lipgloss.NewStyle()
var defaultLabelStyle = lipgloss.NewStyle().
	BorderLeft(true).
	BorderStyle(lipgloss.InnerHalfBlockBorder()).
	Bold(true).
	MarginRight(1).
	PaddingRight(1).
	PaddingLeft(1)

type InfoPane struct {
	sessionService *sessions.SessionService
	currentSession sessions.Session
	colors         util.SchemeColors
	spinner        spinner.Model

	processingIdleLabel   lipgloss.Style
	processingActiveLabel lipgloss.Style
	promptTokensLablel    lipgloss.Style
	completionTokensLabel lipgloss.Style
	copiedLabel           lipgloss.Style

	showCopyLabel  bool
	isProcessing   bool
	terminalWidth  int
	terminalHeight int
}

func NewInfoPane(db *sql.DB, ctx context.Context) InfoPane {
	ss := sessions.NewSessionService(db)

	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}
	colors := config.ColorScheme.GetColors()
	spinner := initInfoSpinner()

	infoSpinnerStyle = infoSpinnerStyle.Copy().Foreground(colors.HighlightColor)
	processingIdleLabel := defaultLabelStyle.Copy().
		BorderLeftForeground(colors.HighlightColor).
		Foreground(colors.DefaultTextColor)
	processingActiveLabel := defaultLabelStyle.Copy().
		BorderLeftForeground(colors.AccentColor).
		Foreground(colors.DefaultTextColor)
	promptTokensLablel := defaultLabelStyle.Copy().
		BorderLeftForeground(colors.ActiveTabBorderColor).
		Foreground(colors.DefaultTextColor)
	completionTokensLabel := defaultLabelStyle.Copy().
		BorderLeftForeground(colors.ActiveTabBorderColor).
		Foreground(colors.DefaultTextColor)
	copiedLabel := defaultLabelStyle.Copy().
		Background(colors.NormalTabBorderColor).
		BorderLeftForeground(colors.HighlightColor).
		Foreground(colors.DefaultTextColor)

	return InfoPane{
		processingIdleLabel:   processingIdleLabel,
		processingActiveLabel: processingActiveLabel,
		promptTokensLablel:    promptTokensLablel,
		completionTokensLabel: completionTokensLabel,
		copiedLabel:           copiedLabel,

		spinner:        spinner,
		colors:         colors,
		sessionService: ss,
		terminalWidth:  util.DefaultTerminalWidth,
		terminalHeight: util.DefaultTerminalHeight,
	}
}

func initInfoSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Ellipsis
	s.Style = infoSpinnerStyle

	return s
}

type tickMsg struct{}

func (p InfoPane) Init() tea.Cmd {
	return nil
}

func (p InfoPane) Update(msg tea.Msg) (InfoPane, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height

	case sessions.LoadDataFromDB:
		p.currentSession = msg.Session

	case sessions.UpdateCurrentSession:
		p.currentSession = msg.Session

	case spinner.TickMsg:
		p.spinner, cmd = p.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case util.CopiedToBufferMsg:
		p.showCopyLabel = true
		cmds = append(cmds, tickAfter(NotificationDisplayDurationSec))

	case tickMsg:
		p.showCopyLabel = false

	case util.ProcessingStateChanged:
		p.isProcessing = msg.IsProcessing
		if !msg.IsProcessing {
			session, err := p.sessionService.GetSession(p.currentSession.ID)
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}
			p.currentSession = session
		} else {
			cmds = append(cmds, p.spinner.Tick)
		}
	}

	return p, tea.Batch(cmds...)
}

func (p InfoPane) View() string {
	paneWidth, _ := util.CalcSettingsPaneSize(p.terminalWidth, p.terminalHeight)
	var processingLabel string
	if p.isProcessing {
		processingLabel = p.processingActiveLabel.Render(processingLabelText + p.spinner.View())
	} else {
		processingLabel = p.processingIdleLabel.Render(idleLabelText)
	}

	promptTokensLablel := p.promptTokensLablel.Render(fmt.Sprintf("IN: %d", p.currentSession.PromptTokens))
	completionTokensLabel := p.completionTokensLabel.Render(fmt.Sprintf("OUT: %d", p.currentSession.CompletionTokens))

	firstRow := processingLabel

	if p.showCopyLabel {
		firstRow = lipgloss.JoinHorizontal(
			lipgloss.Left,
			processingLabel,
			p.copiedLabel.
				MarginLeft(paneWidth-len(copiedLabelText)-len(idleLabelText)-util.CopiedLabelCounterweght).
				Render(copiedLabelText))
	}

	secondRow := lipgloss.JoinHorizontal(
		lipgloss.Left,
		promptTokensLablel,
		completionTokensLabel,
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(p.colors.NormalTabBorderColor).
		Width(paneWidth).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				firstRow,
				secondRow,
			),
		)
}

func tickAfter(seconds int) tea.Cmd {
	return tea.Tick(time.Second*time.Duration(seconds), func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
