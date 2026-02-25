package scaffold

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

type peripheryConfigYAML struct {
	Workspace      string
	Schemes        []string
	RetainPublic   bool
	BuildArguments []string
}

func TestGeneratePeripheryConfigHasWorkspaceSchemeAndRetainPublic(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:   "DemoApp",
		MinTarget: "17.0",
	}

	parsed := decodePeripheryConfig(t, GeneratePeripheryConfig(cfg))

	if parsed.Workspace != "DemoApp.xcworkspace" {
		t.Fatalf("workspace = %q, want %q", parsed.Workspace, "DemoApp.xcworkspace")
	}
	if !reflect.DeepEqual(parsed.Schemes, []string{"DemoApp"}) {
		t.Fatalf("schemes = %#v, want %#v", parsed.Schemes, []string{"DemoApp"})
	}
	if !parsed.RetainPublic {
		t.Fatal("retain_public = false, want true")
	}

	wantBuildArgs := []string{
		"-destination",
		"platform=iOS Simulator,name=iPhone 16,OS=17.0",
		"-derivedDataPath",
		"./DerivedData/Periphery",
	}
	if !reflect.DeepEqual(parsed.BuildArguments, wantBuildArgs) {
		t.Fatalf("build_arguments = %#v, want %#v", parsed.BuildArguments, wantBuildArgs)
	}
}

func TestGeneratePeripheryConfigUsesDefaults(t *testing.T) {
	t.Parallel()

	parsed := decodePeripheryConfig(t, GeneratePeripheryConfig(config.ProjectConfig{}))

	if parsed.Workspace != "App.xcworkspace" {
		t.Fatalf("workspace = %q, want %q", parsed.Workspace, "App.xcworkspace")
	}
	if !reflect.DeepEqual(parsed.Schemes, []string{"App"}) {
		t.Fatalf("schemes = %#v, want %#v", parsed.Schemes, []string{"App"})
	}

	wantDestination := "platform=iOS Simulator,name=iPhone 16,OS=latest"
	if len(parsed.BuildArguments) < 2 || parsed.BuildArguments[1] != wantDestination {
		t.Fatalf("destination build argument = %#v, want %q", parsed.BuildArguments, wantDestination)
	}
}

func decodePeripheryConfig(t *testing.T, content string) peripheryConfigYAML {
	t.Helper()

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	var parsed peripheryConfigYAML
	currentListKey := ""
	for index, raw := range lines {
		lineNumber := index + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(raw, "  - ") {
			itemRaw := strings.TrimSpace(strings.TrimPrefix(raw, "  - "))
			item, err := parseYAMLDoubleQuotedString(itemRaw)
			if err != nil {
				t.Fatalf("invalid YAML list item at line %d: %v\ncontent:\n%s", lineNumber, err, content)
			}

			switch currentListKey {
			case "schemes":
				parsed.Schemes = append(parsed.Schemes, item)
			case "build_arguments":
				parsed.BuildArguments = append(parsed.BuildArguments, item)
			default:
				t.Fatalf("unexpected YAML list item at line %d:\n%s", lineNumber, content)
			}
			continue
		}

		if strings.HasPrefix(raw, " ") {
			t.Fatalf("unexpected YAML indentation at line %d:\n%s", lineNumber, content)
		}

		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			t.Fatalf("invalid YAML mapping at line %d:\n%s", lineNumber, content)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "workspace":
			workspace, err := parseYAMLDoubleQuotedString(value)
			if err != nil {
				t.Fatalf("invalid workspace value at line %d: %v\ncontent:\n%s", lineNumber, err, content)
			}
			parsed.Workspace = workspace
			currentListKey = ""
		case "schemes":
			if value != "" {
				t.Fatalf("schemes key should use block list syntax at line %d:\n%s", lineNumber, content)
			}
			currentListKey = "schemes"
		case "retain_public":
			switch value {
			case "true":
				parsed.RetainPublic = true
			case "false":
				parsed.RetainPublic = false
			default:
				t.Fatalf("retain_public must be true/false at line %d:\n%s", lineNumber, content)
			}
			currentListKey = ""
		case "build_arguments":
			if value != "" {
				t.Fatalf("build_arguments key should use block list syntax at line %d:\n%s", lineNumber, content)
			}
			currentListKey = "build_arguments"
		default:
			t.Fatalf("unexpected YAML key %q at line %d:\n%s", key, lineNumber, content)
		}
	}

	return parsed
}

func parseYAMLDoubleQuotedString(value string) (string, error) {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", fmt.Errorf("value %q must be double-quoted", value)
	}

	encoded := value[1 : len(value)-1]
	var out strings.Builder
	escaped := false
	for _, r := range encoded {
		if escaped {
			switch r {
			case '\\', '"':
				out.WriteRune(r)
			case 'n':
				out.WriteByte('\n')
			case 'r':
				out.WriteByte('\r')
			case 't':
				out.WriteByte('\t')
			default:
				return "", fmt.Errorf("unsupported escape sequence \\%c", r)
			}
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		out.WriteRune(r)
	}

	if escaped {
		return "", fmt.Errorf("unterminated escape sequence")
	}

	return out.String(), nil
}
