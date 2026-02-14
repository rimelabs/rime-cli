package formatters

import "fmt"

func FormatBytes(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	kb := float64(b) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.1fMB", mb)
}
