package relux

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var swiftIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type moduleLayout struct {
	ModulePath          string
	InterfaceSourcesDir string
	ImplSourcesDir      string
	IoCDir              string
}

func resolveModuleLayout(moduleName string, modulePath string) (moduleLayout, error) {
	normalizedModuleName, err := normalizeModuleName(moduleName)
	if err != nil {
		return moduleLayout{}, err
	}

	trimmedPath := strings.TrimSpace(modulePath)
	if trimmedPath == "" {
		return moduleLayout{}, errors.New("module path is required")
	}

	// modulePath = .../Packages/ModuleName (interface package root)
	// impl package = .../Packages/ModuleNameImpl (sibling directory)
	implPackagePath := trimmedPath + "Impl"

	layout := moduleLayout{
		ModulePath: trimmedPath,
	}

	// SPM standard: Sources/{ModuleName}/ inside the package
	// Legacy: {ModuleName}Interface/Sources/ inside the interface package
	layout.InterfaceSourcesDir = pickExistingDirOrDefault(
		trimmedPath,
		[]string{
			filepath.Join("Sources", normalizedModuleName),
			filepath.Join(normalizedModuleName+"Interface", "Sources"),
			filepath.Join("Interface", "Sources"),
		},
		filepath.Join("Sources", normalizedModuleName),
	)

	// SPM standard: impl is a separate package at .../Packages/ModuleNameImpl/Sources/ModuleNameImpl/
	// Legacy: {ModuleName}Impl/Sources/ inside the interface package
	layout.ImplSourcesDir = pickExistingDirOrDefault(
		"",
		[]string{
			filepath.Join(implPackagePath, "Sources", normalizedModuleName+"Impl"),
			filepath.Join(trimmedPath, normalizedModuleName+"Impl", "Sources"),
			filepath.Join(trimmedPath, "Impl", "Sources"),
		},
		filepath.Join(implPackagePath, "Sources", normalizedModuleName+"Impl"),
	)

	// IoC files go into the impl sources dir (no separate IoC directory)
	layout.IoCDir = layout.ImplSourcesDir

	return layout, nil
}

func normalizeModuleName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("module name is required")
	}

	// If already a valid Swift identifier, preserve as-is (keeps PascalCase like "TodoList")
	if swiftIdentifierPattern.MatchString(trimmed) {
		return trimmed, nil
	}

	name, err := toPascalIdentifier(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid module name: %w", err)
	}

	if !swiftIdentifierPattern.MatchString(name) {
		return "", fmt.Errorf("module name %q is not a valid Swift identifier", name)
	}

	return name, nil
}

func normalizeActionName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("action name is required")
	}

	if !swiftIdentifierPattern.MatchString(trimmed) {
		return "", fmt.Errorf("action name %q is not a valid Swift identifier", trimmed)
	}

	return lowerFirst(trimmed), nil
}

func normalizeMiddlewareName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("middleware name is required")
	}

	name, err := toPascalIdentifier(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid middleware name: %w", err)
	}

	if !swiftIdentifierPattern.MatchString(name) {
		return "", fmt.Errorf("middleware name %q is not a valid Swift identifier", name)
	}

	return name, nil
}

func toPascalIdentifier(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("value is empty")
	}

	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
			continue
		}
		flush()
	}
	flush()

	if len(tokens) == 0 {
		return "", fmt.Errorf("%q contains no identifier characters", raw)
	}

	var out strings.Builder
	for _, token := range tokens {
		if token == "" {
			continue
		}

		first, size := utf8.DecodeRuneInString(token)
		out.WriteRune(unicode.ToUpper(first))
		out.WriteString(strings.ToLower(token[size:]))
	}

	result := out.String()
	if result == "" {
		return "", fmt.Errorf("%q is not convertible to identifier", raw)
	}

	return result, nil
}

func lowerFirst(input string) string {
	if input == "" {
		return ""
	}

	r, size := utf8.DecodeRuneInString(input)
	return string(unicode.ToLower(r)) + input[size:]
}

func pickExistingDirOrDefault(root string, candidates []string, fallback string) string {
	for _, candidate := range candidates {
		absoluteCandidate := filepath.Join(root, candidate)
		if isExistingDir(absoluteCandidate) {
			return absoluteCandidate
		}
	}

	if filepath.IsAbs(fallback) {
		return fallback
	}

	return filepath.Join(root, fallback)
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func isExistingFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}

	return nil
}

func readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return content, nil
}

func findFileByName(moduleRoot string, fileName string, preferred []string) (string, error) {
	for _, path := range preferred {
		if isExistingFile(path) {
			return path, nil
		}
	}

	var matches []string
	walkErr := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), fileName) {
			matches = append(matches, path)
		}
		return nil
	})
	if walkErr != nil {
		return "", fmt.Errorf("scan module path %q for %q: %w", moduleRoot, fileName, walkErr)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("file %q not found under %q", fileName, moduleRoot)
	}

	sort.Strings(matches)
	return matches[0], nil
}

func findMatchingBrace(content string, openingBraceIndex int) (int, error) {
	if openingBraceIndex < 0 || openingBraceIndex >= len(content) || content[openingBraceIndex] != '{' {
		return 0, errors.New("opening brace index does not point at '{'")
	}

	depth := 0
	for i := openingBraceIndex; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}

	return 0, errors.New("failed to find matching closing brace")
}

func toSnakeCase(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	var previousWasUnderscore bool

	for i, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if i > 0 && unicode.IsUpper(r) {
				lastRune, _ := utf8.DecodeLastRuneInString(builder.String())
				if lastRune != '_' && !previousWasUnderscore {
					builder.WriteByte('_')
				}
			}
			builder.WriteRune(unicode.ToLower(r))
			previousWasUnderscore = false
			continue
		}

		if !previousWasUnderscore && builder.Len() > 0 {
			builder.WriteByte('_')
			previousWasUnderscore = true
		}
	}

	out := strings.Trim(builder.String(), "_")
	out = strings.ReplaceAll(out, "__", "_")
	if out == "" {
		return "middleware"
	}

	return out
}
