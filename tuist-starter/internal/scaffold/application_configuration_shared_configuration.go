package scaffold

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const applicationConfigurationInfoPlistKey = "ApplicationConfiguration"

func applicationConfigurationSharedConfigurationSourcePath(root string, cfg config.ProjectConfig) string {
	return filepath.Join(sharedConfigurationSourcesPath(root, cfg), "ApplicationConfiguration.swift")
}

func GenerateApplicationConfigurationSharedConfigurationSwift(cfg config.ProjectConfig) string {
	appName := normalizeAppName(cfg.AppName)
	typePrefix := appGroupSharedConfigurationTypePrefix(appName)
	lowerTypePrefix := lowerFirst(typePrefix)

	var b strings.Builder
	b.WriteString("import Foundation\n\n")
	b.WriteString("public enum " + typePrefix + "ApplicationConfigurationField: String, Sendable {\n")
	b.WriteString("    case appName\n")
	b.WriteString("    case applicationBundleIdentifier\n")
	b.WriteString("    case developmentTeamID\n")
	b.WriteString("    case urlScheme\n\n")
	b.WriteString("    public var infoPlistKey: String {\n")
	b.WriteString("        " + strconv.Quote(applicationConfigurationInfoPlistKey) + "\n")
	b.WriteString("    }\n\n")
	b.WriteString("    public var dictionaryKey: String {\n")
	b.WriteString("        rawValue\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("public struct " + typePrefix + "ApplicationConfiguration: Equatable, Sendable {\n")
	b.WriteString("    public let appName: String\n")
	b.WriteString("    public let applicationBundleIdentifier: String\n")
	b.WriteString("    public let developmentTeamID: String\n")
	b.WriteString("    public let urlScheme: String?\n\n")
	b.WriteString("    public init(\n")
	b.WriteString("        appName: String,\n")
	b.WriteString("        applicationBundleIdentifier: String,\n")
	b.WriteString("        developmentTeamID: String,\n")
	b.WriteString("        urlScheme: String?\n")
	b.WriteString("    ) {\n")
	b.WriteString("        self.appName = appName\n")
	b.WriteString("        self.applicationBundleIdentifier = applicationBundleIdentifier\n")
	b.WriteString("        self.developmentTeamID = developmentTeamID\n")
	b.WriteString("        self.urlScheme = urlScheme\n")
	b.WriteString("    }\n\n")
	b.WriteString("    public static func read(from bundle: Bundle = .main) throws -> Self {\n")
	b.WriteString("        try Self(\n")
	b.WriteString("            appName: bundle." + lowerTypePrefix + "String(for: Field.appName.infoPlistKey, dictionaryKey: Field.appName.dictionaryKey),\n")
	b.WriteString("            applicationBundleIdentifier: bundle." + lowerTypePrefix + "String(for: Field.applicationBundleIdentifier.infoPlistKey, dictionaryKey: Field.applicationBundleIdentifier.dictionaryKey),\n")
	b.WriteString("            developmentTeamID: bundle." + lowerTypePrefix + "String(for: Field.developmentTeamID.infoPlistKey, dictionaryKey: Field.developmentTeamID.dictionaryKey),\n")
	b.WriteString("            urlScheme: bundle." + lowerTypePrefix + "OptionalString(for: Field.urlScheme.infoPlistKey, dictionaryKey: Field.urlScheme.dictionaryKey)\n")
	b.WriteString("        )\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("private typealias Field = " + typePrefix + "ApplicationConfigurationField\n")

	return b.String()
}

func syncApplicationConfigurationSharedConfigurationPackage(root string, cfg config.ProjectConfig) ([]string, error) {
	sourcePath := applicationConfigurationSharedConfigurationSourcePath(root, cfg)
	updated, err := syncSharedConfigurationSupportPackage(root, cfg)
	if err != nil {
		return nil, err
	}

	changed, err := writeFileIfChanged(sourcePath, GenerateApplicationConfigurationSharedConfigurationSwift(cfg))
	if err != nil {
		return nil, err
	}
	if changed {
		updated = append(updated, sourcePath)
	}

	return updated, nil
}
