package scaffold

import (
	"fmt"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// GenerateInput is the normalized context passed to scaffold generators.
type GenerateInput struct {
	ConfigPath  string
	ProjectRoot string
	Config      config.ProjectConfig
}

// GenerateResult is the user-facing outcome of a scaffold generator run.
type GenerateResult struct {
	Message string
}

// GeneratorPlugin describes one pluggable `generate` artifact.
type GeneratorPlugin struct {
	Name         string
	Short        string
	Dependencies []string
	Run          func(GenerateInput) (GenerateResult, error)
}

var generatorRegistry = make(map[string]*GeneratorPlugin)

// RegisterGenerator registers a pluggable generator artifact.
func RegisterGenerator(plugin *GeneratorPlugin) {
	if plugin == nil {
		panic("scaffold generator plugin is nil")
	}

	name := strings.TrimSpace(plugin.Name)
	if name == "" {
		panic("scaffold generator plugin name is required")
	}
	if strings.TrimSpace(plugin.Short) == "" {
		panic(fmt.Sprintf("scaffold generator plugin %q short description is required", name))
	}
	if plugin.Run == nil {
		panic(fmt.Sprintf("scaffold generator plugin %q run func is required", name))
	}
	if _, exists := generatorRegistry[name]; exists {
		panic(fmt.Sprintf("duplicate scaffold generator plugin %q", name))
	}

	plugin.Name = name
	generatorRegistry[name] = plugin
}

// AllGenerators returns the registered generators sorted by command name.
func AllGenerators() []*GeneratorPlugin {
	plugins := make([]*GeneratorPlugin, 0, len(generatorRegistry))
	for _, plugin := range generatorRegistry {
		plugins = append(plugins, plugin)
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins
}
