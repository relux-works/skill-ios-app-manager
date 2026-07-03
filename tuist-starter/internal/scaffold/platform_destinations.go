package scaffold

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templaterenderer "github.com/relux-works/ios-app-manager/internal/template"
)

type PlatformDestinationsSyncResult struct {
	Scanned []string
	Updated []string
}

var swiftDestinationsArgumentPattern = regexp.MustCompile(`(destinations:\s*)(?:\[[^\]]*\]|\.[A-Za-z0-9]+)(,)`)

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "platform-destinations",
		Short:        "Sync host app destination platforms",
		Dependencies: []string{"init"},
		Run:          runGeneratePlatformDestinations,
	})
}

func runGeneratePlatformDestinations(input GenerateInput) (GenerateResult, error) {
	result, err := SyncPlatformDestinations(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) > 0 {
		return GenerateResult{
			Message: fmt.Sprintf("regenerated platform destinations in %d file(s)\n", len(result.Updated)),
		}, nil
	}

	return GenerateResult{
		Message: "platform destinations already up to date\n",
	}, nil
}

func SyncPlatformDestinations(projectRoot string, cfg config.ProjectConfig) (PlatformDestinationsSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return PlatformDestinationsSyncResult{}, fmt.Errorf("project root is required")
	}

	result := PlatformDestinationsSyncResult{
		Scanned: make([]string, 0, 4),
		Updated: make([]string, 0, 4),
	}

	projectManifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return result, err
	}

	for _, manifestPath := range projectManifestPaths {
		result.Scanned = appendUniqueStrings(result.Scanned, manifestPath)
		updated, err := syncProjectManifestPlatformDestinations(manifestPath, cfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}
	}

	return result, nil
}

func syncProjectManifestPlatformDestinations(path string, cfg config.ProjectConfig) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}

	updated, changed, err := syncProjectManifestPlatformDestinationsContent(string(payload), cfg)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Project.swift: %w", err)
	}

	return true, nil
}

func syncProjectManifestPlatformDestinationsContent(content string, cfg config.ProjectConfig) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	targets := findProjectTargetBlocks(lines)
	if len(targets) == 0 {
		return "", false, fmt.Errorf("Project.swift target declarations not found")
	}

	expression := templaterenderer.TargetDestinationsExpression(cfg)
	updatedLines := lines
	for index := len(targets) - 1; index >= 0; index-- {
		target := targets[index]
		targetLines := append([]string(nil), updatedLines[target.start:target.end+1]...)
		if !targetReceivesPresentationConfig(targetLines) {
			continue
		}

		nextTargetLines := syncTargetPlatformDestinationsLines(targetLines, expression)
		nextLines := make([]string, 0, len(updatedLines)-len(targetLines)+len(nextTargetLines))
		nextLines = append(nextLines, updatedLines[:target.start]...)
		nextLines = append(nextLines, nextTargetLines...)
		nextLines = append(nextLines, updatedLines[target.end+1:]...)
		updatedLines = nextLines
	}

	updated := joinSyncLines(updatedLines, hasTrailingNewline)
	return updated, updated != content, nil
}

func syncTargetPlatformDestinationsLines(lines []string, expression string) []string {
	updated := make([]string, 0, len(lines))
	for _, line := range lines {
		updated = append(updated, replaceSwiftDestinationsArgument(line, expression))
	}
	return updated
}

func replaceSwiftDestinationsArgument(line string, expression string) string {
	if !strings.Contains(line, "destinations:") {
		return line
	}
	return swiftDestinationsArgumentPattern.ReplaceAllString(line, "${1}"+expression+"${2}")
}
