package panes

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tearingItUp786/chatgpt-tui/components"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/user"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const EditModeDisabled = -1

type SessionsPane struct {
	sessionsListData []sessions.Session
	sessionsList     components.SessionsList
	textInput        textinput.Model
	sessionService   *sessions.SessionService
	userService      *user.UserService
	container        lipgloss.Style
	colors           util.SchemeColors
	currentSession   sessions.Session

	sessionsListReady  bool
	currentSessionID   int
	currentEditID      int
	currentSessionName string
	isFocused          bool
	terminalWidth      int
	terminalHeight     int
}

func NewSessionsPane(db *sql.DB, ctx context.Context) SessionsPane {
	ss := sessions.NewSessionService(db)
	us := user.NewUserService(db)

	config, ok := config.FromContext(ctx)
	if !ok {
		fmt.Println("No config found")
		panic("No config found in context")
	}
	colors := config.ColorScheme.GetColors()

	return SessionsPane{
		colors:         colors,
		sessionService: ss,
		userService:    us,
		isFocused:      false,
		terminalWidth:  util.DefaultTerminalWidth,
		terminalHeight: util.DefaultTerminalHeight,
		container: lipgloss.NewStyle().
			AlignVertical(lipgloss.Top).
			Border(lipgloss.ThickBorder(), true).
			BorderForeground(colors.NormalTabBorderColor),
	}
}

func (p SessionsPane) Init() tea.Cmd {
	return nil
}

func (p SessionsPane) Update(msg tea.Msg) (SessionsPane, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	dKeyPressed := false
	switch msg := msg.(type) {
	case sessions.LoadDataFromDB:
		p.currentSession = msg.Session
		p.sessionsListData = msg.AllSessions
		p.currentSessionID = msg.CurrentActiveSessionID
		listItems := constructSessionsListItems(msg.AllSessions, msg.CurrentActiveSessionID)
		w, h := util.CalcSessionsListSize(p.terminalWidth, p.terminalHeight)
		p.sessionsList = components.NewSessionsList(listItems, w, h, p.colors)
		p.currentEditID = EditModeDisabled
		p.sessionsListReady = true

	case util.FocusEvent:
		p.isFocused = msg.IsFocused
		p.currentEditID = EditModeDisabled

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height
		width, height := util.CalcSessionsPaneSize(p.terminalWidth, p.terminalHeight)
		p.container = p.container.Width(width).Height(height)
		if p.sessionsListReady {
			width, height = util.CalcSessionsListSize(p.terminalWidth, p.terminalHeight)
			p.sessionsList.SetSize(width, height)
		}

	case util.ProcessingStateChanged:
		if !msg.IsProcessing {
			session, err := p.sessionService.GetSession(p.currentSessionID)
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}
			cmds = append(cmds, p.handleUpdateCurrentSession(session))
		}

	case tea.KeyMsg:
		if p.isFocused {
			dKeyPressed = msg.String() == "d"
			if p.currentEditID != EditModeDisabled {
				cmd = p.handleCurrentEditID(msg)
				cmds = append(cmds, cmd)
			} else {
				cmd := p.handleCurrentNormalMode(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	if p.isFocused && p.currentEditID == EditModeDisabled && !dKeyPressed {
		p.sessionsList, cmd = p.sessionsList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return p, tea.Batch(cmds...)
}

func (p SessionsPane) View() string {
	listView := p.normalListView()

	if p.isFocused {
		_, paneHeight := util.CalcSessionsPaneSize(p.terminalWidth, p.terminalHeight)
		listView = p.sessionsList.EditListView(paneHeight)
	}

	editForm := ""
	if p.currentEditID != EditModeDisabled {
		editForm = p.textInput.View()
	}

	borderColor := p.colors.NormalTabBorderColor

	if p.isFocused {
		borderColor = p.colors.ActiveTabBorderColor
	}

	return p.container.BorderForeground(borderColor).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			p.listHeader("Sessions"),
			listView,
			editForm,
		),
	)
}

func (p *SessionsPane) handleCurrentNormalMode(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+n":

		currentTime := time.Now()
		formattedTime := currentTime.Format(time.ANSIC)
		defaultSessionName := fmt.Sprintf("%s", formattedTime)
		newSession, _ := p.sessionService.InsertNewSession(defaultSessionName, []util.MessageToSend{})

		cmd = p.handleUpdateCurrentSession(newSession)
		p.updateSessionsList()

	case "enter":
		i, ok := p.sessionsList.GetSelectedItem()
		if ok {
			session, err := p.sessionService.GetSession(i.Id)
			p.currentSessionID = i.Id
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}

			cmd = p.handleUpdateCurrentSession(session)
		}

	case "e":
		ti := textinput.New()
		ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(util.DefaultElementsPadding)
		p.textInput = ti
		i, ok := p.sessionsList.GetSelectedItem()
		if ok {
			p.currentEditID = i.Id
			p.textInput.Placeholder = "New Session Name"
		}
		p.textInput.Focus()
		p.textInput.CharLimit = 100

	case "d":
		i, ok := p.sessionsList.GetSelectedItem()
		if ok {
			// delete this one if it's not the active one
			if i.Id != p.currentSessionID {
				p.sessionService.DeleteSession(i.Id)
				p.updateSessionsList()
			}
		}
	}

	return cmd
}

func (p *SessionsPane) handleUpdateCurrentSession(session sessions.Session) tea.Cmd {
	p.currentSession = session
	p.userService.UpdateUserCurrentActiveSession(1, session.ID)

	p.currentSessionID = session.ID
	p.currentSessionName = session.SessionName

	listItems := constructSessionsListItems(p.sessionsListData, p.currentSessionID)
	p.sessionsList.SetItems(listItems)

	return sessions.SendUpdateCurrentSessionMsg(session)
}

func (p *SessionsPane) handleCurrentEditID(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	p.textInput, cmd = p.textInput.Update(msg)

	if msg.String() == "enter" {
		if p.textInput.Value() != "" {
			p.sessionService.UpdateSessionName(p.currentEditID, p.textInput.Value())
			p.updateSessionsList()
			p.currentEditID = EditModeDisabled
		}
	}

	if msg.String() == "esc" {
		p.currentEditID = EditModeDisabled
	}

	return cmd
}

func constructSessionsListItems(sessions []sessions.Session, currentSessionId int) []list.Item {
	items := []list.Item{}

	for _, session := range sessions {
		anItem := components.SessionListItem{
			Id:       session.ID,
			Text:     session.SessionName,
			IsActive: session.ID == currentSessionId,
		}
		items = append(items, anItem)
	}

	return items
}

func (p *SessionsPane) updateSessionsList() {
	p.sessionsListData, _ = p.sessionService.GetAllSessions()
	items := constructSessionsListItems(p.sessionsListData, p.currentSessionID)
	p.sessionsList.SetItems(items)
}

func (p SessionsPane) listHeader(str ...string) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Foreground(p.colors.DefaultTextColor).
		MarginLeft(util.ListItemMarginLeft).
		Render(str...)
}

func (p SessionsPane) listItem(heading string, value string, isActive bool, widthCap int) string {
	headingColor := p.colors.MainColor
	color := p.colors.NormalTabBorderColor
	if isActive {
		colorValue := p.colors.ActiveTabBorderColor
		color = colorValue
		headingColor = colorValue
	}
	headingEl := lipgloss.NewStyle().
		PaddingLeft(util.ListItemPaddingLeft).
		Foreground(lipgloss.Color(headingColor)).
		Bold(isActive).
		Render
	spanEl := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render

	value = util.TrimListItem(value, widthCap)

	return headingEl("■ "+heading, spanEl(value))
}

func (p SessionsPane) normalListView() string {
	sessionListItems := []string{}
	listWidth := p.sessionsList.GetWidth()
	for _, session := range p.sessionsListData {
		isCurrentSession := p.currentSessionID == session.ID
		sessionListItems = append(
			sessionListItems,
			p.listItem(fmt.Sprint(session.ID), session.SessionName, isCurrentSession, listWidth),
		)
	}

	w, h := util.CalcSessionsListSize(p.terminalWidth, p.terminalHeight)

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		MaxHeight(h).
		Render(strings.Join(sessionListItems, "\n"))
}
