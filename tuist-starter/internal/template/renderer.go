package template

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	defaultRendererRootDir    = "."
	defaultRendererModulesDir = "Packages"
	templateDir               = "tuist"
)

var templateNames = []string{
	"Tuist.swift.tmpl",
	"Project.swift.tmpl",
	"Workspace.swift.tmpl",
	"Package.swift.tmpl",
}

//go:embed tuist/*.tmpl
var templatesFS embed.FS

// RendererOption configures a Renderer.
type RendererOption func(*Renderer)

// Renderer renders embedded Tuist templates using ProjectConfig variables.
type Renderer struct {
	rootDir               string
	discoverLocalPackages func(modulesPath string) ([]string, error)
}

type rendererTemplateData struct {
	config.ProjectConfig
	LocalPackages        []string
	SwiftToolsVersion    string
	SwiftBuildSettings   []config.SwiftBuildSetting
	PackageSwiftSettings []string
}

// NewRenderer creates a renderer for Tuist template manifests.
func NewRenderer(options ...RendererOption) *Renderer {
	r := &Renderer{
		rootDir: defaultRendererRootDir,
	}
	r.discoverLocalPackages = r.findLocalPackages

	for _, option := range options {
		if option != nil {
			option(r)
		}
	}

	return r
}

// WithRootDir overrides the base directory used to inspect ModulesPath.
func WithRootDir(rootDir string) RendererOption {
	return func(r *Renderer) {
		if trimmed := strings.TrimSpace(rootDir); trimmed != "" {
			r.rootDir = trimmed
		}
	}
}

// Render renders all Tuist templates and returns map[fileName]renderedContent.
func (r *Renderer) Render(cfg config.ProjectConfig) (map[string]string, error) {
	data, err := r.buildTemplateData(cfg)
	if err != nil {
		return nil, err
	}

	rendered := make(map[string]string, len(templateNames))
	for _, templateName := range templateNames {
		content, err := r.renderTemplate(templateName, data)
		if err != nil {
			return nil, err
		}

		outputFile := strings.TrimSuffix(templateName, ".tmpl")
		rendered[outputFile] = content
	}

	return rendered, nil
}

func (r *Renderer) buildTemplateData(cfg config.ProjectConfig) (rendererTemplateData, error) {
	normalized := normalizeProjectConfig(cfg)

	localPackages, err := r.discoverLocalPackages(normalized.ModulesPath)
	if err != nil {
		return rendererTemplateData{}, err
	}

	effectiveSwift := normalized.EffectiveSwiftSettings()

	return rendererTemplateData{
		ProjectConfig:        normalized,
		LocalPackages:        localPackages,
		SwiftToolsVersion:    effectiveSwift.ToolsVersion,
		SwiftBuildSettings:   effectiveSwift.XcodeBuildSettings(),
		PackageSwiftSettings: effectiveSwift.PackageSwiftSettings(),
	}, nil
}

func (r *Renderer) renderTemplate(templateName string, data rendererTemplateData) (string, error) {
	templatePath := filepath.ToSlash(filepath.Join(templateDir, templateName))

	tpl, err := template.New(templateName).
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"appGroupDictionaryKey":                 AppGroupSwiftIdentifier,
			"appGroupInfoPlistKey":                  AppGroupInfoPlistKey,
			"configKind":                            configKind,
			"exportComplianceInfoPlistLines":        exportComplianceInfoPlistLines,
			"infoPlistKey":                          InfoPlistKey,
			"packageBuildSetting":                   packageBuildSetting,
			"packagePath":                           packagePath,
			"privacyUsageDescriptionInfoPlistLines": privacyUsageDescriptionInfoPlistLines,
			"presentationInfoPlistLines":            presentationInfoPlistLines,
			"projectBuildSetting":                   projectBuildSetting,
			"swiftLiteral":                          swiftLiteral,
			"targetDestinationsExpression":          TargetDestinationsExpression,
		}).
		ParseFS(templatesFS, templatePath)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", templateName, err)
	}

	var out strings.Builder
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render template %q: %w", templateName, err)
	}

	return out.String(), nil
}

func (r *Renderer) findLocalPackages(modulesPath string) ([]string, error) {
	path := strings.TrimSpace(modulesPath)
	if path == "" {
		return nil, nil
	}

	scanPath := path
	if !filepath.IsAbs(scanPath) {
		scanPath = filepath.Join(r.rootDir, scanPath)
	}

	entries, err := os.ReadDir(scanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read modules directory %q: %w", path, err)
	}

	localPackages := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(scanPath, entry.Name(), "Package.swift")
		if _, statErr := os.Stat(manifestPath); statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat local package manifest %q: %w", manifestPath, statErr)
		}

		localPackages = append(localPackages, entry.Name())
	}

	sort.Strings(localPackages)
	return localPackages, nil
}

func normalizeProjectConfig(cfg config.ProjectConfig) config.ProjectConfig {
	out := cfg

	out.AppName = strings.TrimSpace(out.AppName)
	out.BundleID = strings.TrimSpace(out.BundleID)
	out.TeamID = strings.TrimSpace(out.TeamID)
	out.OrgName = strings.TrimSpace(out.OrgName)
	out.MarketingVersion = strings.TrimSpace(out.MarketingVersion)
	out.ProjectVersion = strings.TrimSpace(out.ProjectVersion)
	out.SwiftVersion = strings.TrimSpace(out.SwiftVersion)
	out.MinTarget = strings.TrimSpace(out.MinTarget)
	out.URLScheme = strings.TrimSpace(out.URLScheme)
	out.AppGroups = normalizeStringSlice(out.AppGroups)
	out.BackgroundModes = normalizeStringSlice(out.BackgroundModes)
	out.PrivacyUsageDescriptions.BluetoothAlways = strings.TrimSpace(out.PrivacyUsageDescriptions.BluetoothAlways)
	out.PrivacyUsageDescriptions.BluetoothPeripheral = strings.TrimSpace(out.PrivacyUsageDescriptions.BluetoothPeripheral)
	out.PrivacyUsageDescriptions.Camera = strings.TrimSpace(out.PrivacyUsageDescriptions.Camera)
	out.PrivacyUsageDescriptions.Microphone = strings.TrimSpace(out.PrivacyUsageDescriptions.Microphone)
	out.PrivacyUsageDescriptions.LocalNetwork = strings.TrimSpace(out.PrivacyUsageDescriptions.LocalNetwork)
	out.Configurations = normalizeConfigurations(out.Configurations)
	out.ModulesPath = normalizeModulesPath(out.ModulesPath)
	out.ProjectSettings.Swift.LanguageMode = strings.TrimSpace(out.ProjectSettings.Swift.LanguageMode)
	out.ProjectSettings.Swift.Concurrency.DefaultActorIsolation = strings.TrimSpace(out.ProjectSettings.Swift.Concurrency.DefaultActorIsolation)
	out.ProjectSettings.Swift.Concurrency.StrictChecking = strings.TrimSpace(out.ProjectSettings.Swift.Concurrency.StrictChecking)

	return out
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func normalizeConfigurations(configurations []string) []string {
	normalized := normalizeStringSlice(configurations)
	if len(normalized) == 0 {
		return []string{"Debug", "Release"}
	}
	return normalized
}

func normalizeModulesPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = defaultRendererModulesDir
	}
	return filepath.ToSlash(trimmed)
}

func configKind(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "debug") {
		return "debug"
	}
	return "release"
}

func packagePath(modulesPath, packageName string) string {
	modules := strings.Trim(strings.TrimSpace(modulesPath), "/")
	name := strings.Trim(strings.TrimSpace(packageName), "/")
	if modules == "" {
		return filepath.ToSlash("./" + name)
	}
	return filepath.ToSlash("./" + filepath.Join(modules, name))
}

func swiftLiteral(value string) string {
	return strconv.Quote(value)
}

func projectBuildSetting(key, value string) string {
	return fmt.Sprintf(`"%s": %s,`, key, strconv.Quote(value))
}

func TargetDestinationsExpression(value any) string {
	cfg := projectConfigFromTemplateValue(value)
	if !cfg.UsesExplicitPlatformDestinations() {
		return ".iOS"
	}

	destinations := make([]string, 0, 3)
	if cfg.IOSTargetEnabled() {
		destinations = append(destinations, ".iPhone")
	}
	if cfg.IPadTargetEnabled() {
		destinations = append(destinations, ".iPad")
	}
	if cfg.MacWithIPadDesignTargetEnabled() {
		destinations = append(destinations, ".macWithiPadDesign")
	}
	if len(destinations) == 0 {
		destinations = append(destinations, ".iPhone")
	}

	return "[" + strings.Join(destinations, ", ") + "]"
}

func projectConfigFromTemplateValue(value any) config.ProjectConfig {
	switch typed := value.(type) {
	case config.ProjectConfig:
		return typed
	case rendererTemplateData:
		return typed.ProjectConfig
	case *rendererTemplateData:
		if typed == nil {
			return config.ProjectConfig{}
		}
		return typed.ProjectConfig
	default:
		return config.ProjectConfig{}
	}
}

func packageBuildSetting(key, value string) string {
	return fmt.Sprintf(`"%s": %s,`, key, strconv.Quote(value))
}

// InfoPlistKey converts an identifier to an uppercase underscore-separated key.
// Example: "group.com.example-demo" -> "GROUP_COM_EXAMPLE_DEMO"
func InfoPlistKey(s string) string {
	var b strings.Builder
	previousUnderscore := false

	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - ('a' - 'A'))
			previousUnderscore = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			previousUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			previousUnderscore = false
		default:
			if !previousUnderscore {
				b.WriteByte('_')
				previousUnderscore = true
			}
		}
	}

	return strings.Trim(b.String(), "_")
}

// AppGroupInfoPlistKey returns a short app-group Info.plist key.
// Example: bundle "com.example.app", group "group.com.example.app.shared" -> "APP_GROUP_SHARED".
func AppGroupInfoPlistKey(bundleID, group string) string {
	key := InfoPlistKey(appGroupStem(bundleID, group))
	if key == "" {
		key = "MAIN"
	}
	return "APP_GROUP_" + key
}

// AppGroupSwiftIdentifier returns a Swift lowerCamelCase property name for an app group.
// Example: bundle "com.example.app", group "group.com.example.app.shared" -> "shared".
func AppGroupSwiftIdentifier(bundleID, group string) string {
	identifier := lowerCamelIdentifier(appGroupStem(bundleID, group))
	if identifier == "" {
		return "main"
	}
	return avoidSwiftKeyword(identifier)
}

func appGroupStem(bundleID, group string) string {
	bundleID = strings.TrimSpace(bundleID)
	group = strings.TrimSpace(group)
	if group == "" {
		return ""
	}

	if bundleID != "" {
		prefix := "group." + bundleID
		switch {
		case group == prefix:
			return "main"
		case strings.HasPrefix(group, prefix+"."):
			return strings.TrimPrefix(group, prefix+".")
		}
	}

	return strings.TrimPrefix(group, "group.")
}

func lowerCamelIdentifier(raw string) string {
	parts := alphanumericParts(raw)
	if len(parts) == 0 {
		return ""
	}

	var b strings.Builder
	for index, part := range parts {
		part = strings.ToLower(part)
		if index == 0 {
			b.WriteString(applyKnownAcronymSuffixes(part))
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(applyKnownAcronymSuffixes(part[1:]))
		}
	}

	out := b.String()
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "appGroup" + strings.ToUpper(out[:1]) + out[1:]
	}
	return out
}

func applyKnownAcronymSuffixes(part string) string {
	for _, acronym := range []string{"sdk"} {
		if strings.HasSuffix(part, acronym) && len(part) > len(acronym) {
			return part[:len(part)-len(acronym)] + strings.ToUpper(acronym)
		}
	}
	return part
}

func alphanumericParts(raw string) []string {
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

func avoidSwiftKeyword(identifier string) string {
	switch identifier {
	case "associatedtype", "class", "deinit", "enum", "extension", "fileprivate",
		"func", "import", "init", "inout", "internal", "let", "open", "operator",
		"private", "precedencegroup", "protocol", "public", "rethrows", "static",
		"struct", "subscript", "typealias", "var", "break", "case", "catch",
		"continue", "default", "defer", "do", "else", "fallthrough", "for",
		"guard", "if", "in", "repeat", "return", "throw", "switch", "where",
		"while", "as", "Any", "false", "is", "nil", "self", "Self", "super",
		"throws", "true", "try":
		return identifier + "Group"
	default:
		return identifier
	}
}
