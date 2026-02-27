package api

import (
	"fmt"
	"runtime"
)

func UserAgent(version string) string {
	return fmt.Sprintf("rime-cli/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
}
