package tuistproj

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// MinSupportedMajorVersion is the minimum supported tuist major version.
	MinSupportedMajorVersion = 4
)

var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// VersionInfo is parsed from `tuist version` output.
type VersionInfo struct {
	Raw     string
	Version string
	Major   int
	Minor   int
	Patch   int
}

// VersionParseError indicates output that could not be parsed as a version.
type VersionParseError struct {
	Output string
}

func (e *VersionParseError) Error() string {
	return fmt.Sprintf("unable to parse tuist version from output: %q", e.Output)
}

// VersionUnsupportedError indicates version lower than required minimum.
type VersionUnsupportedError struct {
	Detected VersionInfo
	MinMajor int
}

func (e *VersionUnsupportedError) Error() string {
	return fmt.Sprintf(
		"tuist version %s is unsupported; require %d.x or newer",
		e.Detected.Version,
		e.MinMajor,
	)
}

// ParseVersionOutput parses the first semantic version found in command output.
func ParseVersionOutput(output string) (VersionInfo, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return VersionInfo{}, &VersionParseError{Output: output}
	}

	match := versionPattern.FindStringSubmatch(trimmed)
	if match == nil {
		return VersionInfo{}, &VersionParseError{Output: trimmed}
	}

	major, err := strconv.Atoi(match[1])
	if err != nil {
		return VersionInfo{}, &VersionParseError{Output: trimmed}
	}

	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return VersionInfo{}, &VersionParseError{Output: trimmed}
	}

	patch := 0
	if match[3] != "" {
		patch, err = strconv.Atoi(match[3])
		if err != nil {
			return VersionInfo{}, &VersionParseError{Output: trimmed}
		}
	}

	parsedVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	return VersionInfo{
		Raw:     trimmed,
		Version: parsedVersion,
		Major:   major,
		Minor:   minor,
		Patch:   patch,
	}, nil
}

// ValidateMinimumMajorVersion validates the required major version.
func ValidateMinimumMajorVersion(info VersionInfo, minMajor int) error {
	if info.Major < minMajor {
		return &VersionUnsupportedError{
			Detected: info,
			MinMajor: minMajor,
		}
	}
	return nil
}

// CheckVersion runs `tuist version`, parses output and validates support level.
func CheckVersion(ctx context.Context, runner Runner) (VersionInfo, error) {
	if runner == nil {
		return VersionInfo{}, fmt.Errorf("runner is nil")
	}

	result, err := runner.Run(ctx, CommandVersion)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("run tuist version: %w", err)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		output = strings.TrimSpace(result.Stderr)
	}

	info, err := ParseVersionOutput(output)
	if err != nil {
		return VersionInfo{}, err
	}

	if err := ValidateMinimumMajorVersion(info, MinSupportedMajorVersion); err != nil {
		return info, err
	}

	return info, nil
}

// CheckVersion validates the version using this runner instance.
func (r *TuistRunner) CheckVersion(ctx context.Context) (VersionInfo, error) {
	return CheckVersion(ctx, r)
}
