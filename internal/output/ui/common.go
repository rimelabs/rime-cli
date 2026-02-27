package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/output/visualizer"
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

// BoxOverhead is the number of columns consumed by the box borders and padding
// around the logo and right panel: "│ " + logo + " │ " + right + " │" = logoWidth + 7.
var BoxOverhead = UnboxedLogoWidth() + 7

// RenderBoxLayout renders a two-column box with the rime logo on the left
// and the given right-side content lines on the right.
func RenderBoxLayout(rightContentWidth int, rightLines []string) string {
	logo := UnboxedLogoPlain()
	logoLines := strings.Split(logo, "\n")
	logoWidth := UnboxedLogoWidth()

	topSep := DimStyle.Render("┌" + strings.Repeat("─", logoWidth+2) + "┬" + strings.Repeat("─", rightContentWidth+2) + "┐")

	maxRows := len(logoLines)
	if len(rightLines) > maxRows {
		maxRows = len(rightLines)
	}

	var body strings.Builder
	for i := 0; i < maxRows; i++ {
		var leftCell, rightCell string
		if i < len(logoLines) {
			leftCell = logoLines[i]
		} else {
			leftCell = strings.Repeat(" ", logoWidth)
		}
		if i < len(rightLines) {
			line := rightLines[i]
			visibleWidth := lipgloss.Width(line)
			padding := rightContentWidth - visibleWidth
			if padding < 0 {
				padding = 0
			}
			rightCell = line + strings.Repeat(" ", padding)
		} else {
			rightCell = strings.Repeat(" ", rightContentWidth)
		}
		body.WriteString(DimStyle.Render("│") + " " + leftCell + " " + DimStyle.Render("│") + " " + rightCell + " " + DimStyle.Render("│") + "\n")
	}

	botSep := DimStyle.Render("└" + strings.Repeat("─", logoWidth+2) + "┴" + strings.Repeat("─", rightContentWidth+2) + "┘")

	return topSep + "\n" + body.String() + botSep + "\n"
}

// RenderLabeledHeader builds a header like: speaker: astra  model: arcana  lang: eng
func RenderLabeledHeader(speaker, model, lang string) string {
	return DimStyle.Render("speaker: ") + speaker +
		DimStyle.Render("  model: ") + model +
		DimStyle.Render("  lang: ") + lang
}

// RenderRightPanel builds the right-side content lines for the player box layout.
func RenderRightPanel(header string, width int, transcript *visualizer.Transcript, elapsed, total string, waveform *visualizer.Waveform) []string {
	var right strings.Builder

	var timeStr string
	if total != "" {
		timeStr = DimStyle.Render(fmt.Sprintf("[%s / %s]", elapsed, total))
	} else {
		timeStr = DimStyle.Render(fmt.Sprintf("[%s]", elapsed))
	}

	timeWidth := lipgloss.Width(timeStr)
	maxHeaderWidth := width - timeWidth - 2
	if maxHeaderWidth < 0 {
		maxHeaderWidth = 0
	}
	headerWidth := lipgloss.Width(header)
	if headerWidth > maxHeaderWidth {
		header = lipgloss.NewStyle().MaxWidth(maxHeaderWidth).Render(header)
		headerWidth = lipgloss.Width(header)
	}
	gap := width - headerWidth - timeWidth
	if gap < 2 {
		gap = 2
	}
	right.WriteString(header + strings.Repeat(" ", gap) + timeStr + "\n")

	right.WriteString("\n")
	if transcript != nil {
		right.WriteString(DimStyle.Render("text: ") + transcript.Render() + "\n")
		right.WriteString("\n")
	}
	if waveform != nil {
		right.WriteString(waveform.RenderTop() + "\n")
		right.WriteString(waveform.RenderBot())
	}
	return strings.Split(right.String(), "\n")
}
