package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

var packageIOSMinTargetPattern = regexp.MustCompile(`\.iOS\s*\(\s*(?:\.v\d+(?:_\d+)?|"\d+(?:\.\d+)?")\s*\)`)
var packageIOSPlatformReferencePattern = regexp.MustCompile(`\.iOS\b`)

type minTargetManifestPlan struct {
	path     string
	original []byte
	updated  []byte
}

type minTargetManifestSync func(content, minTarget string) (string, bool, error)

// SyncMinTarget synchronizes the configured iOS minimum target across generated
// Tuist targets, root Package.swift deployment overrides, and first-party local
// package platform declarations. It validates and stages every mutation before
// writing any manifest so unsupported or unwritable layouts cannot be partially
// migrated.
func SyncMinTarget(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ManifestSyncResult{}, fmt.Errorf("project root is required")
	}

	minTarget := strings.TrimSpace(cfg.MinTarget)
	if minTarget == "" {
		return ManifestSyncResult{}, fmt.Errorf("min target is required")
	}
	if _, _, err := parseMajorMinorVersion(minTarget); err != nil {
		return ManifestSyncResult{}, fmt.Errorf("invalid min target: %w", err)
	}

	manifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return ManifestSyncResult{}, err
	}
	if len(manifestPaths) == 0 {
		return ManifestSyncResult{}, fmt.Errorf("no scaffold Project.swift manifests found in %q; run init first", root)
	}

	rootPackagePath := filepath.Join(root, "Package.swift")
	rootPackageExists, err := optionalMinTargetManifest(rootPackagePath, "root Package.swift")
	if err != nil {
		return ManifestSyncResult{}, err
	}

	packageManifestPaths, err := discoverMinTargetPackageManifestPaths(root, cfg.ModulesPath)
	if err != nil {
		return ManifestSyncResult{}, err
	}
	if len(packageManifestPaths) > 0 && !rootPackageExists {
		return ManifestSyncResult{}, fmt.Errorf("root Package.swift %q is required when first-party package manifests are present", rootPackagePath)
	}

	manifestSyncs := make(map[string]minTargetManifestSync, len(manifestPaths)+len(packageManifestPaths)+1)
	for _, manifestPath := range manifestPaths {
		manifestSyncs[manifestPath] = syncMinTargetManifest
	}
	if rootPackageExists {
		manifestSyncs[rootPackagePath] = syncRootPackageMinTargetManifest
	}
	for _, manifestPath := range packageManifestPaths {
		manifestSyncs[manifestPath] = syncPackageMinTargetManifest
	}

	manifestPaths = make([]string, 0, len(manifestSyncs))
	for manifestPath := range manifestSyncs {
		manifestPaths = append(manifestPaths, manifestPath)
	}
	sort.Strings(manifestPaths)

	plans := make([]minTargetManifestPlan, 0, len(manifestPaths))
	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("read min target manifest %q: %w", manifestPath, err)
		}

		updated, changed, err := manifestSyncs[manifestPath](string(payload), minTarget)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync min target in %q: %w", manifestPath, err)
		}
		if changed {
			plans = append(plans, minTargetManifestPlan{
				path:     manifestPath,
				original: payload,
				updated:  []byte(updated),
			})
		}
	}

	for _, plan := range plans {
		if err := preflightMinTargetManifestWrite(plan.path); err != nil {
			return ManifestSyncResult{}, err
		}
	}

	if err := writeMinTargetManifestPlans(plans); err != nil {
		return ManifestSyncResult{}, err
	}

	updated := make([]string, 0, len(plans))
	for _, plan := range plans {
		updated = append(updated, plan.path)
	}

	return ManifestSyncResult{
		Scanned: manifestPaths,
		Updated: updated,
	}, nil
}

func optionalMinTargetManifest(path, label string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s %q: %w", label, path, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s %q must be a regular file", label, path)
	}
	return true, nil
}

func discoverMinTargetPackageManifestPaths(projectRoot, modulesPath string) ([]string, error) {
	configuredPath := strings.TrimSpace(modulesPath)
	if configuredPath == "" {
		configuredPath = defaultModulesDir
	}
	if filepath.IsAbs(configuredPath) {
		return nil, fmt.Errorf("modules path %q must be relative to project root for minimum-target synchronization", configuredPath)
	}

	configuredPath = filepath.Clean(filepath.FromSlash(configuredPath))
	if configuredPath == "." || configuredPath == ".." || strings.HasPrefix(configuredPath, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("modules path %q must stay within project root for minimum-target synchronization", modulesPath)
	}

	modulesRoot := filepath.Join(projectRoot, configuredPath)
	info, err := os.Stat(modulesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat first-party modules directory %q: %w", modulesRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("first-party modules path %q must be a directory", modulesRoot)
	}

	paths := make([]string, 0)
	err = filepath.WalkDir(modulesRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "Package.swift" {
				return fmt.Errorf("package manifest %q must be a regular file", path)
			}
			switch entry.Name() {
			case ".build", ".git", ".swiftpm":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if entry.Name() != "Package.swift" {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("package manifest %q must be a regular file", path)
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover first-party package manifests under %q: %w", modulesRoot, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func preflightMinTargetManifestWrite(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat min target manifest %q before write: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("min target manifest %q must be a regular writable file", path)
	}
	if info.Mode().Perm()&0o222 == 0 {
		return fmt.Errorf("min target manifest %q is not writable", path)
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("min target manifest %q is not writable: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close min target manifest %q after write preflight: %w", path, err)
	}
	return nil
}

func writeMinTargetManifestPlans(plans []minTargetManifestPlan) error {
	written := make([]minTargetManifestPlan, 0, len(plans))
	for _, plan := range plans {
		if err := os.WriteFile(plan.path, plan.updated, 0o644); err != nil {
			rollbackErr := rollbackMinTargetManifestPlans(written)
			if rollbackErr != nil {
				return fmt.Errorf("write min target manifest %q: %w; rollback failed: %v", plan.path, err, rollbackErr)
			}
			return fmt.Errorf("write min target manifest %q: %w", plan.path, err)
		}
		written = append(written, plan)
	}
	return nil
}

func rollbackMinTargetManifestPlans(plans []minTargetManifestPlan) error {
	var rollbackErrs []string
	for index := len(plans) - 1; index >= 0; index-- {
		plan := plans[index]
		if err := os.WriteFile(plan.path, plan.original, 0o644); err != nil {
			rollbackErrs = append(rollbackErrs, fmt.Sprintf("%q: %v", plan.path, err))
		}
	}
	if len(rollbackErrs) > 0 {
		return fmt.Errorf("%s", strings.Join(rollbackErrs, "; "))
	}
	return nil
}

func syncPackageMinTargetManifest(content, minTarget string) (string, bool, error) {
	major, minor, err := parseMajorMinorVersion(minTarget)
	if err != nil {
		return "", false, err
	}

	swiftPlatform := fmt.Sprintf(`.iOS(%q)`, fmt.Sprintf("%d.%d", major, minor))

	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	changed := false

	for index := 0; index < len(lines); {
		trimmed := strings.TrimSpace(lines[index])
		platformsIndex := strings.Index(trimmed, "platforms:")
		if strings.HasPrefix(trimmed, "//") || platformsIndex < 0 {
			index++
			continue
		}

		if !strings.Contains(trimmed[platformsIndex:], "[") {
			return "", false, fmt.Errorf("platforms declaration must use an array")
		}
		platformsEnd, err := findDelimitedBlockEnd(lines, index, "[", "]")
		if err != nil {
			return "", false, fmt.Errorf("unterminated platforms declaration: %w", err)
		}

		updatedBlock, blockChanged, err := syncPackagePlatformsBlock(lines[index:platformsEnd+1], swiftPlatform)
		if err != nil {
			return "", false, err
		}
		if blockChanged {
			copy(lines[index:platformsEnd+1], updatedBlock)
			changed = true
		}
		index = platformsEnd + 1
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func syncPackagePlatformsBlock(lines []string, swiftPlatform string) ([]string, bool, error) {
	updated := append([]string(nil), lines...)
	changed := false

	for index, line := range updated {
		code, comment := splitSwiftLineComment(line)
		if !strings.Contains(code, ".iOS") {
			continue
		}

		if packageIOSPlatformReferencePattern.MatchString(packageIOSMinTargetPattern.ReplaceAllString(code, "")) {
			return nil, false, fmt.Errorf("unsupported iOS platform declaration %q; expected .iOS(.v<major>[_<minor>]) or .iOS(\"<major>.<minor>\")", strings.TrimSpace(line))
		}

		replacement := packageIOSMinTargetPattern.ReplaceAllString(code, swiftPlatform)
		if replacement != code {
			updated[index] = replacement + comment
			changed = true
		}
	}

	return updated, changed, nil
}

func splitSwiftLineComment(line string) (code, comment string) {
	inString := false
	escaped := false
	for index := 0; index < len(line); index++ {
		switch line[index] {
		case '\\':
			if inString {
				escaped = !escaped
			}
		case '"':
			if !escaped {
				inString = !inString
			}
			escaped = false
		case '/':
			if !inString && index+1 < len(line) && line[index+1] == '/' {
				return line[:index], line[index:]
			}
		default:
			escaped = false
		}
	}
	return line, ""
}

func syncRootPackageMinTargetManifest(content, minTarget string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	changed := false

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || !strings.Contains(line, `"IPHONEOS_DEPLOYMENT_TARGET":`) {
			continue
		}

		replacement, err := rewriteRootPackageDeploymentTarget(line, minTarget)
		if err != nil {
			return "", false, err
		}
		if replacement != line {
			lines[index] = replacement
			changed = true
		}
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func rewriteRootPackageDeploymentTarget(line, minTarget string) (string, error) {
	const key = `"IPHONEOS_DEPLOYMENT_TARGET":`
	keyIndex := strings.Index(line, key)
	if keyIndex < 0 {
		return line, nil
	}

	valueStart := keyIndex + len(key)
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}
	if strings.HasPrefix(line[valueStart:], ".string(") {
		valueStart += len(".string(")
		for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
			valueStart++
		}
	}

	replacement, ok := replaceSwiftStringLiteralAt(line, valueStart, minTarget)
	if !ok {
		return "", fmt.Errorf("unsupported root Package.swift IPHONEOS_DEPLOYMENT_TARGET override %q; expected a string literal", strings.TrimSpace(line))
	}
	return replacement, nil
}

func replaceSwiftStringLiteralAt(line string, start int, value string) (string, bool) {
	if start < 0 || start >= len(line) || line[start] != '"' {
		return "", false
	}

	for index := start + 1; index < len(line); index++ {
		if line[index] == '\\' {
			index++
			continue
		}
		if line[index] == '"' {
			return line[:start] + fmt.Sprintf("%q", value) + line[index+1:], true
		}
	}
	return "", false
}

func parseMajorMinorVersion(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("version %q must use major.minor format", value)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse major version from %q: %w", value, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse minor version from %q: %w", value, err)
	}

	return major, minor, nil
}

func syncMinTargetManifest(content, minTarget string) (string, bool, error) {
	updated := content
	changed := false

	next, constantChanged, err := ensureMinTargetConstant(updated, minTarget)
	if err != nil {
		return "", false, err
	}
	updated = next
	changed = changed || constantChanged

	next, targetChanged, err := syncTuistTargetMinTargetMarkers(updated)
	if err != nil {
		return "", false, err
	}
	updated = next
	changed = changed || targetChanged

	return updated, changed, nil
}

func ensureMinTargetConstant(content, minTarget string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	insertAfter := -1
	insertBeforeProject := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "let minTarget = ") {
			replacement := leadingIndent(line) + fmt.Sprintf(`let minTarget = %q`, minTarget)
			if replacement == line {
				return content, false, nil
			}
			lines[index] = replacement
			return joinSyncLines(lines, hasTrailingNewline), true, nil
		}

		if strings.HasPrefix(trimmed, "let currentProjectVersion = ") {
			insertAfter = index
			continue
		}
		if insertAfter < 0 && strings.HasPrefix(trimmed, "let marketingVersion = ") {
			insertAfter = index
			continue
		}
		if insertBeforeProject < 0 && strings.HasPrefix(trimmed, "let project = Project(") {
			insertBeforeProject = index
		}
	}

	insertLine := fmt.Sprintf(`let minTarget = %q`, minTarget)
	switch {
	case insertAfter >= 0:
		lines = insertSyncLine(lines, insertAfter+1, insertLine)
	case insertBeforeProject >= 0:
		lines = insertSyncLine(lines, insertBeforeProject, insertLine)
	default:
		return "", false, fmt.Errorf("min target insertion anchor not found")
	}

	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func syncTuistTargetMinTargetMarkers(content string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	changed := false
	targetCount := 0

	for index := 0; index < len(lines); {
		if !isTuistTargetDeclaration(lines[index]) {
			index++
			continue
		}

		targetEnd, err := findDelimitedBlockEnd(lines, index, "(", ")")
		if err != nil {
			return "", false, fmt.Errorf("unterminated target declaration at line %d: %w", index+1, err)
		}

		updatedTarget, targetChanged, err := syncTuistTargetMinTargetBlock(lines[index : targetEnd+1])
		if err != nil {
			return "", false, fmt.Errorf("target declaration at line %d: %w", index+1, err)
		}
		if targetChanged {
			next := make([]string, 0, len(lines)-((targetEnd+1)-index)+len(updatedTarget))
			next = append(next, lines[:index]...)
			next = append(next, updatedTarget...)
			next = append(next, lines[targetEnd+1:]...)
			lines = next
			changed = true
		}

		index += len(updatedTarget)
		targetCount++
	}

	if targetCount == 0 {
		return "", false, fmt.Errorf("no .target declarations found")
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func isTuistTargetDeclaration(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), ".target(")
}

func syncTuistTargetMinTargetBlock(lines []string) ([]string, bool, error) {
	updated := append([]string(nil), lines...)
	changed := false
	fieldIndent := tuistTargetFieldIndent(updated)

	deploymentFound := false
	for index, line := range updated {
		if !strings.HasPrefix(strings.TrimSpace(line), "deploymentTargets:") {
			continue
		}
		deploymentFound = true
		replacement := leadingIndent(line) + "deploymentTargets: .iOS(minTarget),"
		if replacement != line {
			updated[index] = replacement
			changed = true
		}
	}
	if !deploymentFound {
		anchorIndex := -1
		for index, line := range updated {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "bundleId:") {
				anchorIndex = index
				break
			}
			if anchorIndex < 0 && strings.HasPrefix(trimmed, "product:") {
				anchorIndex = index
			}
		}
		if anchorIndex < 0 {
			return nil, false, fmt.Errorf("deploymentTargets insertion anchor not found")
		}
		updated = insertSyncLine(updated, anchorIndex+1, fieldIndent+"deploymentTargets: .iOS(minTarget),")
		changed = true
	}

	settingsIndex := -1
	for index, line := range updated {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "settings:") {
			continue
		}
		if !strings.Contains(trimmed, ".settings(") {
			return nil, false, fmt.Errorf("unsupported settings declaration %q; expected .settings(base: [...])", trimmed)
		}
		settingsIndex = index
		break
	}

	if settingsIndex < 0 {
		insertAt := len(updated) - 1
		updated = insertSyncLines(updated, insertAt, []string{
			fieldIndent + "settings: .settings(",
			fieldIndent + "    base: [",
			fieldIndent + `        "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),`,
			fieldIndent + "    ]",
			fieldIndent + "),",
		})
		return updated, true, nil
	}

	settingsEnd, err := findDelimitedBlockEnd(updated, settingsIndex, "(", ")")
	if err != nil {
		return nil, false, fmt.Errorf("unterminated settings declaration: %w", err)
	}
	baseIndex := -1
	for index := settingsIndex + 1; index < settingsEnd; index++ {
		if strings.Contains(updated[index], "base: [") {
			baseIndex = index
			break
		}
	}
	if baseIndex < 0 {
		return nil, false, fmt.Errorf("settings base insertion anchor not found")
	}
	baseEnd, err := findDelimitedBlockEnd(updated, baseIndex, "[", "]")
	if err != nil {
		return nil, false, fmt.Errorf("unterminated settings base dictionary: %w", err)
	}
	if baseEnd == baseIndex {
		return nil, false, fmt.Errorf("unsupported one-line settings base dictionary; expected a multiline base: [...] declaration")
	}

	buildSettingFound := false
	for index := baseIndex + 1; index < baseEnd; index++ {
		trimmed := strings.TrimSpace(updated[index])
		if strings.HasPrefix(trimmed, "//") || !strings.Contains(updated[index], `"IPHONEOS_DEPLOYMENT_TARGET":`) {
			continue
		}
		buildSettingFound = true
		replacement, err := rewriteTuistDeploymentTarget(updated[index])
		if err != nil {
			return nil, false, err
		}
		if replacement != updated[index] {
			updated[index] = replacement
			changed = true
		}
	}
	if !buildSettingFound {
		insertLine := leadingIndent(updated[baseIndex]) + `    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),`
		updated = insertSyncLine(updated, baseIndex+1, insertLine)
		changed = true
	}

	return updated, changed, nil
}

func tuistTargetFieldIndent(lines []string) string {
	for index := 1; index < len(lines)-1; index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed != "" && trimmed != ")" && trimmed != ")," {
			return leadingIndent(lines[index])
		}
	}
	return leadingIndent(lines[0]) + "    "
}

func rewriteTuistDeploymentTarget(line string) (string, error) {
	const key = `"IPHONEOS_DEPLOYMENT_TARGET":`
	keyIndex := strings.Index(line, key)
	if keyIndex < 0 {
		return line, nil
	}

	stringCallIndex := strings.Index(line[keyIndex+len(key):], ".string(")
	if stringCallIndex < 0 {
		return "", fmt.Errorf("unsupported IPHONEOS_DEPLOYMENT_TARGET setting %q; expected .string(...)", strings.TrimSpace(line))
	}
	valueStart := keyIndex + len(key) + stringCallIndex + len(".string(")
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}
	if strings.HasPrefix(line[valueStart:], "minTarget") {
		remainder := strings.TrimSpace(line[valueStart+len("minTarget"):])
		if strings.HasPrefix(remainder, ")") {
			return line, nil
		}
	}

	replacement, ok := replaceSwiftStringLiteralAt(line, valueStart, "minTarget")
	if !ok {
		return "", fmt.Errorf("unsupported IPHONEOS_DEPLOYMENT_TARGET setting %q; expected a string literal", strings.TrimSpace(line))
	}
	quotedIdentifier := fmt.Sprintf("%q", "minTarget")
	return replacement[:valueStart] + "minTarget" + replacement[valueStart+len(quotedIdentifier):], nil
}

func insertSyncLine(lines []string, index int, line string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(lines) {
		index = len(lines)
	}

	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:index]...)
	updated = append(updated, line)
	updated = append(updated, lines[index:]...)
	return updated
}

func joinSyncLines(lines []string, hasTrailingNewline bool) string {
	updated := strings.Join(lines, "\n")
	if hasTrailingNewline && !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	return updated
}
