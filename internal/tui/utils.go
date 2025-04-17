package tui

import "github.com/charmbracelet/bubbles/textinput"

func isInputsFilled(inputs []textinput.Model) bool {
	for _, i := range inputs {
		if i.Value() == "" {
			return false
		}
	}

	return true
}
