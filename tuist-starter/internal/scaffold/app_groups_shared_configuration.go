package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templatepkg "github.com/relux-works/ios-app-manager/internal/template"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

var swiftIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func appGroupSharedConfigurationModuleName(cfg config.ProjectConfig) string {
	moduleName := strings.TrimSpace(cfg.SharedConfig.ModuleName)
	if moduleName == "" {
		return config.DefaultSharedConfigModuleName
	}
	return moduleName
}

func legacyAppGroupSharedConfigurationModuleName(appName string) string {
	return appGroupSharedConfigurationTypePrefix(appName) + "SharedConfiguration"
}

func appGroupSharedConfigurationPackagePath(root string, cfg config.ProjectConfig) string {
	return filepath.Join(root, normalizeModulesPath(cfg.ModulesPath), appGroupSharedConfigurationModuleName(cfg))
}

func appGroupSharedConfigurationPackageSwiftPath(root string, cfg config.ProjectConfig) string {
	return filepath.Join(appGroupSharedConfigurationPackagePath(root, cfg), "Package.swift")
}

func appGroupSharedConfigurationSourcePath(root string, cfg config.ProjectConfig) string {
	moduleName := appGroupSharedConfigurationModuleName(cfg)
	return filepath.Join(appGroupSharedConfigurationPackagePath(root, cfg), "Sources", moduleName+".swift")
}

func appGroupSharedConfigurationTypePrefix(appName string) string {
	trimmed := strings.TrimSpace(appName)
	if swiftIdentifierPattern.MatchString(trimmed) {
		return trimmed
	}

	parts := swiftIdentifierParts(trimmed)
	if len(parts) == 0 {
		return "App"
	}

	var b strings.Builder
	for _, part := range parts {
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}

	out := b.String()
	if out == "" {
		return "App"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "App" + out
	}
	return out
}

func swiftIdentifierParts(raw string) []string {
	parts := make([]string, 0)
	var current strings.Builder

	flush := func() {
		if current.Len() == 0 {
			return
		}
		parts = append(parts, current.String())
		current.Reset()
	}

	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'a' && r <= 'z':
			current.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			current.WriteRune(r)
		case r >= '0' && r <= '9':
			current.WriteRune(r)
		default:
			flush()
		}
	}
	flush()

	return parts
}

func GenerateAppGroupSharedConfigurationPackageSwift(cfg config.ProjectConfig) (string, error) {
	moduleName := appGroupSharedConfigurationModuleName(cfg)
	return tuistproj.GeneratePackageSwift(tuistproj.PackageGenerationInput{
		ModuleName: moduleName,
		Type:       tuistproj.PackageTypeInterface,
		Platform:   appGroupSharedConfigurationPackagePlatform(cfg),
		Config:     cfg,
	})
}

func appGroupSharedConfigurationPackagePlatform(cfg config.ProjectConfig) string {
	major := strings.TrimSpace(cfg.MinTarget)
	if dot := strings.Index(major, "."); dot >= 0 {
		major = major[:dot]
	}
	if major == "" {
		major = "17"
	}
	return "iOS(.v" + major + ")"
}

func GenerateAppGroupSharedConfigurationSwift(cfg config.ProjectConfig) string {
	appName := normalizeAppName(cfg.AppName)
	bundleID := strings.TrimSpace(cfg.BundleID)
	typePrefix := appGroupSharedConfigurationTypePrefix(appName)
	appGroups := normalizeAppGroups(cfg.AppGroups)

	type appGroup struct {
		PropertyName  string
		SlotCaseName  string
		DictionaryKey string
	}

	groups := make([]appGroup, 0, len(appGroups))
	for _, group := range appGroups {
		propertyName := templatepkg.AppGroupSwiftIdentifier(bundleID, group)
		groups = append(groups, appGroup{
			PropertyName:  propertyName,
			SlotCaseName:  propertyName,
			DictionaryKey: propertyName,
		})
	}

	var b strings.Builder
	b.WriteString("import Foundation\n\n")
	b.WriteString("public enum " + typePrefix + "InfoPlistKey: String, Sendable {\n")
	b.WriteString("    case appGroups = " + strconv.Quote(appGroupsInfoPlistKey) + "\n")
	b.WriteString("}\n\n")
	b.WriteString("public enum " + typePrefix + "AppGroupSlot: String, Sendable {\n")
	for _, group := range groups {
		b.WriteString("    case " + group.SlotCaseName + " = " + strconv.Quote(group.DictionaryKey) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("    public var infoPlistKey: " + typePrefix + "InfoPlistKey {\n")
	b.WriteString("        .appGroups\n")
	b.WriteString("    }\n\n")
	b.WriteString("    public var dictionaryKey: String {\n")
	b.WriteString("        rawValue\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("public struct " + typePrefix + "AppGroup: Equatable, Identifiable, Sendable {\n")
	b.WriteString("    public let slot: " + typePrefix + "AppGroupSlot\n")
	b.WriteString("    public let identifier: String\n\n")
	b.WriteString("    public var id: String { identifier }\n\n")
	b.WriteString("    public init(slot: " + typePrefix + "AppGroupSlot, identifier: String) {\n")
	b.WriteString("        self.slot = slot\n")
	b.WriteString("        self.identifier = identifier\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("public struct " + typePrefix + "AppGroups: Equatable, Sendable {\n")
	for _, group := range groups {
		b.WriteString("    public let " + group.PropertyName + ": String\n")
	}
	b.WriteString("\n")
	b.WriteString("    public init(\n")
	for index, group := range groups {
		suffix := ","
		if index == len(groups)-1 {
			suffix = ""
		}
		b.WriteString("        " + group.PropertyName + ": String" + suffix + "\n")
	}
	b.WriteString("    ) {\n")
	for _, group := range groups {
		b.WriteString("        self." + group.PropertyName + " = " + group.PropertyName + "\n")
	}
	b.WriteString("    }\n\n")
	b.WriteString("    public static func read(from bundle: Bundle = .main) throws -> Self {\n")
	b.WriteString("        try Self(\n")
	for index, group := range groups {
		suffix := ","
		if index == len(groups)-1 {
			suffix = ""
		}
		b.WriteString("            " + group.PropertyName + ": bundle." + lowerFirst(typePrefix) + "AppGroupString(for: ." + group.SlotCaseName + ")" + suffix + "\n")
	}
	b.WriteString("        )\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("public enum " + typePrefix + "SharedConfigError: Error, LocalizedError, Equatable {\n")
	b.WriteString("    case missingInfoPlistDictionary(key: " + typePrefix + "InfoPlistKey, bundleIdentifier: String?)\n")
	b.WriteString("    case missingInfoPlistValue(key: " + typePrefix + "InfoPlistKey, dictionaryKey: String, bundleIdentifier: String?)\n\n")
	b.WriteString("    public var errorDescription: String? {\n")
	b.WriteString("        switch self {\n")
	b.WriteString("        case let .missingInfoPlistDictionary(key, bundleIdentifier):\n")
	b.WriteString("            \"Missing Info.plist dictionary \\(key.rawValue) in bundle \\(bundleIdentifier ?? \"<unknown>\")\"\n")
	b.WriteString("        case let .missingInfoPlistValue(key, dictionaryKey, bundleIdentifier):\n")
	b.WriteString("            \"Missing Info.plist value \\(key.rawValue).\\(dictionaryKey) in bundle \\(bundleIdentifier ?? \"<unknown>\")\"\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("public extension Bundle {\n")
	b.WriteString("    func " + lowerFirst(typePrefix) + "AppGroupString(for slot: " + typePrefix + "AppGroupSlot) throws -> String {\n")
	b.WriteString("        let values = try " + lowerFirst(typePrefix) + "Dictionary(for: slot.infoPlistKey)\n")
	b.WriteString("        guard let value = values[slot.dictionaryKey],\n")
	b.WriteString("              !value.isEmpty else {\n")
	b.WriteString("            throw " + typePrefix + "SharedConfigError.missingInfoPlistValue(\n")
	b.WriteString("                key: slot.infoPlistKey,\n")
	b.WriteString("                dictionaryKey: slot.dictionaryKey,\n")
	b.WriteString("                bundleIdentifier: bundleIdentifier\n")
	b.WriteString("            )\n")
	b.WriteString("        }\n\n")
	b.WriteString("        return value\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("    func " + lowerFirst(typePrefix) + "Dictionary(for key: " + typePrefix + "InfoPlistKey) throws -> [String: String] {\n")
	b.WriteString("        if let value = object(forInfoDictionaryKey: key.rawValue) as? [String: String] {\n")
	b.WriteString("            return value\n")
	b.WriteString("        }\n\n")
	b.WriteString("        if let rawValue = object(forInfoDictionaryKey: key.rawValue) as? [String: Any] {\n")
	b.WriteString("            var value: [String: String] = [:]\n")
	b.WriteString("            for (dictionaryKey, dictionaryValue) in rawValue {\n")
	b.WriteString("                if let stringValue = dictionaryValue as? String {\n")
	b.WriteString("                    value[dictionaryKey] = stringValue\n")
	b.WriteString("                }\n")
	b.WriteString("            }\n")
	b.WriteString("            return value\n")
	b.WriteString("        }\n\n")
	b.WriteString("        throw " + typePrefix + "SharedConfigError.missingInfoPlistDictionary(\n")
	b.WriteString("            key: key,\n")
	b.WriteString("            bundleIdentifier: bundleIdentifier\n")
	b.WriteString("        )\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

func appGroupSwiftCaseSuffix(identifier string) string {
	switch identifier {
	case "sso", "sdk", "id", "url":
		return strings.ToUpper(identifier)
	default:
		if identifier == "" {
			return "Main"
		}
		return strings.ToUpper(identifier[:1]) + identifier[1:]
	}
}

func lowerFirst(identifier string) string {
	if identifier == "" {
		return ""
	}
	return strings.ToLower(identifier[:1]) + identifier[1:]
}

func syncAppGroupSharedConfigurationPackage(root string, cfg config.ProjectConfig) ([]string, error) {
	packageSwiftPath := appGroupSharedConfigurationPackageSwiftPath(root, cfg)
	sourcePath := appGroupSharedConfigurationSourcePath(root, cfg)

	packageSwift, err := GenerateAppGroupSharedConfigurationPackageSwift(cfg)
	if err != nil {
		return nil, err
	}

	updated := make([]string, 0, 2)
	changed, err := writeFileIfChanged(packageSwiftPath, packageSwift)
	if err != nil {
		return nil, fmt.Errorf("sync app-group shared configuration Package.swift: %w", err)
	}
	if changed {
		updated = append(updated, packageSwiftPath)
	}

	changed, err = writeFileIfChanged(sourcePath, GenerateAppGroupSharedConfigurationSwift(cfg))
	if err != nil {
		return nil, fmt.Errorf("sync app-group shared configuration source: %w", err)
	}
	if changed {
		updated = append(updated, sourcePath)
	}

	return updated, nil
}

func cleanupLegacyAppGroupSharedConfigurationPackage(root string, cfg config.ProjectConfig) ([]string, error) {
	appName := normalizeAppName(cfg.AppName)
	legacyModuleName := legacyAppGroupSharedConfigurationModuleName(appName)
	if legacyModuleName == appGroupSharedConfigurationModuleName(cfg) {
		return nil, nil
	}

	candidates := []string{
		filepath.Join(root, "Packages", legacyModuleName),
		filepath.Join(root, normalizeModulesPath(cfg.ModulesPath), legacyModuleName),
	}
	updated := make([]string, 0, len(candidates))
	for _, legacyPackagePath := range appendUniqueStrings(nil, candidates...) {
		if _, err := os.Stat(legacyPackagePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat legacy app-group shared configuration package: %w", err)
		}
		if err := os.RemoveAll(legacyPackagePath); err != nil {
			return nil, fmt.Errorf("remove legacy app-group shared configuration package: %w", err)
		}
		updated = append(updated, legacyPackagePath)
	}

	return updated, nil
}

func syncRootPackageSharedConfigurationDependency(root string, cfg config.ProjectConfig) (bool, error) {
	path := filepath.Join(root, "Package.swift")

	before, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read root Package.swift: %w", err)
	}

	updated, changed, err := syncRootPackageSharedConfigurationDependencyContent(string(before), cfg)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write root Package.swift: %w", err)
	}

	return true, nil
}

func cleanupRootPackageLegacySharedConfigurationDependency(root string, cfg config.ProjectConfig) (bool, error) {
	appName := normalizeAppName(cfg.AppName)
	legacyModuleName := legacyAppGroupSharedConfigurationModuleName(appName)
	if legacyModuleName == appGroupSharedConfigurationModuleName(cfg) {
		return false, nil
	}

	path := filepath.Join(root, "Package.swift")
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read root Package.swift: %w", err)
	}

	updated := string(payload)
	for _, legacyPath := range legacySharedConfigurationPackageRefPaths(cfg, legacyModuleName) {
		updated = removeLineContaining(updated, fmt.Sprintf(`.package(path: "%s")`, legacyPath))
	}
	updated, err = tuistproj.RemoveFrameworkProductTypesInContent(updated, legacyModuleName)
	if err != nil {
		return false, fmt.Errorf("remove legacy shared configuration product type: %w", err)
	}
	if updated == string(payload) {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write root Package.swift: %w", err)
	}

	return true, nil
}

func syncRootPackageSharedConfigurationDependencyContent(content string, cfg config.ProjectConfig) (string, bool, error) {
	moduleName := appGroupSharedConfigurationModuleName(cfg)
	manifest, err := tuistproj.ParseManifest(content)
	if err != nil {
		return "", false, fmt.Errorf("parse root Package.swift: %w", err)
	}

	hasDependency := false
	for _, dependency := range manifest.Dependencies {
		if dependency.Name == moduleName {
			hasDependency = true
			break
		}
	}

	updated := content
	if !hasDependency {
		refPath := filepath.ToSlash(filepath.Join(normalizeModulesPath(cfg.ModulesPath), moduleName))
		updated, err = tuistproj.ApplyManifestEdits(updated, tuistproj.ManifestEdit{
			Type:    tuistproj.AddDependency,
			Name:    moduleName,
			Content: fmt.Sprintf(`.package(path: "%s")`, refPath),
		})
		if err != nil {
			return "", false, fmt.Errorf("add shared configuration package to root Package.swift: %w", err)
		}
	}

	updated, err = tuistproj.EnsureFrameworkProductTypesInContent(updated, moduleName)
	if err != nil {
		return "", false, fmt.Errorf("force shared configuration package product type: %w", err)
	}

	return updated, updated != content, nil
}

func legacySharedConfigurationPackageRefPaths(cfg config.ProjectConfig, legacyModuleName string) []string {
	paths := []string{
		filepath.ToSlash(filepath.Join("Packages", legacyModuleName)),
		"./" + filepath.ToSlash(filepath.Join("Packages", legacyModuleName)),
		filepath.ToSlash(filepath.Join(normalizeModulesPath(cfg.ModulesPath), legacyModuleName)),
		"./" + filepath.ToSlash(filepath.Join(normalizeModulesPath(cfg.ModulesPath), legacyModuleName)),
	}
	return appendUniqueStrings(nil, paths...)
}

func syncProjectManifestSharedConfigurationDependency(path string, moduleName string) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}

	updated, changed, err := syncProjectManifestExternalDependencyContent(string(payload), moduleName)
	if err != nil {
		return false, fmt.Errorf("sync Project.swift shared configuration dependency: %w", err)
	}
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Project.swift: %w", err)
	}

	return true, nil
}

func cleanupProjectManifestLegacySharedConfigurationDependency(path string, cfg config.ProjectConfig) (bool, error) {
	appName := normalizeAppName(cfg.AppName)
	legacyModuleName := legacyAppGroupSharedConfigurationModuleName(appName)
	if legacyModuleName == appGroupSharedConfigurationModuleName(cfg) {
		return false, nil
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}

	updated := removeLineContaining(string(payload), fmt.Sprintf(`.external(name: "%s")`, legacyModuleName))
	if updated == string(payload) {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Project.swift: %w", err)
	}

	return true, nil
}

func removeLineContaining(content string, value string) string {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	if hasTrailingNewline && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, value) {
			continue
		}
		filtered = append(filtered, line)
	}

	updated := strings.Join(filtered, "\n")
	if hasTrailingNewline {
		updated += "\n"
	}
	return updated
}

func syncProjectManifestExternalDependencyContent(content string, moduleName string) (string, bool, error) {
	externalDependency := fmt.Sprintf(`.external(name: "%s")`, moduleName)
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	if hasTrailingNewline && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	changed := false
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if !strings.Contains(line, "dependencies:") {
			continue
		}
		if !strings.Contains(line, "[") {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "dependencies: []") {
			if strings.Contains(line, externalDependency) {
				continue
			}
			indent := leadingIndent(line)
			hasComma := strings.HasSuffix(trimmed, ",")
			closeLine := indent + "]"
			if hasComma {
				closeLine += ","
			}
			replacement := []string{
				indent + "dependencies: [",
				indent + "    " + externalDependency + ",",
				closeLine,
			}
			lines = append(lines[:index], append(replacement, lines[index+1:]...)...)
			index += len(replacement) - 1
			changed = true
			continue
		}

		closeIndex, ok := findArrayCloseLine(lines, index)
		if !ok {
			return "", false, fmt.Errorf("dependencies array opened on line %d has no closing bracket", index+1)
		}
		if sectionContains(lines[index:closeIndex+1], externalDependency) {
			index = closeIndex
			continue
		}

		insertIndent := dependencyInsertionIndent(lines, index, closeIndex)
		lines = append(lines[:closeIndex], append([]string{insertIndent + externalDependency + ","}, lines[closeIndex:]...)...)
		index = closeIndex + 1
		changed = true
	}

	if !changed {
		return content, false, nil
	}

	updated := strings.Join(lines, "\n")
	if hasTrailingNewline {
		updated += "\n"
	}
	return updated, true, nil
}

func findArrayCloseLine(lines []string, openLine int) (int, bool) {
	depth := 0
	started := false
	for index := openLine; index < len(lines); index++ {
		line := lines[index]
		start := 0
		if index == openLine {
			openIndex := indexOutsideStringAndComment(line, 0, '[')
			if openIndex < 0 {
				return 0, false
			}
			start = openIndex
		}

		for col := start; col < len(line); col++ {
			ch := line[col]
			if ch == '/' && col+1 < len(line) && line[col+1] == '/' {
				break
			}
			if ch == '"' {
				next := strings.Index(line[col+1:], `"`)
				if next < 0 {
					break
				}
				col += next + 1
				continue
			}
			switch ch {
			case '[':
				depth++
				started = true
			case ']':
				if depth > 0 {
					depth--
				}
				if started && depth == 0 {
					return index, true
				}
			}
		}
	}

	return 0, false
}

func indexOutsideStringAndComment(line string, start int, target byte) int {
	inString := false
	escaped := false
	for index := start; index < len(line); index++ {
		ch := line[index]
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
		if ch == '/' && index+1 < len(line) && line[index+1] == '/' {
			return -1
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == target {
			return index
		}
	}
	return -1
}

func sectionContains(lines []string, value string) bool {
	for _, line := range lines {
		if strings.Contains(line, value) {
			return true
		}
	}
	return false
}

func dependencyInsertionIndent(lines []string, openIndex int, closeIndex int) string {
	for index := openIndex + 1; index < closeIndex; index++ {
		if strings.TrimSpace(lines[index]) == "" {
			continue
		}
		return leadingIndent(lines[index])
	}
	return leadingIndent(lines[closeIndex]) + "    "
}
