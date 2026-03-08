package main

import (
	"fmt"
	"os"

	"kiloforge/internal/adapter/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
