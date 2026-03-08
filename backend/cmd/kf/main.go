package main

import (
	"fmt"
	"os"

	"kiloforge/internal/adapter/cli"
)

func main() {
	cli.SetVersionInfo(version, commit, date)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
