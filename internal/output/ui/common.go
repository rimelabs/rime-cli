package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var Spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var HeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
var DimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

func GetTerminalWidth() int {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width < 40 {
		return 80
	}
	return width
}

func RenderSeparator(width int) string {
	return DimStyle.Render(strings.Repeat("─", width))
}
