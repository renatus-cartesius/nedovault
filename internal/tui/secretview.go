package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/renatus-cartesius/nedovault/api"
)

type SecretView struct {
	Secret *api.Secret
	style  lipgloss.Style

	ta textarea.Model
}

func NewSecretView() *SecretView {
	ta := textarea.New()

	ta.Placeholder = "Secret data"

	return &SecretView{
		Secret: nil,
		style:  lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("63")).Padding(10, 10, 10, 10).AlignVertical(lipgloss.Center).AlignHorizontal(lipgloss.Center),
		ta:     ta,
	}
}

func (sv *SecretView) Update(secret *api.Secret) {
	sv.ta, _ = sv.ta.Update("sdf")
}

func (sv *SecretView) View() string {

	return sv.style.Render(sv.Secret.String())
}
