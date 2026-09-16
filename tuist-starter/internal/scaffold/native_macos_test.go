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

func TestNativeMacOSAppOnlyDependencyAndPlist(t *testing.T) {
	c := config.ProjectConfig{AppName: "Example", MinTarget: "14.0", SwiftVersion: "6.0", MacOS: &config.MacOSAppConfig{HardenedRuntime: true, InfoPlist: map[string]any{"SUFeedURL": "https://example.com/appcast.xml", "SUEnableAutomaticChecks": true}, Packages: []config.MacOSPackage{{URL: "https://example.com/update.git", Version: "1.0.0", Product: "Updater", Target: "app"}}}}
	root := t.TempDir()
	input := GenerateInput{ProjectRoot: root, Config: c}
	if _, err := runGenerateMacOSApp(input); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "Project.swift"))
	p := string(b)
	for _, want := range []string{`"SUFeedURL": .string("https://example.com/appcast.xml")`, `"SUEnableAutomaticChecks": .boolean(true)`, `"ENABLE_HARDENED_RUNTIME": "YES"`, `dependencies: [.target(name: "ExampleCore"), .package(product: "Updater")]`} {
		if !strings.Contains(p, want) {
			t.Fatal(want)
		}
	}
	manifest, _ := os.ReadFile(filepath.Join(root, "Packages/ExampleCore/Package.swift"))
	if strings.Contains(string(manifest), "Updater") || strings.Contains(string(manifest), "update.git") {
		t.Fatal("app dependency leaked into Core")
	}
	delete(c.MacOS.InfoPlist, "SUEnableAutomaticChecks")
	if _, err := runGenerateMacOSApp(input); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(root, "Project.swift"))
	if strings.Contains(string(b), "SUEnableAutomaticChecks") {
		t.Fatal("removed setting retained")
	}
	c.MacOS.InfoPlist["CFBundleIdentifier"] = "invalid"
	if _, err := runGenerateMacOSApp(input); err == nil {
		t.Fatal("reserved identity must be rejected")
	}
}
