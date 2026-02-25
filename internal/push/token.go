package push

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// ErrDeviceTokenNotFound indicates that no APNs token was found in the inspected source.
	ErrDeviceTokenNotFound = errors.New("no APNs device token found")

	hexTokenPattern       = regexp.MustCompile(`(?i)\b[0-9a-f]{64}\b`)
	groupedTokenPattern   = regexp.MustCompile(`(?i)<([0-9a-f ]+)>`)
	simctlLogCommandParts = []string{
		"simctl", "spawn", "booted", "log", "show", "--last", "1h", "--style", "compact",
	}
)

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
type fileReader func(path string) ([]byte, error)

// TokenExtractor reads APNs device tokens from simulator logs and optional fallback files.
type TokenExtractor struct {
	runCommand commandRunner
	readFile   fileReader
}

// NewTokenExtractor builds a production extractor.
func NewTokenExtractor() *TokenExtractor {
	return &TokenExtractor{
		runCommand: defaultCommandRunner,
		readFile:   os.ReadFile,
	}
}

// LatestDeviceToken returns the newest APNs device token from simulator logs.
// If no token is found from logs and fallbackTokenFilePath is set, it also tries that file.
func (e *TokenExtractor) LatestDeviceToken(ctx context.Context, fallbackTokenFilePath string) (string, error) {
	extractor := e
	if extractor == nil {
		extractor = NewTokenExtractor()
	}

	token, err := extractor.latestFromSimulatorLogs(ctx)
	if err == nil {
		return token, nil
	}

	fallbackPath := strings.TrimSpace(fallbackTokenFilePath)
	if fallbackPath == "" {
		return "", err
	}

	fileToken, fileErr := extractor.latestFromTokenFile(fallbackPath)
	if fileErr != nil {
		return "", fmt.Errorf(
			"simulator token extraction failed (%v); fallback token file extraction failed: %w",
			err,
			fileErr,
		)
	}

	return fileToken, nil
}

func (e *TokenExtractor) latestFromSimulatorLogs(ctx context.Context) (string, error) {
	output, err := e.runCommand(ctx, "xcrun", simctlLogCommandParts...)
	if err != nil {
		return "", fmt.Errorf("run simulator log command: %w", err)
	}

	token, ok := extractLatestAPNsToken(string(output))
	if !ok {
		return "", ErrDeviceTokenNotFound
	}

	return token, nil
}

func (e *TokenExtractor) latestFromTokenFile(path string) (string, error) {
	content, err := e.readFile(path)
	if err != nil {
		return "", fmt.Errorf("read token file %q: %w", path, err)
	}

	token, ok := extractLatestHexToken(string(content))
	if !ok {
		return "", fmt.Errorf("%w in %q", ErrDeviceTokenNotFound, path)
	}

	return token, nil
}

func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return output, err
		}

		return output, fmt.Errorf("%w: %s", err, msg)
	}

	return output, nil
}

func extractLatestAPNsToken(logOutput string) (string, bool) {
	lines := strings.Split(logOutput, "\n")
	latest := ""

	for _, line := range lines {
		if !isLikelyAPNsRegistrationLine(line) {
			continue
		}

		tokens := extractHexTokens(line)
		if len(tokens) == 0 {
			continue
		}

		latest = tokens[len(tokens)-1]
	}

	if latest == "" {
		return "", false
	}

	return latest, true
}

func isLikelyAPNsRegistrationLine(line string) bool {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "token") {
		return false
	}

	if strings.Contains(lower, "didregisterforremotenotificationswithdevicetoken") {
		return true
	}

	if strings.Contains(lower, "remote notification") {
		return true
	}

	if strings.Contains(lower, "apns") {
		return true
	}

	return strings.Contains(lower, "device token") && strings.Contains(lower, "push")
}

func extractLatestHexToken(input string) (string, bool) {
	tokens := extractHexTokens(input)
	if len(tokens) == 0 {
		return "", false
	}

	return tokens[len(tokens)-1], true
}

func extractHexTokens(input string) []string {
	matches := hexTokenPattern.FindAllString(input, -1)
	tokens := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))

	for _, match := range matches {
		token := strings.ToLower(match)
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}

	groupMatches := groupedTokenPattern.FindAllStringSubmatch(input, -1)
	for _, match := range groupMatches {
		if len(match) < 2 {
			continue
		}

		candidate := strings.ToLower(strings.ReplaceAll(match[1], " ", ""))
		if !hexTokenPattern.MatchString(candidate) {
			continue
		}

		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		tokens = append(tokens, candidate)
	}

	return tokens
}
