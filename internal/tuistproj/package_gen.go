package tuistproj

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
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

// PackageGenerationInput configures Package.swift generation.
type PackageGenerationInput struct {
	ModuleName   string
	Type         PackageType
	Dependencies []string
	Platform     string
}

type packageTemplateData struct {
	PackageName          string
	ProductName          string
	TargetName           string
	Platform             string
	PackageDependencies  []packageDependency
	TargetDependencies   []packageDependency
	ManifestComment      string
	SwiftToolsVersionTag string
}

type packageDependency struct {
	Name string
	Path string
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

	data := packageTemplateData{
		Platform:             platform,
		SwiftToolsVersionTag: "6.0",
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
