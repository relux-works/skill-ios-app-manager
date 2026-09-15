package scaffold

import (
	"github.com/relux-works/ios-app-manager/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeMacOSConvergenceAndSourcePreservation(t *testing.T) {
	root := t.TempDir()
	c := config.ProjectConfig{AppName: "Example", BundleID: "com.example.app", TeamID: "TESTTEAM", SwiftVersion: "6.0", MinTarget: "14.0", MarketingVersion: "1.0", ProjectVersion: "1", MacOS: &config.MacOSAppConfig{MenuBar: true}}
	input := GenerateInput{ProjectRoot: root, Config: c}
	if _, err := runGenerateMacOSApp(input); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "Targets/Example/Sources/App.swift")
	os.WriteFile(source, []byte("// custom app\n"), 0644)
	input.Config.BundleID = "com.example.changed"
	if _, err := runGenerateMacOSApp(input); err != nil {
		t.Fatal(err)
	}
	p, _ := os.ReadFile(filepath.Join(root, "Project.swift"))
	s := string(p)
	for _, want := range []string{"com.example.changed", "destinations: [.mac]", "LSUIElement", ".unitTests", "SWIFT_STRICT_CONCURRENCY"} {
		if !strings.Contains(s, want) {
			t.Fatal(want)
		}
	}
	if strings.Contains(s, ".uiTests") || strings.Contains(s, ".iOS") {
		t.Fatal("unexpected iOS/UI test target")
	}
	b, _ := os.ReadFile(source)
	if string(b) != "// custom app\n" {
		t.Fatal("source overwritten")
	}
	if _, err := runGenerateMacOSApp(input); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(filepath.Join(root, "Project.swift"))
	if string(again) != s {
		t.Fatal("not idempotent")
	}
}
func TestNativeMacOSRejectsForeignManifest(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "Project.swift"), []byte("// iOS project"), 0644)
	_, err := runGenerateMacOSApp(GenerateInput{ProjectRoot: root, Config: config.ProjectConfig{AppName: "Example", MinTarget: "14.0", MacOS: &config.MacOSAppConfig{}}})
	if err == nil {
		t.Fatal("must preserve foreign manifest")
	}
	if _, err := os.Stat(filepath.Join(root, "Workspace.swift")); !os.IsNotExist(err) {
		t.Fatal("partial scaffold")
	}
}
