package tuistproj

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	packageSettingsDeclaration = "let packageSettings = PackageSettings("
	packageProductTypesAnchor  = "productTypes: ["
)

// EnsureFrameworkProductTypes forces the provided Swift package products to be
// generated as frameworks in root Package.swift under #if TUIST PackageSettings.
func EnsureFrameworkProductTypes(path string, productNames ...string) error {
	normalized := normalizeFrameworkProductNames(productNames)
	if len(normalized) == 0 {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Package.swift: %w", err)
	}

	updated, err := ensureFrameworkProductTypesContent(string(content), normalized)
	if err != nil {
		return err
	}

	if updated == string(content) {
		return nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Package.swift: %w", err)
	}

	return nil
}

// EnsureFrameworkProductTypesInContent forces the provided Swift package
// products to be generated as frameworks in a root Package.swift payload.
func EnsureFrameworkProductTypesInContent(content string, productNames ...string) (string, error) {
	normalized := normalizeFrameworkProductNames(productNames)
	if len(normalized) == 0 {
		return content, nil
	}

	return ensureFrameworkProductTypesContent(content, normalized)
}

// RemoveFrameworkProductTypes removes the provided framework product overrides
// from root Package.swift PackageSettings when present.
func RemoveFrameworkProductTypes(path string, productNames ...string) error {
	normalized := normalizeFrameworkProductNames(productNames)
	if len(normalized) == 0 {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Package.swift: %w", err)
	}

	updated, err := removeFrameworkProductTypesContent(string(content), normalized)
	if err != nil {
		return err
	}

	if updated == string(content) {
		return nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Package.swift: %w", err)
	}

	return nil
}

// RemoveFrameworkProductTypesInContent removes the provided framework product
// overrides from a root Package.swift payload.
func RemoveFrameworkProductTypesInContent(content string, productNames ...string) (string, error) {
	normalized := normalizeFrameworkProductNames(productNames)
	if len(normalized) == 0 {
		return content, nil
	}

	return removeFrameworkProductTypesContent(content, normalized)
}

func ensureFrameworkProductTypesContent(content string, productNames []string) (string, error) {
	if !strings.Contains(content, packageSettingsDeclaration) {
		return appendFrameworkPackageSettingsBlock(content, productNames), nil
	}

	if !strings.Contains(content, packageProductTypesAnchor) {
		return insertFrameworkProductTypesSection(content, productNames)
	}

	return mergeFrameworkProductTypes(content, productNames)
}

func appendFrameworkPackageSettingsBlock(content string, productNames []string) string {
	var block strings.Builder
	block.WriteString("\n\n#if TUIST\n")
	block.WriteString("import ProjectDescription\n\n")
	block.WriteString("let packageSettings = PackageSettings(\n")
	block.WriteString("    productTypes: [\n")
	block.WriteString(renderFrameworkProductTypeEntries(productNames))
	block.WriteString("    ]\n")
	block.WriteString(")\n")
	block.WriteString("#endif\n")

	return strings.TrimRight(content, "\n") + block.String()
}

func insertFrameworkProductTypesSection(content string, productNames []string) (string, error) {
	anchor := strings.Index(content, packageSettingsDeclaration)
	if anchor == -1 {
		return "", fmt.Errorf("PackageSettings declaration not found")
	}

	insertPos := anchor + len(packageSettingsDeclaration)
	var section strings.Builder
	section.WriteString("\n")
	section.WriteString("    productTypes: [\n")
	section.WriteString(renderFrameworkProductTypeEntries(productNames))
	section.WriteString("    ],")

	return content[:insertPos] + section.String() + content[insertPos:], nil
}

func mergeFrameworkProductTypes(content string, productNames []string) (string, error) {
	missing := make([]string, 0, len(productNames))
	for _, productName := range productNames {
		if strings.Contains(content, frameworkProductTypeEntry(productName)) {
			continue
		}
		missing = append(missing, productName)
	}

	if len(missing) == 0 {
		return content, nil
	}

	anchor := strings.Index(content, packageProductTypesAnchor)
	if anchor == -1 {
		return "", fmt.Errorf("productTypes section not found")
	}

	arrayStart := anchor + len(packageProductTypesAnchor)
	arrayEndRelative := strings.Index(content[arrayStart:], "]")
	if arrayEndRelative == -1 {
		return "", fmt.Errorf("productTypes section is not closed")
	}

	arrayEnd := arrayStart + arrayEndRelative
	insertPos := lineStartOffset(content, arrayEnd)

	return content[:insertPos] + renderFrameworkProductTypeEntries(missing) + content[insertPos:], nil
}

func removeFrameworkProductTypesContent(content string, productNames []string) (string, error) {
	if !strings.Contains(content, packageSettingsDeclaration) || !strings.Contains(content, packageProductTypesAnchor) {
		return content, nil
	}

	anchor := strings.Index(content, packageProductTypesAnchor)
	if anchor == -1 {
		return "", fmt.Errorf("productTypes section not found")
	}

	arrayStart := anchor + len(packageProductTypesAnchor)
	arrayEndRelative := strings.Index(content[arrayStart:], "]")
	if arrayEndRelative == -1 {
		return "", fmt.Errorf("productTypes section is not closed")
	}

	arrayEnd := arrayStart + arrayEndRelative
	arrayContent := content[arrayStart:arrayEnd]
	lines := strings.SplitAfter(arrayContent, "\n")
	removeEntries := make(map[string]struct{}, len(productNames))
	for _, productName := range productNames {
		removeEntries[frameworkProductTypeEntry(productName)] = struct{}{}
	}

	filtered := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
		trimmed = strings.TrimSuffix(trimmed, ",")
		if _, shouldRemove := removeEntries[trimmed]; shouldRemove {
			changed = true
			continue
		}
		filtered = append(filtered, line)
	}

	if !changed {
		return content, nil
	}

	return content[:arrayStart] + strings.Join(filtered, "") + content[arrayEnd:], nil
}

func renderFrameworkProductTypeEntries(productNames []string) string {
	var builder strings.Builder
	for _, productName := range productNames {
		builder.WriteString("        ")
		builder.WriteString(frameworkProductTypeEntry(productName))
		builder.WriteString(",\n")
	}
	return builder.String()
}

func frameworkProductTypeEntry(productName string) string {
	return fmt.Sprintf(`"%s": .framework`, productName)
}

func normalizeFrameworkProductNames(productNames []string) []string {
	seen := make(map[string]struct{}, len(productNames))
	normalized := make([]string, 0, len(productNames))

	for _, productName := range productNames {
		trimmed := strings.TrimSpace(productName)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	sort.Strings(normalized)
	return normalized
}

func lineStartOffset(content string, pos int) int {
	if pos <= 0 {
		return 0
	}
	start := strings.LastIndex(content[:pos], "\n")
	if start == -1 {
		return 0
	}
	return start + 1
}
