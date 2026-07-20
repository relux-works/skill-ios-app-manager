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
	fieldTypeName := typePrefix + "ApplicationConfigurationField"
	configurationTypeName := typePrefix + "ApplicationConfiguration"

	var b strings.Builder
	b.WriteString("import Foundation\n\n")
	if len(fieldTypeName) > 40 {
		b.WriteString("// swiftlint:disable:next type_name\n")
	}
	b.WriteString("public enum " + fieldTypeName + ": String, Sendable {\n")
	b.WriteString("    case appName\n")
	b.WriteString("    case applicationBundleIdentifier\n")
	b.WriteString("    case developmentTeamID\n")
	if cfg.HasRuntimeProfiles() {
		b.WriteString("    case distributionProfile\n")
	}
	b.WriteString("    case urlScheme\n\n")
	b.WriteString("    public var infoPlistKey: String {\n")
	b.WriteString("        " + strconv.Quote(applicationConfigurationInfoPlistKey) + "\n")
	b.WriteString("    }\n\n")
	b.WriteString("    public var dictionaryKey: String {\n")
	b.WriteString("        rawValue\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	if len(configurationTypeName) > 40 {
		b.WriteString("// swiftlint:disable:next type_name\n")
	}
	b.WriteString("public struct " + configurationTypeName + ": Equatable, Sendable {\n")
	b.WriteString("    public let appName: String\n")
	b.WriteString("    public let applicationBundleIdentifier: String\n")
	b.WriteString("    public let developmentTeamID: String\n")
	if cfg.HasRuntimeProfiles() {
		b.WriteString("    public let distributionProfile: String\n")
	}
	b.WriteString("    public let urlScheme: String?\n\n")
	b.WriteString("    public init(\n")
	b.WriteString("        appName: String,\n")
	b.WriteString("        applicationBundleIdentifier: String,\n")
	b.WriteString("        developmentTeamID: String,\n")
	if cfg.HasRuntimeProfiles() {
		b.WriteString("        distributionProfile: String,\n")
	}
	b.WriteString("        urlScheme: String?\n")
	b.WriteString("    ) {\n")
	b.WriteString("        self.appName = appName\n")
	b.WriteString("        self.applicationBundleIdentifier = applicationBundleIdentifier\n")
	b.WriteString("        self.developmentTeamID = developmentTeamID\n")
	if cfg.HasRuntimeProfiles() {
		b.WriteString("        self.distributionProfile = distributionProfile\n")
	}
	b.WriteString("        self.urlScheme = urlScheme\n")
	b.WriteString("    }\n\n")
	b.WriteString("    public static func read(from bundle: Bundle = .main) throws -> Self {\n")
	b.WriteString("        try Self(\n")
	writeReadArgument := func(label string, method string, field string, trailingComma bool) {
		b.WriteString("            " + label + ": bundle." + method + "(\n")
		b.WriteString("                for: Field." + field + ".infoPlistKey,\n")
		b.WriteString("                dictionaryKey: Field." + field + ".dictionaryKey\n")
		b.WriteString("            )")
		if trailingComma {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	writeReadArgument("appName", lowerTypePrefix+"String", "appName", true)
	writeReadArgument(
		"applicationBundleIdentifier",
		lowerTypePrefix+"String",
		"applicationBundleIdentifier",
		true,
	)
	writeReadArgument("developmentTeamID", lowerTypePrefix+"String", "developmentTeamID", true)
	if cfg.HasRuntimeProfiles() {
		writeReadArgument("distributionProfile", lowerTypePrefix+"String", "distributionProfile", true)
	}
	writeReadArgument("urlScheme", lowerTypePrefix+"OptionalString", "urlScheme", false)
	b.WriteString("        )\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("private typealias Field = " + fieldTypeName + "\n")

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
