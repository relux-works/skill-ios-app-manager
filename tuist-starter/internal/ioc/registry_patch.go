package ioc

import (
	"fmt"
	"os"
	"strings"
)

const (
	registryFoundationAnchor         = "// MARK: - Foundation (scaffolding anchor: foundation)"
	registryFoundationBuildersAnchor = "// MARK: - Foundation Builders (scaffolding anchor: foundation-builders)"
)

// RegistryFoundationPatch describes one non-destructive addition to the
// Foundation section of an existing Registry.swift composition root.
type RegistryFoundationPatch struct {
	Imports            []string
	RegistrationMarker string
	RegistrationLine   string
	BuilderMarker      string
	BuilderFunction    string
}

// PatchFoundationRegistry applies a focused registration/builder patch while
// preserving every unrelated import, registration, helper, and builder.
func PatchFoundationRegistry(path string, appTypeName string, patch RegistryFoundationPatch) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Registry.swift: %w", err)
	}

	updated, err := PatchFoundationRegistryContent(string(payload), appTypeName, patch)
	if err != nil {
		return err
	}
	if updated == string(payload) {
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Registry.swift: %w", err)
	}
	return nil
}

// PatchFoundationRegistryContent is the pure content form used by setup
// plugins and regression tests.
func PatchFoundationRegistryContent(content string, appTypeName string, patch RegistryFoundationPatch) (string, error) {
	if strings.TrimSpace(patch.RegistrationMarker) == "" || strings.TrimSpace(patch.RegistrationLine) == "" {
		return "", fmt.Errorf("Registry foundation patch requires a registration marker and line")
	}
	if strings.TrimSpace(patch.BuilderMarker) == "" || strings.TrimSpace(patch.BuilderFunction) == "" {
		return "", fmt.Errorf("Registry foundation patch requires a builder marker and function")
	}

	updated := content
	for _, moduleName := range patch.Imports {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" {
			continue
		}
		updated = EnsureImport(updated, moduleName)
	}

	if !strings.Contains(updated, patch.RegistrationMarker) {
		anchorIndex := strings.Index(updated, registryFoundationAnchor)
		if anchorIndex < 0 {
			return "", fmt.Errorf("foundation anchor not found in Registry.swift")
		}
		lineStart := strings.LastIndex(updated[:anchorIndex], "\n") + 1
		lineEndOffset := strings.Index(updated[anchorIndex:], "\n")
		lineEnd := len(updated)
		if lineEndOffset >= 0 {
			lineEnd = anchorIndex + lineEndOffset
		}
		indent := leadingRegistryIndent(updated[lineStart:anchorIndex])
		registration := indent + strings.TrimSpace(patch.RegistrationLine)
		updated = updated[:lineEnd] + "\n" + registration + updated[lineEnd:]
	}

	if strings.Contains(updated, patch.BuilderMarker) {
		return updated, nil
	}

	buildersIndex := strings.Index(updated, registryFoundationBuildersAnchor)
	if buildersIndex < 0 {
		appTypeName = strings.TrimSpace(appTypeName)
		if appTypeName == "" {
			return "", fmt.Errorf("app type name is required when Registry.swift has no foundation-builders anchor")
		}
		builder := indentRegistryBlock(strings.TrimSpace(patch.BuilderFunction), "    ")
		updated = strings.TrimRight(updated, "\n") +
			"\n\n" + registryFoundationBuildersAnchor +
			"\nextension " + appTypeName + ".Registry {\n" +
			builder + "\n}\n"
		return updated, nil
	}

	openingOffset := strings.Index(updated[buildersIndex:], "{")
	if openingOffset < 0 {
		return "", fmt.Errorf("extension opening brace not found after foundation-builders anchor")
	}
	openingIndex := buildersIndex + openingOffset
	closingIndex := matchingRegistryBrace(updated, openingIndex)
	if closingIndex < 0 {
		return "", fmt.Errorf("extension closing brace not found after foundation-builders anchor")
	}

	extensionLineStart := strings.LastIndex(updated[:openingIndex], "\n") + 1
	extensionIndent := leadingRegistryIndent(updated[extensionLineStart:openingIndex])
	builder := indentRegistryBlock(strings.TrimSpace(patch.BuilderFunction), extensionIndent+"    ")
	insertion := builder + "\n"
	if closingIndex > 0 && updated[closingIndex-1] != '\n' {
		insertion = "\n" + insertion
	}
	updated = updated[:closingIndex] + insertion + updated[closingIndex:]
	return updated, nil
}

func leadingRegistryIndent(line string) string {
	end := 0
	for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
		end++
	}
	return line[:end]
}

func indentRegistryBlock(block string, indent string) string {
	lines := strings.Split(block, "\n")
	for index := range lines {
		if strings.TrimSpace(lines[index]) == "" {
			lines[index] = ""
			continue
		}
		lines[index] = indent + lines[index]
	}
	return strings.Join(lines, "\n")
}

func matchingRegistryBrace(content string, openingIndex int) int {
	depth := 0
	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false

	for index := openingIndex; index < len(content); index++ {
		ch := content[index]
		next := byte(0)
		if index+1 < len(content) {
			next = content[index+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				index++
			}
			continue
		}
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '/' && next == '/' {
			inLineComment = true
			index++
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			index++
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}
