package styles

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var noColor bool

func init() {
	noColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}

var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	ErrorLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true).
			SetString("Error: ")

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	SuccessLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Bold(true).
				SetString("✓ ")

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

func Error(msg string) string {
	if noColor {
		return "Error: " + msg
	}
	return ErrorLabelStyle.String() + ErrorStyle.Render(msg)
}

func Success(msg string) string {
	if noColor {
		return "✓ " + msg
	}
	return SuccessLabelStyle.String() + SuccessStyle.Render(msg)
}

func Successf(format string, a ...interface{}) string {
	return Success(fmt.Sprintf(format, a...))
}

func Info(msg string) string {
	if noColor {
		return msg
	}
	return InfoStyle.Render(msg)
}

func Dim(msg string) string {
	if noColor {
		return msg
	}
	return DimStyle.Render(msg)
}
