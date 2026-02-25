package main

import (
	"fmt"
	"os"

	"github.com/relux-works/ios-app-manager/internal/cli"
)

var Version = "dev"

func main() {
	cli.SetVersion(Version)

	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
