package scaffold

import (
	"fmt"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templatepkg "github.com/relux-works/ios-app-manager/internal/template"
)

func validateAppGroupsConfig(cfg config.ProjectConfig) error {
	issues := make([]string, 0)
	seenGroups := make(map[string]int)
	seenProperties := make(map[string]string)

	if moduleName := appGroupSharedConfigurationModuleName(cfg); !swiftIdentifierPattern.MatchString(moduleName) {
		issues = append(issues, fmt.Sprintf("shared_config.module_name %q must be a valid Swift module identifier", moduleName))
	}

	for index, raw := range cfg.AppGroups {
		group := strings.TrimSpace(raw)
		if group == "" {
			issues = append(issues, fmt.Sprintf("app_groups[%d] must not be empty", index))
			continue
		}
		if !strings.HasPrefix(group, "group.") {
			issues = append(issues, fmt.Sprintf("app_groups[%d] %q must start with \"group.\"", index, group))
		}
		if previousIndex, ok := seenGroups[group]; ok {
			issues = append(issues, fmt.Sprintf("app_groups[%d] %q duplicates app_groups[%d]", index, group, previousIndex))
			continue
		}
		seenGroups[group] = index

		property := templatepkg.AppGroupSwiftIdentifier(cfg.BundleID, group)
		if property == "main" && group != "group."+strings.TrimSpace(cfg.BundleID) {
			issues = append(issues, fmt.Sprintf("app_groups[%d] %q cannot produce a specific AppGroups key", index, group))
		}
		if previousGroup, ok := seenProperties[property]; ok {
			issues = append(issues, fmt.Sprintf("app_groups[%d] %q conflicts with %q: both map to %s key %q", index, group, previousGroup, appGroupsInfoPlistKey, property))
		} else {
			seenProperties[property] = group
		}
	}

	if len(issues) == 0 {
		return nil
	}

	return fmt.Errorf("invalid app_groups config: %s", strings.Join(issues, "; "))
}
