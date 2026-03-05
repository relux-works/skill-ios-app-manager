package blueprint

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var pascalCasePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// Blueprint is the top-level config for module scaffolding from a JSON file.
type Blueprint struct {
	Name string      `json:"name"`
	Type string      `json:"type"`
	Data *DataConfig `json:"data,omitempty"`
	UI   *UIConfig   `json:"ui,omitempty"`
}

// DataConfig describes which data layers to generate.
type DataConfig struct {
	HTTP  bool         `json:"http"`
	WS    bool         `json:"ws"`
	Local *LocalConfig `json:"local,omitempty"`
}

// LocalConfig describes local data sources.
type LocalConfig struct {
	FileManager bool `json:"fileManager"`
	DB          bool `json:"db"`
}

// UIConfig describes which UI layers to generate.
type UIConfig struct {
	EntryPoint string   `json:"entryPoint,omitempty"`
	Features   []string `json:"features"`
	Components []string `json:"components"`
}

// HasHTTP returns true if HTTP data layer should be generated.
func (b *Blueprint) HasHTTP() bool {
	return b.Data != nil && b.Data.HTTP
}

// HasWS returns true if WebSocket data layer should be generated.
func (b *Blueprint) HasWS() bool {
	return b.Data != nil && b.Data.WS
}

// HasLocal returns true if any local data layer should be generated.
func (b *Blueprint) HasLocal() bool {
	return b.Data != nil && b.Data.Local != nil
}

// HasUI returns true if any UI layer should be generated.
func (b *Blueprint) HasUI() bool {
	return b.UI != nil && (len(b.UI.Features) > 0 || len(b.UI.Components) > 0)
}

// EntryPoint returns the entry point feature name.
// Defaults to the first feature if not explicitly set.
func (b *Blueprint) EntryPoint() string {
	if b.UI == nil || len(b.UI.Features) == 0 {
		return ""
	}
	if b.UI.EntryPoint != "" {
		return b.UI.EntryPoint
	}
	return b.UI.Features[0]
}

// IsEntryPoint returns true if the given feature name is the entry point.
func (b *Blueprint) IsEntryPoint(featureName string) bool {
	return b.EntryPoint() == featureName
}

// HasFeatures returns true if feature pages should be generated.
func (b *Blueprint) HasFeatures() bool {
	return b.UI != nil && len(b.UI.Features) > 0
}

// HasComponents returns true if shared UI components should be generated.
func (b *Blueprint) HasComponents() bool {
	return b.UI != nil && len(b.UI.Components) > 0
}

// Features returns the list of UI features, or nil.
func (b *Blueprint) Features() []Feature {
	if b.UI == nil {
		return nil
	}
	out := make([]Feature, len(b.UI.Features))
	for i, name := range b.UI.Features {
		out[i] = Feature{
			Name:      name,
			NameLower: lowerFirst(name),
		}
	}
	return out
}

// Components returns the list of UI components, or nil.
func (b *Blueprint) Components() []Component {
	if b.UI == nil {
		return nil
	}
	out := make([]Component, len(b.UI.Components))
	for i, name := range b.UI.Components {
		out[i] = Component{
			Name:      name,
			NameLower: lowerFirst(name),
		}
	}
	return out
}

// Feature describes a named UI feature (Container + Page + Props).
type Feature struct {
	Name      string // PascalCase, e.g. "Login"
	NameLower string // camelCase, e.g. "login"
}

// Component describes a named shared UI component.
type Component struct {
	Name      string // PascalCase, e.g. "PasswordField"
	NameLower string // camelCase, e.g. "passwordField"
}

// ParseFile reads and parses a blueprint JSON file.
func ParseFile(path string) (*Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read blueprint file %q: %w", path, err)
	}
	return Parse(data)
}

// Parse parses blueprint JSON bytes.
func Parse(data []byte) (*Blueprint, error) {
	var bp Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("parse blueprint JSON: %w", err)
	}
	return &bp, nil
}

// Validate checks that a Blueprint is well-formed.
func (b *Blueprint) Validate() error {
	if strings.TrimSpace(b.Name) == "" {
		return fmt.Errorf("blueprint: name is required")
	}
	if !pascalCasePattern.MatchString(b.Name) {
		return fmt.Errorf("blueprint: name %q must be PascalCase", b.Name)
	}

	if strings.TrimSpace(b.Type) == "" {
		return fmt.Errorf("blueprint: type is required")
	}
	if b.Type != "relux-feature" {
		return fmt.Errorf("blueprint: unsupported type %q (only \"relux-feature\" is supported)", b.Type)
	}

	if b.UI != nil {
		for _, name := range b.UI.Features {
			if !pascalCasePattern.MatchString(name) {
				return fmt.Errorf("blueprint: feature name %q must be PascalCase", name)
			}
		}
		for _, name := range b.UI.Components {
			if !pascalCasePattern.MatchString(name) {
				return fmt.Errorf("blueprint: component name %q must be PascalCase", name)
			}
		}
		if b.UI.EntryPoint != "" && len(b.UI.Features) > 0 {
			found := false
			for _, f := range b.UI.Features {
				if f == b.UI.EntryPoint {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("blueprint: entryPoint %q must be one of features %v", b.UI.EntryPoint, b.UI.Features)
			}
		}
	}

	return nil
}

// DefaultBlueprint returns a Blueprint with all layers enabled as a starting point.
func DefaultBlueprint(name string) *Blueprint {
	return &Blueprint{
		Name: name,
		Type: "relux-feature",
		Data: &DataConfig{
			HTTP:  true,
			WS:    false,
			Local: nil,
		},
		UI: &UIConfig{
			EntryPoint: "Main",
			Features:   []string{"Main"},
			Components: nil,
		},
	}
}

// ToJSON serializes a Blueprint to pretty-printed JSON.
func (b *Blueprint) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
