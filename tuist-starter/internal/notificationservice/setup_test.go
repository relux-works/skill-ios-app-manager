package notificationservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSetupValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SetupInput
		want  string
	}{
		{
			name:  "missing project root",
			input: SetupInput{AppName: "DemoApp"},
			want:  "project root is required",
		},
		{
			name:  "missing app name",
			input: SetupInput{ProjectRoot: "/tmp"},
			want:  "app name is required",
		},
		{
			name:  "invalid extension target",
			input: SetupInput{ProjectRoot: "/tmp", AppName: "DemoApp", ExtensionTarget: "1Bad"},
			want:  "extension target name",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Setup(tc.input)
			if err == nil {
				t.Fatal("Setup() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Setup() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSetupCreatesNotificationServiceExtensionAndCore(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	extensionName := "DemoAppNotificationServiceExtension"
	coreName := extensionName + "Core"
	extensionRoot := filepath.Join(projectRoot, "Extensions", extensionName)
	coreRoot := filepath.Join(extensionRoot, coreName)

	requireFile(t, filepath.Join(extensionRoot, "Project.swift"))
	requireFile(t, filepath.Join(extensionRoot, "Sources", extensionName+".swift"))
	requireFile(t, filepath.Join(coreRoot, "Package.swift"))
	requireFile(t, filepath.Join(coreRoot, "Sources", coreName+".swift"))
	requireFile(t, filepath.Join(coreRoot, "Sources", coreName+"NotificationServiceHandler.swift"))
	requireFile(t, filepath.Join(coreRoot, "Tests", coreName+"Tests", coreName+"Tests.swift"))

	projectSwift := readFile(t, filepath.Join(extensionRoot, "Project.swift"))
	for _, want := range []string{
		`bundleId: "\(hostBundleId).notification-service"`,
		`"NSExtensionPointIdentifier": .string("com.apple.usernotifications.service")`,
		`"NSExtensionPrincipalClass": .string("$(PRODUCT_MODULE_NAME).DemoAppNotificationServiceExtension")`,
		`.external(name: "DemoAppNotificationServiceExtensionCore")`,
		`.sdk(name: "UserNotifications", type: .framework)`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}

	wrapper := readFile(t, filepath.Join(extensionRoot, "Sources", extensionName+".swift"))
	for _, want := range []string{
		"import UserNotifications",
		"import DemoAppNotificationServiceExtensionCore",
		"final class DemoAppNotificationServiceExtension: UNNotificationServiceExtension",
		"private let handler = DemoAppNotificationServiceExtensionCoreNotificationServiceHandler()",
		"override func didReceive(",
		"override func serviceExtensionTimeWillExpire()",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("wrapper missing %q:\n%s", want, wrapper)
		}
	}

	handler := readFile(t, filepath.Join(coreRoot, "Sources", coreName+"NotificationServiceHandler.swift"))
	for _, want := range []string{
		"import UserNotifications",
		"public final class DemoAppNotificationServiceExtensionCoreNotificationServiceHandler",
		"public func process(",
		"request: UNNotificationRequest",
		"contentHandler(bestAttemptContent)",
		"public func serviceExtensionTimeWillExpire(",
	} {
		if !strings.Contains(handler, want) {
			t.Fatalf("handler missing %q:\n%s", want, handler)
		}
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackage, `.package(path: "Extensions/DemoAppNotificationServiceExtension/DemoAppNotificationServiceExtensionCore")`) {
		t.Fatalf("root Package.swift missing Core package dependency:\n%s", rootPackage)
	}
}

func TestSetupSupportsCustomTargetAndSuffix(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot)

	if err := Setup(SetupInput{
		ProjectRoot:     projectRoot,
		AppName:         "DemoApp",
		ExtensionTarget: "VideoCallNotificationService",
		BundleIDSuffix:  "push.service",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Extensions", "VideoCallNotificationService", "Project.swift"))
	for _, want := range []string{
		`bundleId: "\(hostBundleId).push.service"`,
		`"NSExtensionPrincipalClass": .string("$(PRODUCT_MODULE_NAME).VideoCallNotificationService")`,
		`.external(name: "VideoCallNotificationServiceCore")`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}
}

func TestSetupIsIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot)

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}
	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	extensionName := "DemoAppNotificationServiceExtension"
	coreName := extensionName + "Core"
	extensionRoot := filepath.Join(projectRoot, "Extensions", extensionName)

	projectSwift := readFile(t, filepath.Join(extensionRoot, "Project.swift"))
	for _, want := range []string{
		`.external(name: "DemoAppNotificationServiceExtensionCore")`,
		`.sdk(name: "UserNotifications", type: .framework)`,
		`"NSExtensionPrincipalClass": .string("$(PRODUCT_MODULE_NAME).DemoAppNotificationServiceExtension")`,
	} {
		if got := strings.Count(projectSwift, want); got != 1 {
			t.Fatalf("%q appears %d times, want 1:\n%s", want, got, projectSwift)
		}
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if got := strings.Count(rootPackage, `.package(path: "Extensions/DemoAppNotificationServiceExtension/DemoAppNotificationServiceExtensionCore")`); got != 1 {
		t.Fatalf("Core package dependency appears %d times, want 1:\n%s", got, rootPackage)
	}

	corePackage := readFile(t, filepath.Join(extensionRoot, coreName, "Package.swift"))
	if got := strings.Count(corePackage, `name: "DemoAppNotificationServiceExtensionCoreTests"`); got != 1 {
		t.Fatalf("Core test target appears %d times, want 1:\n%s", got, corePackage)
	}
}

func setupProjectFiles(t *testing.T, projectRoot string) {
	t.Helper()

	cfg := config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.demo.app",
		TeamID:           "TEAM123456",
		SwiftVersion:     "6.0",
		MinTarget:        "17.0",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(config) error = %v", err)
	}
	writeTestFile(t, filepath.Join(projectRoot, config.DefaultConfigPath), string(raw))

	rootPackageSwift := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoAppDependencies",
    dependencies: [
    ],
    targets: []
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Package.swift"), rootPackageSwift)
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("path %q is a directory, want file", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
