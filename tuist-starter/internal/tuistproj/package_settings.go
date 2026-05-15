package tuistproj

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	packageSettingsDeclaration  = "let packageSettings = PackageSettings("
	packageProductTypesAnchor   = "productTypes: ["
	packageTargetSettingsAnchor = "targetSettings: ["
)

// TargetBuildSetting is one Tuist PackageSettings target build setting override.
type TargetBuildSetting struct {
	ProductName string
	Key         string
	Value       string
}

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

// EnsureTargetBuildSettings writes Tuist PackageSettings targetSettings entries
// for external package products.
func EnsureTargetBuildSettings(path string, settings ...TargetBuildSetting) error {
	normalized := normalizeTargetBuildSettings(settings)
	if len(normalized) == 0 {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Package.swift: %w", err)
	}

	updated, err := ensureTargetBuildSettingsContent(string(content), normalized)
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

func ensureFrameworkProductTypesContent(content string, productNames []string) (string, error) {
	if !strings.Contains(content, packageSettingsDeclaration) {
		return appendFrameworkPackageSettingsBlock(content, productNames), nil
	}

	if !strings.Contains(content, packageProductTypesAnchor) {
		return insertFrameworkProductTypesSection(content, productNames)
	}

	return mergeFrameworkProductTypes(content, productNames)
}

func ensureTargetBuildSettingsContent(content string, settings []TargetBuildSetting) (string, error) {
	if !strings.Contains(content, packageSettingsDeclaration) {
		return appendTargetSettingsPackageSettingsBlock(content, settings), nil
	}

	if !strings.Contains(content, packageTargetSettingsAnchor) {
		return insertTargetSettingsSection(content, settings)
	}

	return mergeTargetBuildSettings(content, settings)
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

func appendTargetSettingsPackageSettingsBlock(content string, settings []TargetBuildSetting) string {
	var block strings.Builder
	block.WriteString("\n\n#if TUIST\n")
	block.WriteString("import ProjectDescription\n\n")
	block.WriteString("let packageSettings = PackageSettings(\n")
	block.WriteString("    targetSettings: [\n")
	block.WriteString(renderTargetSettingsEntries(settings))
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

func insertTargetSettingsSection(content string, settings []TargetBuildSetting) (string, error) {
	anchor := strings.Index(content, packageSettingsDeclaration)
	if anchor == -1 {
		return "", fmt.Errorf("PackageSettings declaration not found")
	}

	insertPos := anchor + len(packageSettingsDeclaration)
	var section strings.Builder
	section.WriteString("\n")
	section.WriteString("    targetSettings: [\n")
	section.WriteString(renderTargetSettingsEntries(settings))
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

func mergeTargetBuildSettings(content string, settings []TargetBuildSetting) (string, error) {
	missing := make([]TargetBuildSetting, 0, len(settings))
	for _, setting := range settings {
		if strings.Contains(content, targetBuildSettingEntry(setting)) {
			continue
		}
		missing = append(missing, setting)
	}

	if len(missing) == 0 {
		return content, nil
	}

	anchor := strings.Index(content, packageTargetSettingsAnchor)
	if anchor == -1 {
		return "", fmt.Errorf("targetSettings section not found")
	}

	arrayStart := anchor + len(packageTargetSettingsAnchor)
	arrayEnd, err := matchingSquareBracket(content, arrayStart-1)
	if err != nil {
		return "", fmt.Errorf("targetSettings section is not closed")
	}

	insertPos := lineStartOffset(content, arrayEnd)
	return content[:insertPos] + renderTargetSettingsEntries(missing) + content[insertPos:], nil
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

func renderTargetSettingsEntries(settings []TargetBuildSetting) string {
	grouped := make(map[string][]TargetBuildSetting)
	productNames := make([]string, 0)
	for _, setting := range settings {
		if _, exists := grouped[setting.ProductName]; !exists {
			productNames = append(productNames, setting.ProductName)
		}
		grouped[setting.ProductName] = append(grouped[setting.ProductName], setting)
	}
	sort.Strings(productNames)

	var builder strings.Builder
	for _, productName := range productNames {
		productSettings := grouped[productName]
		sort.Slice(productSettings, func(i, j int) bool {
			return productSettings[i].Key < productSettings[j].Key
		})

		builder.WriteString("        ")
		builder.WriteString(fmt.Sprintf(`"%s": .settings(base: [`, productName))
		builder.WriteString("\n")
		for _, setting := range productSettings {
			builder.WriteString("            ")
			builder.WriteString(targetBuildSettingEntry(setting))
			builder.WriteString(",\n")
		}
		builder.WriteString("        ]),\n")
	}
	return builder.String()
}

func frameworkProductTypeEntry(productName string) string {
	return fmt.Sprintf(`"%s": .framework`, productName)
}

func targetBuildSettingEntry(setting TargetBuildSetting) string {
	return fmt.Sprintf(`"%s": "%s"`, setting.Key, setting.Value)
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

func normalizeTargetBuildSettings(settings []TargetBuildSetting) []TargetBuildSetting {
	seen := make(map[string]struct{}, len(settings))
	normalized := make([]TargetBuildSetting, 0, len(settings))

	for _, setting := range settings {
		productName := strings.TrimSpace(setting.ProductName)
		key := strings.TrimSpace(setting.Key)
		value := strings.TrimSpace(setting.Value)
		if productName == "" || key == "" || value == "" {
			continue
		}

		identity := productName + "\x00" + key + "\x00" + value
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		normalized = append(normalized, TargetBuildSetting{
			ProductName: productName,
			Key:         key,
			Value:       value,
		})
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].ProductName != normalized[j].ProductName {
			return normalized[i].ProductName < normalized[j].ProductName
		}
		if normalized[i].Key != normalized[j].Key {
			return normalized[i].Key < normalized[j].Key
		}
		return normalized[i].Value < normalized[j].Value
	})
	return normalized
}

func matchingSquareBracket(content string, openBracket int) (int, error) {
	if openBracket < 0 || openBracket >= len(content) || content[openBracket] != '[' {
		return -1, fmt.Errorf("opening bracket not found")
	}

	depth := 0
	for index := openBracket; index < len(content); index++ {
		switch content[index] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return index, nil
			}
		}
	}

	return -1, fmt.Errorf("closing bracket not found")
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
