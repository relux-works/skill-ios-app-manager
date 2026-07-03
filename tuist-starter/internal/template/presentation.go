package template

import (
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	infoPlistKeyUserInterfaceStyle             = "UIUserInterfaceStyle"
	infoPlistKeySupportedInterfaceOrientations = "UISupportedInterfaceOrientations"
	infoPlistKeyIPadSupportedOrientations      = "UISupportedInterfaceOrientations~ipad"
)

func presentationInfoPlistLines(value any) []string {
	cfg := projectConfigFromTemplateValue(value)
	lines := make([]string, 0, 10)

	switch strings.ToLower(strings.TrimSpace(cfg.Theme)) {
	case config.ThemeLight:
		lines = append(lines, strconv.Quote(infoPlistKeyUserInterfaceStyle)+": .string("+strconv.Quote("Light")+"),")
	case config.ThemeDark:
		lines = append(lines, strconv.Quote(infoPlistKeyUserInterfaceStyle)+": .string("+strconv.Quote("Dark")+"),")
	}

	if !cfg.UsesExplicitPlatformDestinations() {
		return appendOrientationInfoPlistLines(lines, infoPlistKeySupportedInterfaceOrientations, cfg.Orientation)
	}

	if cfg.IOSTargetEnabled() {
		lines = appendOrientationInfoPlistLines(lines, infoPlistKeySupportedInterfaceOrientations, cfg.IOSTargetOrientation())
	}
	if cfg.IPadTargetEnabled() {
		lines = appendOrientationInfoPlistLines(lines, infoPlistKeyIPadSupportedOrientations, cfg.IPadTargetOrientation())
	}

	return lines
}

func appendOrientationInfoPlistLines(lines []string, key string, orientation string) []string {
	switch strings.ToLower(strings.TrimSpace(orientation)) {
	case config.OrientationPortrait:
		return append(lines,
			strconv.Quote(key)+": .array([",
			"    .string("+strconv.Quote("UIInterfaceOrientationPortrait")+"),",
			"]),",
		)
	case config.OrientationLandscape:
		return append(lines,
			strconv.Quote(key)+": .array([",
			"    .string("+strconv.Quote("UIInterfaceOrientationLandscapeLeft")+"),",
			"    .string("+strconv.Quote("UIInterfaceOrientationLandscapeRight")+"),",
			"]),",
		)
	}

	return lines
}
