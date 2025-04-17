package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/renatus-cartesius/nedovault/api"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle     = lipgloss.NewStyle().Margin(1, 5)
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})
	loginStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63")).Align(lipgloss.Center)
	noStyle      = lipgloss.NewStyle()
)

//type SecretItem struct {
//	name, stype, updated string
//}

type SecretItem struct {
	SecretMeta *api.SecretMeta
}

func (i *SecretItem) Title() string { return string(i.SecretMeta.Key) }
func (i *SecretItem) Description() string {
	return fmt.Sprintf("type: %s, updated: %s", i.SecretMeta.Type, i.SecretMeta.Timestamp.AsTime().Format(time.RFC850))
}
func (i *SecretItem) FilterValue() string { return string(i.SecretMeta.Key) }

type loginPage struct {
	inputs  []textinput.Model
	lastErr error
	current int
}

type model struct {
	sp list.Model

	lp loginPage

	client api.NedoVaultClient
	token  string

	username string

	mx         *sync.Mutex
	isLoggedIn bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) updateLoginPage(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := context.Background()

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {
		case "ctrl+t":
			m.isLoggedIn = !m.isLoggedIn
		case "ctrl+c":
			return m, tea.Quit
		case "enter", "tab", "up", "down":
			s := msg.String()

			if s == "enter" && isInputsFilled(m.lp.inputs) {

				// obtaining token and reset logpass

				username := m.lp.inputs[0].Value()
				password := m.lp.inputs[1].Value()

				res, err := m.client.Authorize(
					ctx,
					&api.AuthRequest{
						Username: []byte(username),
						Password: []byte(password),
					},
				)

				if err != nil {
					m.lp.lastErr = err
					return m, nil
				}

				m.lp.lastErr = nil
				m.token = res.Token
				m.username = username

				m.lp.inputs[0].SetValue("")
				m.lp.inputs[1].SetValue("")

				m.isLoggedIn = true
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.lp.current--
			} else {
				m.lp.current++
			}

			if m.lp.current >= len(m.lp.inputs) {
				m.lp.current = 0
			} else if m.lp.current < 0 {
				m.lp.current = len(m.lp.inputs) - 1
			}

			cmds := make([]tea.Cmd, len(m.lp.inputs))
			for i := 0; i <= len(m.lp.inputs)-1; i++ {
				if i == m.lp.current {
					// Set focused state
					cmds[i] = m.lp.inputs[i].Focus()
					m.lp.inputs[i].PromptStyle = focusedStyle
					m.lp.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.lp.inputs[i].Blur()
				m.lp.inputs[i].PromptStyle = noStyle
				m.lp.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)

		}

	}

	var cmd tea.Cmd
	m.lp.inputs[m.lp.current], cmd = m.lp.inputs[m.lp.current].Update(msg)

	return m, cmd
}

func (m model) updateSecretsPage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+t" {
			m.isLoggedIn = !m.isLoggedIn
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.sp.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.sp, cmd = m.sp.Update(msg)
	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	if !m.isLoggedIn {
		return m.updateLoginPage(msg)
	}

	return m.updateSecretsPage(msg)

}

func (m model) View() string {
	//m.mx.Lock()
	//defer m.mx.Unlock()
	if !m.isLoggedIn {
		m.isLoggedIn = true

		var rend strings.Builder

		rend.WriteString(headerStyle.Render("Authorization"))

		if m.lp.lastErr != nil {
			rend.WriteString(fmt.Sprint("\nERROR:", m.lp.lastErr, "\n\n"))
		}

		for _, i := range m.lp.inputs {
			rend.WriteString(fmt.Sprintf("\n%s", i.View()))
		}

		return loginStyle.Render(rend.String())
	} else {
		m.sp.Title = "Nedovault v1.0 " + "user: " + string(m.username)
		return docStyle.Render(m.sp.View())
	}
	//return docStyle.Render(m.sp.View())
}

type UI struct {
	m model
}

func NewUI(items []list.Item, client api.NedoVaultClient) *UI {

	loginInputs := make([]textinput.Model, 0)

	ti := textinput.New()
	ti.Placeholder = "Garfield"
	ti.Focus()
	ti.TextStyle = focusedStyle
	ti.PromptStyle = focusedStyle
	ti.CharLimit = 100
	ti.Width = 100
	ti.Prompt = "Username: "
	loginInputs = append(loginInputs, ti)

	ti = textinput.New()
	ti.Placeholder = "supersecretpassword"
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 100
	ti.TextStyle = noStyle
	ti.PromptStyle = noStyle
	ti.Width = 100
	ti.Prompt = "Password: "
	loginInputs = append(loginInputs, ti)

	return &UI{
		m: model{
			sp:         list.New(items, list.NewDefaultDelegate(), 0, 0),
			lp:         loginPage{inputs: loginInputs, current: 0},
			mx:         &sync.Mutex{},
			isLoggedIn: false,
			client:     client,
		},
	}
}

func (u *UI) Run() {

	u.m.sp.Title = "Nedovault v1.0"

	p := tea.NewProgram(u.m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func (u *UI) LoginPage() {
	u.m.mx.Lock()
	u.m.isLoggedIn = false
	u.m.mx.Unlock()
}
