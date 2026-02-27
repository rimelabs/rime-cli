package ui

import (
	"strings"
)

const rimeLogo = "" +
	"⡔⠉⠉⢢ ⡔⠉⠉⢢ ⡔⠉⠉⢢ ⡔⠉⠉⢢\n" +
	"⡇⠀⠀⢸ ⡇⠀⠀⢸ ⡇⠀⠀⢸ ⠣⣀⣀⠜\n" +
	"⡇⠀⠀⢸ ⠣⣀⣀⠜ ⡇⠀⠀⢸ ⡔⠉⠉⢢\n" +
	"⡇⠀⠀⢸ ⡔⠉⠉⢢ ⠣⣀⣀⠜ ⡇⠀⠀⢸\n" +
	"⠣⣀⣀⠜ ⡇⠀⠀⢸ ⡔⠉⠉⢢ ⡇⠀⠀⢸\n" +
	"⡔⠉⠉⢢ ⡇⠀⠀⢸ ⡇⠀⠀⢸ ⡇⠀⠀⢸\n" +
	"⠣⣀⣀⠜ ⠣⣀⣀⠜ ⠣⣀⣀⠜ ⠣⣀⣀⠜"

const rimeLogoWidth = 19

func PaddedLogo() string {
	lines := strings.Split(rimeLogo, "\n")
	pad := "  "

	var b strings.Builder
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString(pad + line + "\n")
	}
	return b.String()
}

func UnboxedLogoPlain() string {
	return rimeLogo
}

func UnboxedLogoWidth() int {
	return rimeLogoWidth
}
