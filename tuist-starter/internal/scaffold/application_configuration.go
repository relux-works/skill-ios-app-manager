package scaffold

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

type ApplicationConfigurationSyncResult struct {
	Scanned []string
	Updated []string
}

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "application-configuration",
		Short:        "Sync generic application runtime configuration",
		Dependencies: []string{"init"},
		Run:          runGenerateApplicationConfiguration,
	})
}

func runGenerateApplicationConfiguration(input GenerateInput) (GenerateResult, error) {
	result, err := SyncApplicationConfiguration(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) > 0 {
		return GenerateResult{
			Message: fmt.Sprintf("regenerated application configuration in %d file(s)\n", len(result.Updated)),
		}, nil
	}

	return GenerateResult{
		Message: "application configuration already up to date\n",
	}, nil
}

func SyncApplicationConfiguration(projectRoot string, cfg config.ProjectConfig) (ApplicationConfigurationSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ApplicationConfigurationSyncResult{}, fmt.Errorf("project root is required")
	}
	if err := validateApplicationConfigurationConfig(cfg); err != nil {
		return ApplicationConfigurationSyncResult{}, err
	}

	appName := normalizeAppName(cfg.AppName)
	sharedConfigurationModuleName := appGroupSharedConfigurationModuleName(cfg)
	result := ApplicationConfigurationSyncResult{
		Scanned: make([]string, 0, 8),
		Updated: make([]string, 0, 8),
	}

	configurationPath := configurationApplicationConfigurationPath(root, appName)
	staleConfigurationPaths := staleConfigurationApplicationConfigurationPaths(root, appName, configurationPath)
	result.Scanned = append(result.Scanned, configurationPath)
	result.Scanned = append(result.Scanned, staleConfigurationPaths...)
	updated, err := writeFileIfChanged(configurationPath, GenerateConfigurationApplicationConfiguration(cfg))
	if err != nil {
		return result, fmt.Errorf("sync Configuration+ApplicationConfiguration.swift: %w", err)
	}
	if updated {
		result.Updated = appendUniqueStrings(result.Updated, configurationPath)
	}
	for _, stalePath := range staleConfigurationPaths {
		updated, err := removeFileIfExists(stalePath)
		if err != nil {
			return result, fmt.Errorf("remove stale Configuration+ApplicationConfiguration.swift: %w", err)
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, stalePath)
		}
	}

	packageSwiftPath := sharedConfigurationPackageSwiftPath(root, cfg)
	infoPlistReadingPath := sharedConfigurationInfoPlistReadingSourcePath(root, cfg)
	sourcePath := applicationConfigurationSharedConfigurationSourcePath(root, cfg)
	result.Scanned = append(result.Scanned, packageSwiftPath, infoPlistReadingPath, sourcePath)
	updatedPaths, err := syncApplicationConfigurationSharedConfigurationPackage(root, cfg)
	if err != nil {
		return result, err
	}
	result.Updated = appendUniqueStrings(result.Updated, updatedPaths...)

	rootPackagePath := filepath.Join(root, "Package.swift")
	result.Scanned = append(result.Scanned, rootPackagePath)
	updated, err = syncRootPackageSharedConfigurationDependency(root, cfg)
	if err != nil {
		return result, err
	}
	if updated {
		result.Updated = appendUniqueStrings(result.Updated, rootPackagePath)
	}

	updated, err = cleanupRootPackageLegacySharedConfigurationDependency(root, cfg)
	if err != nil {
		return result, err
	}
	if updated {
		result.Updated = appendUniqueStrings(result.Updated, rootPackagePath)
	}

	projectManifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return result, err
	}
	for _, manifestPath := range projectManifestPaths {
		result.Scanned = appendUniqueStrings(result.Scanned, manifestPath)
		updated, err := syncProjectManifestApplicationConfiguration(manifestPath, cfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}

		updated, err = syncProjectManifestSharedConfigurationDependency(manifestPath, sharedConfigurationModuleName)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}

		updated, err = cleanupProjectManifestLegacySharedConfigurationDependency(manifestPath, cfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}
	}

	return result, nil
}

func validateApplicationConfigurationConfig(cfg config.ProjectConfig) error {
	issues := make([]string, 0, 4)
	if strings.TrimSpace(cfg.AppName) == "" {
		issues = append(issues, "app_name is required")
	}
	if strings.TrimSpace(cfg.BundleID) == "" {
		issues = append(issues, "bundle_id is required")
	}
	if strings.TrimSpace(cfg.TeamID) == "" {
		issues = append(issues, "team_id is required")
	}
	if moduleName := strings.TrimSpace(cfg.SharedConfig.ModuleName); moduleName != "" && !swiftIdentifierPattern.MatchString(moduleName) {
		issues = append(issues, fmt.Sprintf("shared_config.module_name %q must be a valid Swift module identifier", moduleName))
	}
	if len(issues) > 0 {
		return fmt.Errorf("invalid application configuration config: %s", strings.Join(issues, "; "))
	}
	return nil
}

func configurationApplicationConfigurationPath(root string, appName string) string {
	return configurationFilePath(root, appName, "Configuration+ApplicationConfiguration.swift")
}

func staleConfigurationApplicationConfigurationPaths(root string, appName string, selectedPath string) []string {
	return staleConfigurationFilePaths(root, appName, "Configuration+ApplicationConfiguration.swift", selectedPath)
}

func syncProjectManifestApplicationConfiguration(path string, cfg config.ProjectConfig) (bool, error) {
	return syncProjectManifestInfoPlistDictionary(
		path,
		applicationConfigurationInfoPlistKey,
		true,
		func(indent string) []string {
			return renderApplicationConfigurationInfoPlistLines(indent, cfg)
		},
	)
}

func renderApplicationConfigurationInfoPlistLines(indent string, cfg config.ProjectConfig) []string {
	lines := []string{
		indent + strconv.Quote(applicationConfigurationInfoPlistKey) + ": .dictionary([",
		indent + "    " + strconv.Quote("appName") + ": .string(" + strconv.Quote(strings.TrimSpace(cfg.AppName)) + "),",
		indent + "    " + strconv.Quote("applicationBundleIdentifier") + ": .string(" + strconv.Quote(strings.TrimSpace(cfg.BundleID)) + "),",
		indent + "    " + strconv.Quote("developmentTeamID") + ": .string(" + strconv.Quote(strings.TrimSpace(cfg.TeamID)) + "),",
	}

	if urlScheme := strings.TrimSpace(cfg.URLScheme); urlScheme != "" {
		lines = append(lines, indent+"    "+strconv.Quote("urlScheme")+": .string("+strconv.Quote(urlScheme)+"),")
	}

	lines = append(lines, indent+"]),")
	return lines
}
