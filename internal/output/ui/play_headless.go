//go:build headless

package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type PlayModel struct {
	err error
}

func NewPlayModel(filepath string) PlayModel {
	return PlayModel{
		err: fmt.Errorf("play command requires audio support"),
	}
}

func (m PlayModel) Init() tea.Cmd {
	return tea.Quit
}

func (m PlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m PlayModel) View() string {
	return ""
}

func (m PlayModel) Err() error {
	return m.err
}
