package scaffold

import (
	"fmt"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

type RuntimeProfileInput struct {
	ProjectRoot string
	Config      config.ProjectConfig
}

type RuntimeProfilePluginResult struct {
	Name    string
	Enabled bool
	Scanned []string
	Updated []string
	Message string
}

type RuntimeProfilePlugin struct {
	Name         string
	Short        string
	Dependencies []string
	Sync         func(RuntimeProfileInput) (RuntimeProfilePluginResult, error)
}

var runtimeProfilePluginRegistry = make(map[string]*RuntimeProfilePlugin)

func init() {
	RegisterRuntimeProfilePlugin(&RuntimeProfilePlugin{
		Name:  "runtime-profile-schema",
		Short: "Validate the typed runtime-profile schema and policy matrix",
		Sync: func(input RuntimeProfileInput) (RuntimeProfilePluginResult, error) {
			enabled := input.Config.HasRuntimeProfiles()
			message := "runtime profiles disabled"
			if enabled {
				message = "validated typed runtime-profile schema and policy matrix"
			}
			return RuntimeProfilePluginResult{
				Name:    "runtime-profile-schema",
				Enabled: enabled,
				Message: message,
			}, nil
		},
	})
}

func RegisterRuntimeProfilePlugin(plugin *RuntimeProfilePlugin) {
	if plugin == nil {
		panic("runtime profile plugin is nil")
	}
	name := strings.TrimSpace(plugin.Name)
	if name == "" {
		panic("runtime profile plugin name is required")
	}
	if strings.TrimSpace(plugin.Short) == "" {
		panic(fmt.Sprintf("runtime profile plugin %q short description is required", name))
	}
	if plugin.Sync == nil {
		panic(fmt.Sprintf("runtime profile plugin %q sync func is required", name))
	}
	if _, exists := runtimeProfilePluginRegistry[name]; exists {
		panic(fmt.Sprintf("duplicate runtime profile plugin %q", name))
	}
	plugin.Name = name
	runtimeProfilePluginRegistry[name] = plugin
}

func AllRuntimeProfilePlugins() []*RuntimeProfilePlugin {
	plugins := make([]*RuntimeProfilePlugin, 0, len(runtimeProfilePluginRegistry))
	for _, plugin := range runtimeProfilePluginRegistry {
		plugins = append(plugins, plugin)
	}
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})
	return plugins
}

func runtimeProfilePluginsInDependencyOrder() ([]*RuntimeProfilePlugin, error) {
	plugins := AllRuntimeProfilePlugins()
	byName := make(map[string]*RuntimeProfilePlugin, len(plugins))
	for _, plugin := range plugins {
		byName[plugin.Name] = plugin
	}

	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(plugins))
	ordered := make([]*RuntimeProfilePlugin, 0, len(plugins))
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case visiting:
			return fmt.Errorf("runtime profile plugin dependency cycle includes %q", name)
		case visited:
			return nil
		}
		plugin, ok := byName[name]
		if !ok {
			return fmt.Errorf("runtime profile plugin dependency %q is not registered", name)
		}
		state[name] = visiting
		dependencies := append([]string(nil), plugin.Dependencies...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return fmt.Errorf("%s: %w", plugin.Name, err)
			}
		}
		state[name] = visited
		ordered = append(ordered, plugin)
		return nil
	}
	for _, plugin := range plugins {
		if err := visit(plugin.Name); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

type RuntimeProfilesSyncResult struct {
	Plugins []RuntimeProfilePluginResult
	Scanned []string
	Updated []string
}

func SyncRuntimeProfiles(projectRoot string, cfg config.ProjectConfig) (RuntimeProfilesSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return RuntimeProfilesSyncResult{}, fmt.Errorf("project root is required")
	}
	if err := cfg.Validate(); err != nil {
		return RuntimeProfilesSyncResult{}, fmt.Errorf("invalid runtime profile config: %w", err)
	}

	plugins, err := runtimeProfilePluginsInDependencyOrder()
	if err != nil {
		return RuntimeProfilesSyncResult{}, err
	}
	result := RuntimeProfilesSyncResult{
		Plugins: make([]RuntimeProfilePluginResult, 0, len(plugins)),
	}
	for _, plugin := range plugins {
		pluginResult, err := plugin.Sync(RuntimeProfileInput{
			ProjectRoot: root,
			Config:      cfg,
		})
		if err != nil {
			return result, fmt.Errorf("%s: %w", plugin.Name, err)
		}
		if pluginResult.Name == "" {
			pluginResult.Name = plugin.Name
		}
		result.Plugins = append(result.Plugins, pluginResult)
		result.Scanned = appendUniqueStrings(result.Scanned, pluginResult.Scanned...)
		result.Updated = appendUniqueStrings(result.Updated, pluginResult.Updated...)
	}

	return result, nil
}
