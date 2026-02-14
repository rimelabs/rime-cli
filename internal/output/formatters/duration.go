package formatters

import (
	"fmt"
	"time"
)

func FormatDuration(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
