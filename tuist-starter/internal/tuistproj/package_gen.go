package tuistproj

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const packageTemplatePath = "templates/package.swift.tmpl"

var (
	//go:embed templates/package.swift.tmpl
	packageTemplatesFS embed.FS

	parsedPackageTemplate = template.Must(
		template.New("package.swift.tmpl").ParseFS(packageTemplatesFS, packageTemplatePath),
	)
)

// PackageType defines generated module package flavor.
type PackageType string

const (
	PackageTypeInterface PackageType = "interface"
	PackageTypeImpl      PackageType = "impl"
)

// ExternalProductDep describes an external package dependency with a separate product name.
type ExternalProductDep struct {
	PackageName string // e.g., "swift-relux"
	ProductName string // e.g., "Relux"
	URL         string // e.g., "https://github.com/relux-works/swift-relux.git"
	Version     string // e.g., `from: "9.0.0"`
}

// PackageGenerationInput configures Package.swift generation.
type PackageGenerationInput struct {
	ModuleName   string
	Type         PackageType
	Dependencies []string
	ExternalDeps []ExternalProductDep
	Platform     string
	Config       config.ProjectConfig
}

type packageTemplateData struct {
	PackageName          string
	ProductName          string
	TargetName           string
	Platform             string
	PackageDependencies  []packageDependency
	TargetDependencies   []packageDependency
	ExternalDependencies []externalPackageDep
	ExternalProducts     []externalProductDep
	ManifestComment      string
	SwiftToolsVersionTag string
	SwiftPackageSettings []string
}

type packageDependency struct {
	Name string
	Path string
}

type externalPackageDep struct {
	URL     string
	Version string
}

type externalProductDep struct {
	ProductName string
	PackageName string
}

// GeneratePackageSwift renders Package.swift using embedded templates.
func GeneratePackageSwift(input PackageGenerationInput) (string, error) {
	moduleName := strings.TrimSpace(input.ModuleName)
	if moduleName == "" {
		return "", fmt.Errorf("ModuleName is required")
	}

	platform := normalizePackagePlatform(input.Platform)
	if platform == "" {
		return "", fmt.Errorf("Platform is required")
	}

	moduleType := normalizePackageType(input.Type)
	if moduleType == "" {
		return "", fmt.Errorf("Type must be %q or %q", PackageTypeInterface, PackageTypeImpl)
	}

	additionalDependencies := normalizeDependencies(input.Dependencies)

	externalDeps := buildExternalPackageDeps(input.ExternalDeps)
	externalProducts := buildExternalProductDeps(input.ExternalDeps)
	effectiveSwift := input.Config.EffectiveSwiftSettings()

	data := packageTemplateData{
		Platform:             platform,
		SwiftToolsVersionTag: effectiveSwift.ToolsVersion,
		SwiftPackageSettings: effectiveSwift.PackageSwiftSettings(),
		ExternalDependencies: externalDeps,
		ExternalProducts:     externalProducts,
	}

	switch moduleType {
	case PackageTypeInterface:
		data.ManifestComment = "Interface package"
		data.PackageName = moduleName
		data.ProductName = moduleName
		data.TargetName = moduleName
		data.PackageDependencies = buildPackageDependencies(additionalDependencies)
		data.TargetDependencies = buildProductDependencies(additionalDependencies)

	case PackageTypeImpl:
		implName := moduleName + "Impl"
		data.ManifestComment = "Implementation package"
		data.PackageName = implName
		data.ProductName = implName
		data.TargetName = implName

		allDependencies := make([]string, 0, len(additionalDependencies)+1)
		allDependencies = append(allDependencies, moduleName)
		allDependencies = append(allDependencies, additionalDependencies...)
		allDependencies = normalizeDependencies(allDependencies)

		data.PackageDependencies = buildPackageDependencies(allDependencies)
		data.TargetDependencies = buildProductDependencies(allDependencies)

	default:
		return "", fmt.Errorf("unsupported package type %q", moduleType)
	}

	var rendered strings.Builder
	if err := parsedPackageTemplate.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("render package.swift template: %w", err)
	}

	return rendered.String(), nil
}

func normalizePackageType(raw PackageType) PackageType {
	switch PackageType(strings.ToLower(strings.TrimSpace(string(raw)))) {
	case PackageTypeInterface:
		return PackageTypeInterface
	case PackageTypeImpl:
		return PackageTypeImpl
	default:
		return ""
	}
}

func normalizePackagePlatform(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, ".") {
		return value
	}
	return "." + value
}

func normalizeDependencies(dependencies []string) []string {
	normalized := make([]string, 0, len(dependencies))
	seen := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		name := strings.TrimSpace(dependency)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized
}

func buildPackageDependencies(names []string) []packageDependency {
	dependencies := make([]packageDependency, 0, len(names))
	for _, name := range names {
		dependencies = append(dependencies, packageDependency{
			Name: name,
			Path: "../" + name,
		})
	}
	return dependencies
}

func buildProductDependencies(names []string) []packageDependency {
	dependencies := make([]packageDependency, 0, len(names))
	for _, name := range names {
		dependencies = append(dependencies, packageDependency{
			Name: name,
			Path: "../" + name,
		})
	}
	return dependencies
}

func buildExternalPackageDeps(deps []ExternalProductDep) []externalPackageDep {
	out := make([]externalPackageDep, 0, len(deps))
	for _, d := range deps {
		out = append(out, externalPackageDep{
			URL:     d.URL,
			Version: d.Version,
		})
	}
	return out
}

func buildExternalProductDeps(deps []ExternalProductDep) []externalProductDep {
	out := make([]externalProductDep, 0, len(deps))
	for _, d := range deps {
		out = append(out, externalProductDep{
			ProductName: d.ProductName,
			PackageName: d.PackageName,
		})
	}
	return out
}
