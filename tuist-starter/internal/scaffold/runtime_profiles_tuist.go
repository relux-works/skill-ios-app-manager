package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	runtimeProfileConfigurationsBegin  = "// ios-app-manager:runtime-profile-configurations:begin"
	runtimeProfileConfigurationsEnd    = "// ios-app-manager:runtime-profile-configurations:end"
	runtimeProfileProjectSettingsBegin = "// ios-app-manager:runtime-profile-project-settings:begin"
	runtimeProfileProjectSettingsEnd   = "// ios-app-manager:runtime-profile-project-settings:end"
	runtimeProfileSchemesBegin         = "// ios-app-manager:runtime-profile-schemes:begin"
	runtimeProfileSchemesEnd           = "// ios-app-manager:runtime-profile-schemes:end"
	runtimeProfileHelpersImportMarker  = "// ios-app-manager:runtime-profiles"
	runtimeProfilePackageConfigMarker  = "// ios-app-manager:runtime-profile-package-configurations"
)

func init() {
	RegisterRuntimeProfilePlugin(&RuntimeProfilePlugin{
		Name:         "tuist-project",
		Short:        "Generate typed Tuist configurations and schemes",
		Dependencies: []string{"runtime-descriptors"},
		Sync:         syncRuntimeProfileTuistProject,
	})
}

func syncRuntimeProfileTuistProject(input RuntimeProfileInput) (RuntimeProfilePluginResult, error) {
	helperPath := runtimeProfilesProjectDescriptionPath(input.ProjectRoot)
	rootPackagePath := filepath.Join(input.ProjectRoot, "Package.swift")
	manifestPaths, err := discoverScaffoldManifestPaths(input.ProjectRoot)
	if err != nil {
		return RuntimeProfilePluginResult{}, err
	}
	result := RuntimeProfilePluginResult{
		Name:    "tuist-project",
		Enabled: input.Config.HasRuntimeProfiles(),
		Scanned: append([]string{helperPath, rootPackagePath}, manifestPaths...),
	}

	if result.Enabled {
		updated, err := writeFileIfChanged(helperPath, GenerateRuntimeProfilesProjectDescriptionSwift(input.Config))
		if err != nil {
			return result, fmt.Errorf("write RuntimeProfiles ProjectDescription helper: %w", err)
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, helperPath)
		}
	} else {
		updated, err := removeFileIfExists(helperPath)
		if err != nil {
			return result, fmt.Errorf("remove RuntimeProfiles ProjectDescription helper: %w", err)
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, helperPath)
		}
	}

	rootManifest := filepath.Join(input.ProjectRoot, "Project.swift")
	for _, manifestPath := range manifestPaths {
		updated, err := syncRuntimeProfilesProjectManifest(
			manifestPath,
			input.Config,
			result.Enabled,
			manifestPath == rootManifest,
		)
		if err != nil {
			return result, fmt.Errorf("sync %s: %w", filepath.Base(manifestPath), err)
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}
	}

	updated, err := syncRuntimeProfilesRootPackage(rootPackagePath, input.Config, result.Enabled)
	if err != nil {
		return result, fmt.Errorf("sync root Package.swift: %w", err)
	}
	if updated {
		result.Updated = appendUniqueStrings(result.Updated, rootPackagePath)
	}

	if result.Enabled {
		result.Message = "generated typed Tuist configurations and schemes"
	} else {
		result.Message = "removed typed Tuist runtime-profile output"
	}
	return result, nil
}

func syncRuntimeProfilesRootPackage(path string, cfg config.ProjectConfig, enabled bool) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Package.swift: %w", err)
	}
	updated, err := syncRuntimeProfilePackageManifestContent(string(payload), cfg, enabled)
	if err != nil {
		return false, err
	}
	if updated == string(payload) {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Package.swift: %w", err)
	}
	return true, nil
}

func syncRuntimeProfilePackageManifestContent(content string, _ config.ProjectConfig, enabled bool) (string, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, runtimeProfileHelpersImportMarker) {
			continue
		}
		if !strings.Contains(line, runtimeProfilePackageConfigMarker) {
			filtered = append(filtered, line)
			continue
		}
		restored, keep, err := restoreRuntimeProfilePackageConfigLine(line)
		if err != nil {
			return "", err
		}
		if keep {
			filtered = append(filtered, restored)
		}
	}
	lines = filtered
	if !enabled {
		return joinSyncLines(lines, hasTrailingNewline), nil
	}

	tuistStart, tuistEnd := findTuistBlockLineRange(lines)
	if tuistStart < 0 || tuistEnd < 0 {
		return "", fmt.Errorf("Package.swift must contain a complete #if TUIST block")
	}
	if findLineContainingInRange(lines, tuistStart, tuistEnd, "import ProjectDescriptionHelpers") < 0 {
		importLine := findLineContainingInRange(lines, tuistStart, tuistEnd, "import ProjectDescription")
		if importLine < 0 {
			return "", fmt.Errorf("Package.swift TUIST block must import ProjectDescription")
		}
		lines = insertRuntimeProfileLines(lines, importLine+1, []string{"import ProjectDescriptionHelpers " + runtimeProfileHelpersImportMarker})
		tuistEnd++
	}

	packageSettingsLine := findLineContainingInRange(lines, tuistStart, tuistEnd, "let packageSettings = PackageSettings(")
	if packageSettingsLine < 0 {
		return "", fmt.Errorf("Package.swift TUIST block must declare PackageSettings")
	}
	packageSettingsEnd, err := findDelimitedBlockEnd(lines, packageSettingsLine, "(", ")")
	if err != nil {
		return "", fmt.Errorf("unterminated PackageSettings block")
	}
	baseSettingsLine := findLineContainingInRange(lines, packageSettingsLine+1, packageSettingsEnd, "baseSettings:")
	if baseSettingsLine < 0 {
		var targetSettingsBlock []string
		targetSettingsLine := findLineContainingInRange(lines, packageSettingsLine+1, packageSettingsEnd, "targetSettings:")
		if targetSettingsLine >= 0 {
			targetSettingsEnd, err := packageSettingsDictionaryArgumentEnd(lines, targetSettingsLine)
			if err != nil {
				return "", fmt.Errorf("unterminated Package.swift targetSettings argument: %w", err)
			}
			targetSettingsBlock = append([]string(nil), lines[targetSettingsLine:targetSettingsEnd+1]...)
			lines = append(append([]string(nil), lines[:targetSettingsLine]...), lines[targetSettingsEnd+1:]...)
		}

		packageSettingsEnd, err = findDelimitedBlockEnd(lines, packageSettingsLine, "(", ")")
		if err != nil {
			return "", fmt.Errorf("unterminated PackageSettings block")
		}
		insertAt, err := runtimeProfileBaseSettingsInsertionIndex(lines, packageSettingsLine, packageSettingsEnd)
		if err != nil {
			return "", err
		}
		managedLine := "    baseSettings: .settings(configurations: RuntimeProfilesProjectDescription.configurations), " +
			runtimeProfilePackageConfigMarker + " inserted"
		lines = insertRuntimeProfileLines(lines, insertAt, []string{managedLine})

		if len(targetSettingsBlock) > 0 {
			packageSettingsEnd, err = findDelimitedBlockEnd(lines, packageSettingsLine, "(", ")")
			if err != nil {
				return "", fmt.Errorf("unterminated PackageSettings block")
			}
			targetSettingsInsertAt := packageSettingsEnd
			if projectOptionsLine := findLineContainingInRange(lines, packageSettingsLine+1, packageSettingsEnd, "projectOptions:"); projectOptionsLine >= 0 {
				targetSettingsInsertAt = projectOptionsLine
			}
			lines = insertRuntimeProfileLines(lines, targetSettingsInsertAt, targetSettingsBlock)
		}
		return joinSyncLines(lines, hasTrailingNewline), nil
	}

	line, originalConfigurations, err := replaceRuntimeProfilePackageConfigurations(lines[baseSettingsLine])
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, " \t") + " " + runtimeProfilePackageConfigMarker +
		" original=" + strconv.Quote(originalConfigurations)
	lines[baseSettingsLine] = line
	return joinSyncLines(lines, hasTrailingNewline), nil
}

func restoreRuntimeProfilePackageConfigLine(line string) (string, bool, error) {
	markerIndex := strings.Index(line, runtimeProfilePackageConfigMarker)
	if markerIndex < 0 {
		return line, true, nil
	}
	code := strings.TrimRight(line[:markerIndex], " \t")
	metadata := strings.TrimSpace(line[markerIndex+len(runtimeProfilePackageConfigMarker):])
	if metadata == "inserted" {
		return "", false, nil
	}
	if strings.HasPrefix(metadata, "original=") {
		original, err := strconv.Unquote(strings.TrimPrefix(metadata, "original="))
		if err != nil {
			return "", false, fmt.Errorf("invalid Package.swift runtime-profile configurations marker: %w", err)
		}
		restored, err := restoreRuntimeProfilePackageConfigurations(code, original)
		if err != nil {
			return "", false, err
		}
		return strings.TrimRight(restored, " \t"), true, nil
	}

	// Backward compatibility for output produced before original argument
	// metadata was recorded.
	trimmed := strings.TrimSpace(code)
	insertedPrefix := "baseSettings: .settings(configurations: RuntimeProfilesProjectDescription.configurations)"
	if strings.HasPrefix(trimmed, insertedPrefix) {
		return "", false, nil
	}
	restored := strings.ReplaceAll(
		code,
		", configurations: RuntimeProfilesProjectDescription.configurations",
		"",
	)
	return strings.TrimRight(restored, " \t"), true, nil
}

func replaceRuntimeProfilePackageConfigurations(line string) (string, string, error) {
	argsStart, argsEnd, err := settingsCallArgumentBounds(line)
	if err != nil {
		return "", "", fmt.Errorf("Package.swift baseSettings must use a single-line .settings(...) call for runtime-profile convergence")
	}
	args := line[argsStart:argsEnd]
	updatedArgs, original, found, err := replaceSwiftNamedArgument(
		args,
		"configurations",
		"RuntimeProfilesProjectDescription.configurations",
	)
	if err != nil {
		return "", "", fmt.Errorf("replace Package.swift baseSettings configurations: %w", err)
	}
	if !found {
		separator := ""
		if strings.TrimSpace(args) != "" {
			separator = ", "
		}
		updatedArgs = args + separator + "configurations: RuntimeProfilesProjectDescription.configurations"
	}
	return line[:argsStart] + updatedArgs + line[argsEnd:], original, nil
}

func restoreRuntimeProfilePackageConfigurations(line string, original string) (string, error) {
	argsStart, argsEnd, err := settingsCallArgumentBounds(line)
	if err != nil {
		return "", fmt.Errorf("restore Package.swift baseSettings configurations: %w", err)
	}
	args := line[argsStart:argsEnd]
	if original != "" {
		updatedArgs, _, found, err := replaceSwiftNamedArgument(args, "configurations", original)
		if err != nil {
			return "", err
		}
		if !found {
			return "", fmt.Errorf("managed Package.swift baseSettings configurations argument not found")
		}
		return line[:argsStart] + updatedArgs + line[argsEnd:], nil
	}

	updatedArgs, found, err := removeSwiftNamedArgument(args, "configurations")
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("managed Package.swift baseSettings configurations argument not found")
	}
	return line[:argsStart] + updatedArgs + line[argsEnd:], nil
}

func settingsCallArgumentBounds(line string) (int, int, error) {
	settingsCall := strings.Index(line, ".settings(")
	if settingsCall < 0 {
		return 0, 0, fmt.Errorf(".settings call not found")
	}
	openIndex := settingsCall + len(".settings")
	closeIndex := matchingSwiftDelimiterInLine(line, openIndex, '(', ')')
	if closeIndex < 0 {
		return 0, 0, fmt.Errorf("unterminated .settings call")
	}
	return openIndex + 1, closeIndex, nil
}

type swiftArgumentRange struct {
	start int
	end   int
}

func replaceSwiftNamedArgument(args string, name string, replacement string) (string, string, bool, error) {
	ranges, err := splitTopLevelSwiftArguments(args)
	if err != nil {
		return "", "", false, err
	}
	for _, argumentRange := range ranges {
		segment := args[argumentRange.start:argumentRange.end]
		leading := len(segment) - len(strings.TrimLeft(segment, " \t"))
		trimmed := strings.TrimSpace(segment)
		prefix := name + ":"
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		valueOffset := leading + len(prefix)
		for valueOffset < len(segment) && (segment[valueOffset] == ' ' || segment[valueOffset] == '\t') {
			valueOffset++
		}
		valueEnd := len(segment)
		for valueEnd > valueOffset && (segment[valueEnd-1] == ' ' || segment[valueEnd-1] == '\t') {
			valueEnd--
		}
		original := segment[valueOffset:valueEnd]
		updatedSegment := segment[:valueOffset] + replacement + segment[valueEnd:]
		return args[:argumentRange.start] + updatedSegment + args[argumentRange.end:], original, true, nil
	}
	return args, "", false, nil
}

func removeSwiftNamedArgument(args string, name string) (string, bool, error) {
	ranges, err := splitTopLevelSwiftArguments(args)
	if err != nil {
		return "", false, err
	}
	for index, argumentRange := range ranges {
		segment := strings.TrimSpace(args[argumentRange.start:argumentRange.end])
		if !strings.HasPrefix(segment, name+":") {
			continue
		}
		switch {
		case len(ranges) == 1:
			return "", true, nil
		case index > 0:
			removeStart := ranges[index-1].end
			return strings.TrimRight(args[:removeStart], " \t") + args[argumentRange.end:], true, nil
		default:
			removeEnd := ranges[index+1].start
			return strings.TrimLeft(args[removeEnd:], " \t"), true, nil
		}
	}
	return args, false, nil
}

func splitTopLevelSwiftArguments(args string) ([]swiftArgumentRange, error) {
	ranges := make([]swiftArgumentRange, 0, 4)
	start := 0
	parentheses := 0
	brackets := 0
	braces := 0
	inString := false
	escaped := false
	for index := 0; index < len(args); index++ {
		ch := args[index]
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
		if ch == '"' {
			inString = true
			continue
		}
		switch ch {
		case '(':
			parentheses++
		case ')':
			parentheses--
		case '[':
			brackets++
		case ']':
			brackets--
		case '{':
			braces++
		case '}':
			braces--
		case ',':
			if parentheses == 0 && brackets == 0 && braces == 0 {
				ranges = append(ranges, swiftArgumentRange{start: start, end: index})
				start = index + 1
			}
		}
		if parentheses < 0 || brackets < 0 || braces < 0 {
			return nil, fmt.Errorf("unbalanced Swift argument delimiters")
		}
	}
	if inString || parentheses != 0 || brackets != 0 || braces != 0 {
		return nil, fmt.Errorf("unterminated Swift argument")
	}
	if strings.TrimSpace(args[start:]) != "" || len(ranges) > 0 {
		ranges = append(ranges, swiftArgumentRange{start: start, end: len(args)})
	}
	return ranges, nil
}

func matchingSwiftDelimiterInLine(line string, openingIndex int, opening byte, closing byte) int {
	if openingIndex < 0 || openingIndex >= len(line) || line[openingIndex] != opening {
		return -1
	}
	depth := 0
	inString := false
	escaped := false
	for index := openingIndex; index < len(line); index++ {
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
		if ch == opening {
			depth++
		}
		if ch == closing {
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func runtimeProfileBaseSettingsInsertionIndex(lines []string, packageSettingsLine, packageSettingsEnd int) (int, error) {
	insertAt := packageSettingsLine + 1
	for _, argument := range []string{"productTypes:", "productDestinations:"} {
		argumentLine := findLineContainingInRange(lines, packageSettingsLine+1, packageSettingsEnd, argument)
		if argumentLine < 0 {
			continue
		}
		argumentEnd, err := packageSettingsDictionaryArgumentEnd(lines, argumentLine)
		if err != nil {
			return 0, fmt.Errorf("unterminated Package.swift %s argument: %w", strings.TrimSuffix(argument, ":"), err)
		}
		if argumentEnd+1 > insertAt {
			insertAt = argumentEnd + 1
		}
	}

	if projectOptionsLine := findLineContainingInRange(lines, packageSettingsLine+1, packageSettingsEnd, "projectOptions:"); projectOptionsLine >= 0 && insertAt > projectOptionsLine {
		return 0, fmt.Errorf("Package.swift PackageSettings arguments are not in Tuist initializer order")
	}
	return insertAt, nil
}

func packageSettingsDictionaryArgumentEnd(lines []string, argumentLine int) (int, error) {
	if argumentLine < 0 || argumentLine >= len(lines) {
		return 0, fmt.Errorf("argument line is out of range")
	}
	if !strings.Contains(lines[argumentLine], "[") {
		return argumentLine, nil
	}
	if strings.Count(lines[argumentLine], "[") == strings.Count(lines[argumentLine], "]") {
		return argumentLine, nil
	}
	return findDelimitedBlockEnd(lines, argumentLine, "[", "]")
}

func runtimeProfilesProjectDescriptionPath(root string) string {
	return filepath.Join(root, "Tuist", "ProjectDescriptionHelpers", "RuntimeProfiles.swift")
}

func GenerateRuntimeProfilesProjectDescriptionSwift(cfg config.ProjectConfig) string {
	if !cfg.HasRuntimeProfiles() {
		return ""
	}

	var b strings.Builder
	b.WriteString(generatedRuntimeProfilesHeader + "\n")
	b.WriteString("import ProjectDescription\n\n")
	b.WriteString("public enum RuntimeDistributionProfile: String, CaseIterable {\n")
	for _, profile := range cfg.OrderedDistributionProfiles() {
		b.WriteString("    case " + swiftRuntimeEnumCase(string(profile)) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("    public var configurationName: ConfigurationName {\n")
	b.WriteString("        switch self {\n")
	for _, profile := range cfg.OrderedDistributionProfiles() {
		descriptor := cfg.RuntimeProfiles.DistributionProfiles[profile]
		b.WriteString("        case ." + swiftRuntimeEnumCase(string(profile)) + ": .configuration(" + strconv.Quote(descriptor.BuildConfiguration) + ")\n")
	}
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("public enum RuntimeProfilesProjectDescription {\n")
	b.WriteString("    public static let configurations: [Configuration] = [\n")
	for _, profile := range cfg.OrderedDistributionProfiles() {
		descriptor := cfg.RuntimeProfiles.DistributionProfiles[profile]
		b.WriteString("        ." + string(descriptor.BuildKind) + "(\n")
		b.WriteString("            name: " + strconv.Quote(descriptor.BuildConfiguration) + ",\n")
		b.WriteString("            settings: [\"DISTRIBUTION_PROFILE\": " + strconv.Quote(string(profile)) + "]\n")
		b.WriteString("        ),\n")
	}
	b.WriteString("    ]\n\n")
	b.WriteString("    public static func schemes(appName: String) -> [Scheme] {\n")
	b.WriteString("        RuntimeDistributionProfile.allCases.map { scheme(for: $0, appName: appName) }\n")
	b.WriteString("    }\n\n")
	b.WriteString("    public static func scheme(for profile: RuntimeDistributionProfile, appName: String) -> Scheme {\n")
	b.WriteString("        let configuration = profile.configurationName\n")
	b.WriteString("        return .scheme(\n")
	b.WriteString("            name: \"\\(appName)-\\(configuration.rawValue)\",\n")
	b.WriteString("            shared: true,\n")
	b.WriteString("            buildAction: .buildAction(targets: [.target(appName)]),\n")
	b.WriteString("            testAction: profile == .tests\n")
	b.WriteString("                ? .targets([], configuration: configuration, expandVariableFromTarget: .target(appName))\n")
	b.WriteString("                : nil,\n")
	b.WriteString("            runAction: .runAction(configuration: configuration, executable: .target(appName)),\n")
	b.WriteString("            archiveAction: profile == .tests ? nil : .archiveAction(configuration: configuration),\n")
	b.WriteString("            profileAction: .profileAction(configuration: configuration, executable: .target(appName)),\n")
	b.WriteString("            analyzeAction: .analyzeAction(configuration: configuration)\n")
	b.WriteString("        )\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")
	return b.String()
}

func syncRuntimeProfilesProjectManifest(path string, cfg config.ProjectConfig, enabled bool, syncSchemes bool) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}
	content := string(payload)
	updated := content

	updated, err = syncRuntimeProfileHelperImportContent(updated, enabled)
	if err != nil {
		return false, err
	}
	updated, err = syncRuntimeProfileConfigurationsContent(updated, cfg, enabled)
	if err != nil {
		return false, err
	}
	updated, err = syncRuntimeProfileProjectSettingsContent(updated, enabled)
	if err != nil {
		return false, err
	}
	if syncSchemes {
		updated, err = syncRuntimeProfileSchemesContent(updated, cfg, enabled)
		if err != nil {
			return false, err
		}
	}

	if updated == content {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Project.swift: %w", err)
	}
	return true, nil
}

func syncRuntimeProfileHelperImportContent(content string, enabled bool) (string, error) {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, runtimeProfileHelpersImportMarker) {
			continue
		}
		filtered = append(filtered, line)
	}
	if !enabled || strings.Contains(strings.Join(filtered, "\n"), "import ProjectDescriptionHelpers") {
		return joinSyncLines(filtered, strings.HasSuffix(content, "\n")), nil
	}

	insertAt := findLineContaining(filtered, "import ProjectDescription")
	if insertAt < 0 {
		return "", fmt.Errorf("Project.swift import ProjectDescription not found")
	}
	filtered = insertSyncLine(filtered, insertAt+1, "import ProjectDescriptionHelpers "+runtimeProfileHelpersImportMarker)
	return joinSyncLines(filtered, strings.HasSuffix(content, "\n")), nil
}

func syncRuntimeProfileConfigurationsContent(content string, cfg config.ProjectConfig, enabled bool) (string, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	var hadManaged bool
	var err error
	lines, hadManaged, err = removeManagedRuntimeProfileBlock(lines, runtimeProfileConfigurationsBegin, runtimeProfileConfigurationsEnd)
	if err != nil {
		return "", err
	}

	projectLine := findLineContaining(lines, "let project = Project(")
	if projectLine < 0 {
		return "", fmt.Errorf("Project.swift project declaration not found")
	}

	if enabled {
		lines, err = removeLegacyConfigurationDeclaration(lines, projectLine)
		if err != nil {
			return "", err
		}
		projectLine = findLineContaining(lines, "let project = Project(")
		lines, projectLine = removeBlankLinesBefore(lines, projectLine)
		block := []string{
			runtimeProfileConfigurationsBegin,
			"let configurations: [Configuration] = RuntimeProfilesProjectDescription.configurations",
			runtimeProfileConfigurationsEnd,
			"",
		}
		lines = insertRuntimeProfileLines(lines, projectLine, block)
		return joinSyncLines(lines, hasTrailingNewline), nil
	}

	if !hadManaged {
		return content, nil
	}
	projectLine = findLineContaining(lines, "let project = Project(")
	lines, projectLine = removeBlankLinesBefore(lines, projectLine)
	block := renderLegacyConfigurations(cfg.Configurations)
	block = append(block, "")
	lines = insertRuntimeProfileLines(lines, projectLine, block)
	return joinSyncLines(lines, hasTrailingNewline), nil
}

func removeLegacyConfigurationDeclaration(lines []string, before int) ([]string, error) {
	for index := 0; index < before && index < len(lines); index++ {
		if !strings.Contains(lines[index], "let configurations") {
			continue
		}
		end := index
		initializerLine, initializerColumn, hasArrayInitializer := configurationArrayInitializer(lines, index)
		if hasArrayInitializer {
			closeLine, ok := findArrayCloseLineFrom(lines, initializerLine, initializerColumn)
			if !ok {
				return nil, fmt.Errorf("configurations array opened on line %d has no closing bracket", index+1)
			}
			end = closeLine
		}
		if end+1 < len(lines) && strings.TrimSpace(lines[end+1]) == "" {
			end++
		}
		return append(append([]string(nil), lines[:index]...), lines[end+1:]...), nil
	}
	return lines, nil
}

func configurationArrayInitializer(lines []string, declarationLine int) (int, int, bool) {
	line := lines[declarationLine]
	equals := indexOutsideStringAndComment(line, 0, '=')
	if equals < 0 {
		return 0, 0, false
	}
	for index := declarationLine; index < len(lines); index++ {
		start := 0
		if index == declarationLine {
			start = equals + 1
		}
		for column := start; column < len(lines[index]); column++ {
			switch lines[index][column] {
			case ' ', '\t':
				continue
			case '[':
				return index, column, true
			default:
				return 0, 0, false
			}
		}
	}
	return 0, 0, false
}

func findArrayCloseLineFrom(lines []string, openLine int, openColumn int) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for lineIndex := openLine; lineIndex < len(lines); lineIndex++ {
		start := 0
		if lineIndex == openLine {
			start = openColumn
		}
		line := lines[lineIndex]
		inString = false
		escaped = false
		for column := start; column < len(line); column++ {
			ch := line[column]
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
			if ch == '/' && column+1 < len(line) && line[column+1] == '/' {
				break
			}
			if ch == '"' {
				inString = true
				continue
			}
			switch ch {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					return lineIndex, true
				}
			}
		}
	}
	return 0, false
}

func renderLegacyConfigurations(configurations []string) []string {
	values := make([]string, 0, len(configurations))
	for _, raw := range configurations {
		if value := strings.TrimSpace(raw); value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		values = []string{"Debug", "Release"}
	}
	lines := []string{"let configurations: [Configuration] = ["}
	for _, value := range values {
		kind := "release"
		if strings.EqualFold(value, "Debug") {
			kind = "debug"
		}
		lines = append(lines, "    ."+kind+"(name: "+strconv.Quote(value)+"),")
	}
	return append(lines, "]")
}

func syncRuntimeProfileProjectSettingsContent(content string, enabled bool) (string, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	var err error
	lines, _, err = removeManagedRuntimeProfileBlock(lines, runtimeProfileProjectSettingsBegin, runtimeProfileProjectSettingsEnd)
	if err != nil {
		return "", err
	}
	if !enabled {
		return joinSyncLines(lines, hasTrailingNewline), nil
	}

	projectLine := findLineContaining(lines, "let project = Project(")
	if projectLine < 0 {
		return "", fmt.Errorf("Project.swift project declaration not found")
	}
	targetsLine := findLineContainingInRange(lines, projectLine, len(lines), "targets:")
	if targetsLine < 0 {
		return "", fmt.Errorf("Project.swift project targets declaration not found")
	}
	for index := projectLine; index < targetsLine; index++ {
		if strings.Contains(lines[index], "configurations: configurations") {
			return joinSyncLines(lines, hasTrailingNewline), nil
		}
	}
	for index := projectLine; index < targetsLine; index++ {
		if strings.Contains(lines[index], "settings:") {
			return "", fmt.Errorf("project settings exist without configurations: configurations; cannot add a second settings argument")
		}
	}
	indent := leadingIndent(lines[targetsLine])
	block := []string{
		indent + runtimeProfileProjectSettingsBegin,
		indent + "settings: .settings(configurations: configurations),",
		indent + runtimeProfileProjectSettingsEnd,
	}
	lines = insertRuntimeProfileLines(lines, targetsLine, block)
	return joinSyncLines(lines, hasTrailingNewline), nil
}

func syncRuntimeProfileSchemesContent(content string, cfg config.ProjectConfig, enabled bool) (string, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	var err error
	lines, _, err = removeManagedRuntimeProfileBlock(lines, runtimeProfileSchemesBegin, runtimeProfileSchemesEnd)
	if err != nil {
		return "", err
	}
	if !enabled {
		return joinSyncLines(lines, hasTrailingNewline), nil
	}

	projectLine := findLineContaining(lines, "let project = Project(")
	if projectLine < 0 {
		return "", fmt.Errorf("Project.swift project declaration not found")
	}
	schemesLine := findLineContainingInRange(lines, projectLine, len(lines), "schemes:")
	if schemesLine >= 0 {
		if !strings.Contains(lines[schemesLine], "[") {
			return "", fmt.Errorf("existing schemes declaration must use an array for runtime-profile convergence")
		}
		closeLine, ok := findArrayCloseLine(lines, schemesLine)
		if !ok {
			return "", fmt.Errorf("schemes array opened on line %d has no closing bracket", schemesLine+1)
		}
		lines, closeLine, err = removeLegacyAppSchemes(lines, schemesLine, closeLine, cfg.AppName)
		if err != nil {
			return "", err
		}
		ensureArrayLastItemComma(lines, schemesLine, closeLine)
		indent := leadingIndent(lines[closeLine]) + "    "
		block := []string{indent + runtimeProfileSchemesBegin}
		for _, profile := range cfg.OrderedDistributionProfiles() {
			block = append(block, indent+"RuntimeProfilesProjectDescription.scheme(for: ."+swiftRuntimeEnumCase(string(profile))+", appName: appName),")
		}
		block = append(block, indent+runtimeProfileSchemesEnd)
		lines = insertRuntimeProfileLines(lines, closeLine, block)
		return joinSyncLines(lines, hasTrailingNewline), nil
	}

	targetsLine := findLineContainingInRange(lines, projectLine, len(lines), "targets:")
	if targetsLine < 0 {
		return "", fmt.Errorf("Project.swift targets declaration not found")
	}
	targetsClose, ok := findArrayCloseLine(lines, targetsLine)
	if !ok {
		return "", fmt.Errorf("targets array opened on line %d has no closing bracket", targetsLine+1)
	}
	indent := leadingIndent(lines[targetsLine])
	block := []string{
		indent + runtimeProfileSchemesBegin,
		indent + "schemes: RuntimeProfilesProjectDescription.schemes(appName: appName),",
		indent + runtimeProfileSchemesEnd,
	}
	lines = insertRuntimeProfileLines(lines, targetsClose+1, block)
	return joinSyncLines(lines, hasTrailingNewline), nil
}

func removeLegacyAppSchemes(lines []string, schemesLine int, closeLine int, appName string) ([]string, int, error) {
	for index := schemesLine + 1; index < closeLine; {
		if !strings.HasPrefix(strings.TrimSpace(lines[index]), ".scheme(") {
			index++
			continue
		}

		end, err := findSwiftCallEnd(lines, index)
		if err != nil {
			return nil, 0, fmt.Errorf("legacy app scheme opened on line %d is unterminated", index+1)
		}
		if end >= closeLine {
			return nil, 0, fmt.Errorf("legacy app scheme opened on line %d crosses the schemes array boundary", index+1)
		}

		name, err := swiftSchemeNameArgument(strings.Join(lines[index:end+1], "\n"))
		if err != nil {
			return nil, 0, fmt.Errorf("parse scheme opened on line %d: %w", index+1, err)
		}
		if name != "appName" && name != strconv.Quote(strings.TrimSpace(appName)) {
			index = end + 1
			continue
		}

		removed := end - index + 1
		lines = append(append([]string(nil), lines[:index]...), lines[end+1:]...)
		closeLine -= removed
	}
	return lines, closeLine, nil
}

func findSwiftCallEnd(lines []string, startLine int) (int, error) {
	depth := 0
	started := false
	inString := false
	escaped := false
	for lineIndex := startLine; lineIndex < len(lines); lineIndex++ {
		inLineComment := false
		for column := 0; column < len(lines[lineIndex]); column++ {
			ch := lines[lineIndex][column]
			if inLineComment {
				break
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
			if ch == '/' && column+1 < len(lines[lineIndex]) && lines[lineIndex][column+1] == '/' {
				inLineComment = true
				continue
			}
			if ch == '"' {
				inString = true
				continue
			}
			switch ch {
			case '(':
				depth++
				started = true
			case ')':
				if started {
					depth--
					if depth == 0 {
						return lineIndex, nil
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("unterminated Swift call")
}

func swiftSchemeNameArgument(call string) (string, error) {
	prefixIndex := strings.Index(call, ".scheme(")
	if prefixIndex < 0 {
		return "", fmt.Errorf(".scheme call not found")
	}
	argsStart := prefixIndex + len(".scheme(")
	argsEnd := matchingSwiftCallClose(call, argsStart-1)
	if argsEnd < 0 {
		return "", fmt.Errorf("unterminated .scheme call")
	}
	ranges, err := splitTopLevelSwiftArguments(call[argsStart:argsEnd])
	if err != nil {
		return "", err
	}
	args := call[argsStart:argsEnd]
	for _, argumentRange := range ranges {
		argument := strings.TrimSpace(args[argumentRange.start:argumentRange.end])
		if !strings.HasPrefix(argument, "name:") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(argument, "name:")), nil
	}
	return "", nil
}

func matchingSwiftCallClose(content string, openingIndex int) int {
	depth := 0
	inString := false
	escaped := false
	inLineComment := false
	for index := openingIndex; index < len(content); index++ {
		ch := content[index]
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
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
		if ch == '/' && index+1 < len(content) && content[index+1] == '/' {
			inLineComment = true
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func removeManagedRuntimeProfileBlock(lines []string, begin string, end string) ([]string, bool, error) {
	beginIndex := findLineContaining(lines, begin)
	if beginIndex < 0 {
		return lines, false, nil
	}
	endIndex := findLineContainingInRange(lines, beginIndex+1, len(lines), end)
	if endIndex < 0 {
		return nil, false, fmt.Errorf("managed block %q has no closing marker", begin)
	}
	updated := append([]string(nil), lines[:beginIndex]...)
	updated = append(updated, lines[endIndex+1:]...)
	return updated, true, nil
}

func insertRuntimeProfileLines(lines []string, index int, inserted []string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(lines) {
		index = len(lines)
	}
	updated := make([]string, 0, len(lines)+len(inserted))
	updated = append(updated, lines[:index]...)
	updated = append(updated, inserted...)
	updated = append(updated, lines[index:]...)
	return updated
}

func removeBlankLinesBefore(lines []string, index int) ([]string, int) {
	start := index
	for start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start--
	}
	if start == index {
		return lines, index
	}
	updated := append([]string(nil), lines[:start]...)
	updated = append(updated, lines[index:]...)
	return updated, start
}
