package main

import (
	"fmt"
	"os"

	"github.com/rimelabs/rime-cli/cmd"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

var version = "dev"

func main() {
	rootCmd := cmd.NewRootCmd(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, styles.Error(err.Error()))
		os.Exit(1)
	}
}
