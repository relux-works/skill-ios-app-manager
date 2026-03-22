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
			"configKind":          configKind,
			"infoPlistKey":        InfoPlistKey,
			"packageBuildSetting": packageBuildSetting,
			"packagePath":         packagePath,
			"projectBuildSetting": projectBuildSetting,
			"swiftLiteral":        swiftLiteral,
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

func packageBuildSetting(key, value string) string {
	return fmt.Sprintf(`"%s": %s,`, key, strconv.Quote(value))
}

// InfoPlistKey converts a dotted identifier to an uppercase underscore-separated key.
// Example: "group.com.example.demo" → "GROUP_COM_EXAMPLE_DEMO"
func InfoPlistKey(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, ".", "_"))
}
