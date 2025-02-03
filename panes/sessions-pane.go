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
	"github.com/tearingItUp786/chatgpt-tui/sessions"
	"github.com/tearingItUp786/chatgpt-tui/user"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type SessionsPane struct {
	sessionsListData []sessions.Session
	sessionsList     components.SessionsList
	textInput        textinput.Model
	sessionService   *sessions.SessionService
	userService      *user.UserService
	container        lipgloss.Style

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

	return SessionsPane{
		sessionService: ss,
		userService:    us,
		isFocused:      false,
		terminalWidth:  120,
		terminalHeight: 50,
		container: lipgloss.NewStyle().
			AlignVertical(lipgloss.Top).
			Border(lipgloss.ThickBorder(), true).
			BorderForeground(util.NormalTabBorderColor),
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
		p.sessionsListData = msg.AllSessions
		p.currentSessionID = msg.CurrentActiveSessionID
		listItems := constructSessionsListItems(msg.AllSessions, msg.CurrentActiveSessionID)
		p.sessionsList = components.NewSessionsList(listItems)
		p.currentEditID = -1

	case util.FocusEvent:
		p.isFocused = msg.IsFocused
		p.currentEditID = -1

	case util.OurWindowResize:
		width, _ := util.CalcSidePaneSize(p.terminalWidth, p.terminalHeight)
		p.container = p.container.Width(width)

	case tea.WindowSizeMsg:
		p.terminalWidth = msg.Width
		p.terminalHeight = msg.Height

	case tea.KeyMsg:
		if p.isFocused {
			dKeyPressed = msg.String() == "d"
			if p.currentEditID != -1 {
				cmd = p.handleCurrentEditID(msg)
				cmds = append(cmds, cmd)
			} else {
				cmd := p.handleCurrentNormalMode(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	if p.isFocused && p.currentEditID == -1 && !dKeyPressed {
		p.sessionsList, cmd = p.sessionsList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return p, tea.Batch(cmds...)
}

func (p SessionsPane) View() string {
	listView := p.normalListView()

	if p.isFocused {
		listView = p.sessionsList.EditListView(p.terminalHeight)
	}

	editForm := ""
	if p.currentEditID != -1 {
		editForm = p.textInput.View()
	}

	borderColor := util.NormalTabBorderColor

	if p.isFocused {
		borderColor = util.ActiveTabBorderColor
	}

	return p.container.BorderForeground(borderColor).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			listHeader("Sessions"),
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
			if err != nil {
				util.MakeErrorMsg(err.Error())
			}

			cmd = p.handleUpdateCurrentSession(session)
		}

	case "e":
		ti := textinput.New()
		ti.PromptStyle = lipgloss.NewStyle().PaddingLeft(2)
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
			p.currentEditID = -1
		}
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

func listHeader(str ...string) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		MarginLeft(2).
		Render(str...)
}

func listItem(heading string, value string, isActive bool) string {
	headingColor := util.Pink100
	color := "#bbb"
	if isActive {
		colorValue := util.Pink200
		color = colorValue
		headingColor = colorValue
	}
	headingEl := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color(headingColor)).
		Render
	spanEl := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render

	return headingEl(" "+heading, spanEl(value))
}

func (p SessionsPane) normalListView() string {
	sessionListItems := []string{}
	for _, session := range p.sessionsListData {
		isCurrentSession := p.currentSessionID == session.ID
		sessionListItems = append(
			sessionListItems,
			listItem(fmt.Sprint(session.ID), session.SessionName, isCurrentSession),
		)
	}

	return lipgloss.NewStyle().
		// TODO: figure out how to get height from the settings model
		Height(p.terminalHeight - 22).
		MaxHeight(p.terminalHeight - 22).
		Render(strings.Join(sessionListItems, "\n"))
}
