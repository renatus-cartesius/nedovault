package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/google/uuid"
	"github.com/renatus-cartesius/metricserv/pkg/logger"
	"github.com/renatus-cartesius/nedovault/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle     = lipgloss.NewStyle().Margin(1, 5).Padding(4, 4, 4, 4)
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})
	//loginStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63")).Align(lipgloss.Center).Padding(10, 2, 10, 2)
	loginStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63")).Margin(1, 5).Padding(10, 2, 10, 2)
	noStyle    = lipgloss.NewStyle()
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
	sp             list.Model
	selectedSecret *api.Secret

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

				ctx = metadata.AppendToOutgoingContext(ctx, "token", m.token)
				listSecretsMetaResponse, err := m.client.ListSecretsMeta(ctx, &emptypb.Empty{})
				if err != nil {
					logger.Log.Error(
						"error on listing secrets",
						zap.Error(err),
					)
				}

				var secrets []list.Item
				for _, sm := range listSecretsMetaResponse.SecretsMeta {
					secrets = append(secrets, &SecretItem{sm})
				}

				m.mx.Lock()
				m.sp.SetItems(secrets)
				m.mx.Unlock()

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
	ctx := context.Background()

	switch msg := msg.(type) {
	case tea.KeyMsg:

		s := msg.String()
		switch s {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+l":
			m.isLoggedIn = !m.isLoggedIn
		case "a":

			ctx = metadata.AppendToOutgoingContext(ctx, "token", m.token)
			_, _ = m.client.AddSecret(ctx, &api.AddSecretRequest{
				Key: []byte(fmt.Sprintf("%s-%s", "tui", uuid.NewString())),
				Secret: &api.Secret{
					Secret: &api.Secret_Text{
						Text: &api.Text{
							Data: "Hello World!",
						},
					},
				},
			},
			)

		case "enter":
			item := m.sp.Items()[m.sp.GlobalIndex()].(*SecretItem)

			ctx = metadata.AppendToOutgoingContext(ctx, "token", m.token)
			getSecretResponse, err := m.client.GetSecret(ctx, &api.GetSecretRequest{
				Key: item.SecretMeta.Key,
			})
			if err != nil {
				m.selectedSecret.Secret = nil
				return m, nil
			}

			m.selectedSecret = getSecretResponse.Secret

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

	switch mtype := msg.(type) {
	case secretsUpdate:
		m.sp.SetItems(mtype.secrets)
		return m, nil
	}

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

		var s string

		m.sp.Title = "Nedovault v1.0 "

		if m.username == "" {
			m.sp.Title += "unauthorized"
		} else {
			m.sp.Title = "user: " + string(m.username)
		}

		m.mx.Lock()
		s += docStyle.Render(m.sp.View())
		m.mx.Unlock()

		if m.selectedSecret != nil {
			s += lipgloss.JoinVertical(lipgloss.Right, fmt.Sprintf("%s", m.selectedSecret.String()))
		}

		return s
	}
	//return docStyle.Render(m.sp.View())
}

type UI struct {
	m model
}

type secretsUpdate struct {
	secrets []list.Item
}

func (u *UI) CheckUpdates(ctx context.Context, wg *sync.WaitGroup, p *tea.Program) {
	//defer wg.Done()

	stream, err := u.m.client.ListSecretsMetaStream(ctx, &emptypb.Empty{})
	if err != nil {
		panic(err)
	}

	// TOOD: add gracefull shutdown
	for {

		select {
		case <-ctx.Done():
			return
		default:
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}

			if err != nil {
				logger.Log.Error(
					"error receiving metadata from server",
					zap.Error(err),
				)
			}

			var secrets []list.Item

			for _, sm := range resp.SecretsMeta {
				secrets = append(secrets, &SecretItem{sm})
			}

			p.Send(secretsUpdate{
				secrets: secrets,
			})
		}

	}
}

func NewUI(client api.NedoVaultClient) *UI {

	var items []list.Item

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

	sp := list.New(items, list.NewDefaultDelegate(), 0, 0)
	//sp.SetSize(docStyle.GetFrameSize())

	return &UI{
		m: model{
			sp: sp,
			selectedSecret: &api.Secret{
				Secret: nil,
			},
			lp:         loginPage{inputs: loginInputs, current: 0},
			mx:         &sync.Mutex{},
			isLoggedIn: false,
			client:     client,
		},
	}
}

func (u *UI) Run() {

	u.m.sp.Title = "Nedovault v1.0"

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := tea.NewProgram(u.m, tea.WithAltScreen())

	//wg.Add(1)

	go u.CheckUpdates(ctx, wg, p)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	//wg.Wait()
}

func (u *UI) LoginPage() {
	u.m.mx.Lock()
	u.m.isLoggedIn = false
	u.m.mx.Unlock()
}
