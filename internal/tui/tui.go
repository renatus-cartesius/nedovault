package tui

import (
	"fmt"
	"github.com/renatus-cartesius/nedovault/api"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 5)

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

type model struct {
	list list.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

type UI struct {
	m model
}

func NewUI(items []list.Item) *UI {

	return &UI{
		m: model{
			list: list.New(items, list.NewDefaultDelegate(), 0, 0),
		},
	}
}

func (u *UI) Run() {

	u.m.list.Title = "Nedovault v1.0"

	p := tea.NewProgram(u.m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
