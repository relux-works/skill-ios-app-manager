package components

import (
	"fmt"
	"strings"
)

// Platform is the closed set of platforms supported by generated Swift packages.
type Platform string

const (
	PlatformIOS      Platform = "iOS"
	PlatformMacOS    Platform = "macOS"
	PlatformTVOS     Platform = "tvOS"
	PlatformWatchOS  Platform = "watchOS"
	PlatformVisionOS Platform = "visionOS"
)

// ParsePlatform converts user/scaffold input into a supported platform enum value.
func ParsePlatform(raw string) (Platform, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ios":
		return PlatformIOS, nil
	case "macos", "mac":
		return PlatformMacOS, nil
	case "tvos", "tv":
		return PlatformTVOS, nil
	case "watchos", "watch":
		return PlatformWatchOS, nil
	case "visionos", "vision":
		return PlatformVisionOS, nil
	default:
		return "", fmt.Errorf(
			"unsupported platform %q; supported platforms: %s",
			raw,
			SupportedPlatformsText(),
		)
	}
}

// SupportedPlatformsText returns a stable user-facing list for error messages.
func SupportedPlatformsText() string {
	return "iOS, macOS, tvOS, watchOS, visionOS"
}
