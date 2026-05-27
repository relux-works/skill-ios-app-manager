package template

import (
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func presentationInfoPlistLines(theme string, orientation string) []string {
	lines := make([]string, 0, 6)

	switch strings.ToLower(strings.TrimSpace(theme)) {
	case config.ThemeLight:
		lines = append(lines, strconv.Quote("UIUserInterfaceStyle")+": .string("+strconv.Quote("Light")+"),")
	case config.ThemeDark:
		lines = append(lines, strconv.Quote("UIUserInterfaceStyle")+": .string("+strconv.Quote("Dark")+"),")
	}

	switch strings.ToLower(strings.TrimSpace(orientation)) {
	case config.OrientationPortrait:
		lines = append(lines,
			strconv.Quote("UISupportedInterfaceOrientations")+": .array([",
			"    .string("+strconv.Quote("UIInterfaceOrientationPortrait")+"),",
			"]),",
		)
	case config.OrientationLandscape:
		lines = append(lines,
			strconv.Quote("UISupportedInterfaceOrientations")+": .array([",
			"    .string("+strconv.Quote("UIInterfaceOrientationLandscapeLeft")+"),",
			"    .string("+strconv.Quote("UIInterfaceOrientationLandscapeRight")+"),",
			"]),",
		)
	}

	return lines
}
