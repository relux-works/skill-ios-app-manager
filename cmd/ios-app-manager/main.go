package main

import (
	"fmt"
	"os"

	"github.com/relux-works/ios-app-manager/internal/cli"
	_ "github.com/relux-works/ios-app-manager/internal/appintents"
	_ "github.com/relux-works/ios-app-manager/internal/extensions"
	_ "github.com/relux-works/ios-app-manager/internal/liveactivity"
	_ "github.com/relux-works/ios-app-manager/internal/staticwidget"
	_ "github.com/relux-works/ios-app-manager/internal/widgetbase"
)

var Version = "dev"

func main() {
	cli.SetVersion(Version)

	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
