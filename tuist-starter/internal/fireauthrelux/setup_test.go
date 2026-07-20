package fireauthrelux

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

func TestSetupConvergesMatureProjectAndPreservesCustomComposition(t *testing.T) {
	projectRoot := prepareMatureFireAuthProject(t)
	cfg := matureFireAuthConfig(t)
	configureFirebaseInputs(t, cfg, "protected-test-api-key")

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "MatureApp",
		ModulesPath: "Packages",
		Config:      cfg,
	}
	if err := Setup(input); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	first := snapshotProjectTree(t, projectRoot)
	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}
	second := snapshotProjectTree(t, projectRoot)
	if second != first {
		t.Fatalf("FireAuthRelux setup is not byte-idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	registry := readFireAuthTestFile(t, filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "App", "MatureApp.Registry.swift",
	))
	for _, preserved := range []string{
		"private(set) static var runtimeMode",
		"case hostedTests",
		"disabledForHostedTests()",
		"ioc.register(AppCore.Runtime.self",
		"ioc.register(MatureFeature.Persistence.self",
		"ioc.register(MatureFeature.SyncEngine.self",
		"ioc.register(MatureFeature.APIClient.self",
		"preconditionFailure(\"Unregistered mature dependency",
		"let module = await MatureFeature.Module(",
		"return relux.register(module)",
		"func buildAppConfigManager()",
		"func buildTokenProvider()",
	} {
		if !strings.Contains(registry, preserved) {
			t.Fatalf("setup lost mature Registry composition %q:\n%s", preserved, registry)
		}
	}
	for _, integrated := range []string{
		"import FireAuthRelux",
		"// ios-app-manager:fireauth-relux-registration:begin",
		"FireAuthRelux.Module.Interface.self",
		"// ios-app-manager:fireauth-relux-builder:begin",
		"configureFireAuthReluxFromProcess",
		"installFireAuthReluxModuleFactoryForTesting",
		"resetFireAuthReluxModuleFactoryForTesting",
		`// ios-app-manager:fireauth-relux-relux-registration:begin original="buildRelux"`,
		"resolver: Self.buildReluxWithFireAuthRelux",
		"let relux = await buildRelux()",
		"return relux.register(resolve(FireAuthRelux.Module.Interface.self))",
	} {
		if !strings.Contains(registry, integrated) {
			t.Fatalf("setup missing Registry integration %q:\n%s", integrated, registry)
		}
	}

	appBootstrap := readFireAuthTestFile(t, filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "App.swift",
	))
	for _, want := range []string{
		"CustomRuntime.prepare()",
		"// ios-app-manager:fireauth-relux-bootstrap:begin",
		"Registry.configureFireAuthReluxFromProcess()",
		"Registry.configure(runtimeMode: .application)",
	} {
		if !strings.Contains(appBootstrap, want) {
			t.Fatalf("App.swift missing or lost %q:\n%s", want, appBootstrap)
		}
	}
	processCall := strings.Index(appBootstrap, "Registry.configureFireAuthReluxFromProcess()")
	registryCall := strings.Index(appBootstrap, "Registry.configure(runtimeMode:")
	if processCall < 0 || registryCall < 0 || processCall >= registryCall {
		t.Fatalf("FireAuth process selection must run before Registry.configure(...):\n%s", appBootstrap)
	}

	packageManifest := readFireAuthTestFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`.package(name: "FireAuthRelux", url: "https://github.com/relux-works/FireAuthRelux.git", .exact("1.2.1"))`,
		`.package(name: "FireAuthKit", url: "https://github.com/relux-works/FireAuthKit.git", .exact("1.1.0"))`,
		`"FireAuthRelux": .framework`,
		`"FireAuthKit": .framework`,
		`"FireAuthProvider": .framework`,
		`"CUSTOM_SETTING": "preserved"`,
		`"IPHONEOS_DEPLOYMENT_TARGET": "17.0"`,
	} {
		if !strings.Contains(packageManifest, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, packageManifest)
		}
	}
	if strings.Contains(packageManifest, `.exact("1.0.0")`) ||
		strings.Contains(packageManifest, `"IPHONEOS_DEPLOYMENT_TARGET": "18.0"`) {
		t.Fatalf("Package.swift retained stale FireAuth values:\n%s", packageManifest)
	}

	projectManifest := readFireAuthTestFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`.external(name: "FireAuthRelux")`,
		`.external(name: "FireAuthKit")`,
		`.external(name: "FireAuthProvider")`,
		`.launchArgument(name: "--mature-hosted-tests", isEnabled: true)`,
		`.testableTarget(target: .target("MatureAppTests"))`,
		`.testableTarget(target: .target("MatureAppUITests"))`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing or lost %q:\n%s", want, projectManifest)
		}
	}

	loader := readFireAuthTestFile(t, filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "Configuration", "FireAuth", generatedFileName,
	))
	for _, want := range []string{
		"typealias ModuleFactory",
		"static func liveModule(",
		"static func deterministicModule(",
		"DeterministicTransport",
		"throw DeterministicTransportError.networkDisabled",
		"static func moduleFactory(",
		"for descriptor: BackendEnvironmentDescriptor",
		"FireAuthProvider.Configuration.load(",
		"googleServiceInfoResource: resourceName",
		"resolved.firebaseProjectID == firebase.projectID",
		"resolved.googleAppID == firebase.googleAppID",
		"resolved.bundleID == firebase.bundleID",
		"Configuration(status: .missing([.firebaseAPIKey]))",
	} {
		if !strings.Contains(loader, want) {
			t.Fatalf("generated loader missing %q:\n%s", want, loader)
		}
	}

	processSelection := readFireAuthTestFile(t, filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "Configuration", "FireAuth", generatedProcessFileName,
	))
	for _, want := range []string{
		"enum GeneratedFireAuthReluxProcess",
		"case live",
		"case deterministic",
		`selectionEnvironmentKey = "FIREAUTH_RELUX_PROCESS_MODE"`,
		`deterministicLaunchArgument = "--fireauth-relux-deterministic"`,
		"ProcessInfo.processInfo.arguments",
		"ProcessInfo.processInfo.environment",
		"conflictingSelections",
	} {
		if !strings.Contains(processSelection, want) {
			t.Fatalf("generated process selection missing %q:\n%s", want, processSelection)
		}
	}

	for _, target := range cfg.RuntimeProfiles.TestAction.Targets {
		testLaunch := readFireAuthTestFile(t, filepath.Join(
			projectRoot,
			"Targets", target, "Sources", "Support", generatedTestLaunchFileName,
		))
		for _, want := range []string{
			"enum GeneratedFireAuthReluxTestLaunch",
			"deterministicLaunchArguments",
			`"--fireauth-relux-deterministic"`,
			"deterministicLaunchEnvironment",
			`"FIREAUTH_RELUX_PROCESS_MODE": "deterministic"`,
		} {
			if !strings.Contains(testLaunch, want) {
				t.Fatalf("generated test launch helper for %s missing %q:\n%s", target, want, testLaunch)
			}
		}
	}
	if strings.Contains(first, "protected-test-api-key") {
		t.Fatal("setup retained protected Firebase input in the project tree")
	}
}

func TestGeneratedProcessSelectionCrossesApplicationProcessBoundary(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("generated Swift process-boundary regression requires macOS")
	}
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("generated Swift process-boundary regression requires xcrun")
	}

	projectRoot := prepareMatureFireAuthProject(t)
	cfg := matureFireAuthConfig(t)
	configureFirebaseInputs(t, cfg, "protected-process-test-key")
	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "MatureApp",
		ModulesPath: "Packages",
		Config:      cfg,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	processSource := filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "Configuration", "FireAuth", generatedProcessFileName,
	)
	probeRoot := t.TempDir()
	probeSource := filepath.Join(probeRoot, "ProcessSelectionProbe.swift")
	writeFireAuthTestFile(t, probeSource, `import Foundation

@main
struct ProcessSelectionProbe {
    static func main() {
        do {
            print(try GeneratedFireAuthReluxProcess.selection().rawValue)
        } catch {
            fatalError("selection-error: \(error)")
        }
    }
}
`)
	probeBinary := filepath.Join(probeRoot, "process-selection-probe")
	compile := exec.Command("xcrun", "swiftc", processSource, probeSource, "-o", probeBinary)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile generated process selection error = %v\n%s", err, output)
	}

	assertFireAuthProcessSelection(t, probeBinary, nil, nil, "live")
	deterministic := "deterministic"
	assertFireAuthProcessSelection(t, probeBinary, nil, &deterministic, "deterministic")
	assertFireAuthProcessSelection(
		t,
		probeBinary,
		[]string{"--fireauth-relux-deterministic"},
		nil,
		"deterministic",
	)

	live := "live"
	output, err := runFireAuthProcessSelectionProbe(
		probeBinary,
		[]string{"--fireauth-relux-deterministic"},
		&live,
	)
	if err == nil || !strings.Contains(output, "conflicts") {
		t.Fatalf("conflicting process selection output = %q, error = %v", output, err)
	}
	unknown := "unexpected"
	output, err = runFireAuthProcessSelectionProbe(probeBinary, nil, &unknown)
	if err == nil || !strings.Contains(output, "must be live or deterministic") {
		t.Fatalf("unknown process selection output = %q, error = %v", output, err)
	}
}

func TestMatureXCUITestFixtureUsesGeneratedProcessLaunchConfiguration(t *testing.T) {
	t.Parallel()

	payload := readFireAuthTestFile(t, filepath.Join(
		"..", "..", "testdata", "mature-fireauth", "FireAuthProcessLaunchTests.swift",
	))
	for _, want := range []string{
		"XCUIApplication()",
		"GeneratedFireAuthReluxTestLaunch.deterministicLaunchArguments",
		"app.launch()",
		"app.wait(for: .runningForeground",
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("mature XCUITest process fixture missing %q:\n%s", want, payload)
		}
	}
	for _, rawSelectionKey := range []string{
		processSelectionEnvironmentKey,
		deterministicLaunchArgument,
	} {
		if strings.Contains(payload, rawSelectionKey) {
			t.Fatalf("mature XCUITest fixture duplicates generated launch key %q:\n%s", rawSelectionKey, payload)
		}
	}
}

func TestSetupRejectsCustomProcessOrTestLaunchOutputBeforeMutation(t *testing.T) {
	for name, relativePath := range map[string]string{
		"process selection": filepath.Join(
			"Targets", "MatureApp", "Sources", "Configuration", "FireAuth", generatedProcessFileName,
		),
		"test launch helper": filepath.Join(
			"Targets", "MatureAppTests", "Sources", "Support", generatedTestLaunchFileName,
		),
	} {
		t.Run(name, func(t *testing.T) {
			projectRoot := prepareMatureFireAuthProject(t)
			cfg := matureFireAuthConfig(t)
			configureFirebaseInputs(t, cfg, "protected-custom-output-key")
			writeFireAuthTestFile(t, filepath.Join(projectRoot, relativePath), "// user-owned\n")
			before := snapshotProjectTree(t, projectRoot)

			err := Setup(SetupInput{
				ProjectRoot: projectRoot,
				AppName:     "MatureApp",
				ModulesPath: "Packages",
				Config:      cfg,
			})
			if err == nil || !strings.Contains(err.Error(), "refusing to replace custom") {
				t.Fatalf("Setup() error = %v, want custom generated-output refusal", err)
			}
			after := snapshotProjectTree(t, projectRoot)
			if after != before {
				t.Fatal("custom generated-output failure mutated the mature project")
			}
		})
	}
}

func TestSetupRejectsMissingRuntimeProfilesBeforeMutation(t *testing.T) {
	projectRoot := prepareMatureFireAuthProject(t)
	cfg := matureFireAuthConfig(t)
	cfg.RuntimeProfiles = nil
	before := snapshotProjectTree(t, projectRoot)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "MatureApp",
		ModulesPath: "Packages",
		Config:      cfg,
	})
	if err == nil || !strings.Contains(err.Error(), "runtime profiles are required") {
		t.Fatalf("Setup() error = %v, want explicit runtime-profile requirement", err)
	}
	after := snapshotProjectTree(t, projectRoot)
	if after != before {
		t.Fatal("failed setup mutated the mature project")
	}
}

func TestSetupRejectsIncompleteFirebaseConfigAndDoesNotRetainInput(t *testing.T) {
	projectRoot := prepareMatureFireAuthProject(t)
	cfg := matureFireAuthConfig(t)
	production := cfg.RuntimeProfiles.BackendEnvironments[config.BackendEnvironmentProduction]
	production.Firebase = nil
	cfg.RuntimeProfiles.BackendEnvironments[config.BackendEnvironmentProduction] = production
	configureFirebaseInputs(t, cfg, "protected-incomplete-key")
	before := snapshotProjectTree(t, projectRoot)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "MatureApp",
		ModulesPath: "Packages",
		Config:      cfg,
	})
	if err == nil || !strings.Contains(err.Error(), "Firebase is required outside fixture") {
		t.Fatalf("Setup() error = %v, want incomplete Firebase error", err)
	}
	after := snapshotProjectTree(t, projectRoot)
	if after != before {
		t.Fatal("incomplete Firebase config mutated the mature project")
	}
	if strings.Contains(after, "protected-incomplete-key") {
		t.Fatal("failed setup retained protected Firebase input")
	}
}

func TestSetupRejectsMissingFirebaseValidationHookWithoutReportingPathOrMutating(t *testing.T) {
	projectRoot := prepareMatureFireAuthProject(t)
	cfg := matureFireAuthConfig(t)
	before := snapshotProjectTree(t, projectRoot)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "MatureApp",
		ModulesPath: "Packages",
		Config:      cfg,
	})
	if err == nil || !strings.Contains(err.Error(), "Firebase validation hook") {
		t.Fatalf("Setup() error = %v, want missing validation hook", err)
	}
	after := snapshotProjectTree(t, projectRoot)
	if after != before {
		t.Fatal("missing Firebase validation hook mutated the mature project")
	}
}

func prepareMatureFireAuthProject(t *testing.T) string {
	t.Helper()
	projectRoot := t.TempDir()
	fixtureRoot := filepath.Join("..", "..", "testdata", "mature-fireauth")

	for _, name := range []string{"Package.swift", "Project.swift", "App.swift"} {
		payload, err := os.ReadFile(filepath.Join(fixtureRoot, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		destination := filepath.Join(projectRoot, name)
		if name == "App.swift" {
			destination = filepath.Join(projectRoot, "Targets", "MatureApp", "Sources", name)
		}
		writeFireAuthTestFile(t, destination, string(payload))
	}
	registry, err := os.ReadFile(filepath.Join(fixtureRoot, "Registry.swift"))
	if err != nil {
		t.Fatalf("ReadFile(Registry.swift) error = %v", err)
	}
	writeFireAuthTestFile(t, filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "App", "MatureApp.Registry.swift",
	), string(registry))

	cfg := matureFireAuthConfig(t)
	writeFireAuthTestFile(t, filepath.Join(
		projectRoot,
		"Targets", "MatureApp", "Sources", "Configuration", "Runtime", "RuntimeProfiles.swift",
	), scaffold.GenerateRuntimeProfilesSwift(cfg))
	for _, target := range cfg.RuntimeProfiles.TestAction.Targets {
		if err := os.MkdirAll(filepath.Join(projectRoot, "Targets", target, "Sources"), 0o755); err != nil {
			t.Fatalf("MkdirAll(test target %s) error = %v", target, err)
		}
	}
	return projectRoot
}

func matureFireAuthConfig(t *testing.T) config.ProjectConfig {
	t.Helper()
	cfg, err := config.LoadConfig(filepath.Join("..", "..", "testdata", "runtime-profiles-config.json"))
	if err != nil {
		t.Fatalf("LoadConfig(runtime-profiles-config.json) error = %v", err)
	}
	cfg.AppName = "MatureApp"
	cfg.ProductName = "MatureApp"
	cfg.RuntimeProfiles.TestAction.Targets = []string{"MatureAppTests", "MatureAppUITests"}
	cfg.RuntimeProfiles.TestAction.LaunchArguments = []string{"--mature-hosted-tests"}
	return cfg
}

func configureFirebaseInputs(t *testing.T, cfg config.ProjectConfig, apiKey string) {
	t.Helper()
	inputRoot := t.TempDir()
	written := make(map[string]struct{})
	for _, environment := range cfg.OrderedBackendEnvironments() {
		descriptor := cfg.RuntimeProfiles.BackendEnvironments[environment]
		if descriptor.Firebase == nil {
			continue
		}
		firebase := descriptor.Firebase
		if _, exists := written[firebase.ValidationInputEnvironmentVar]; exists {
			continue
		}
		written[firebase.ValidationInputEnvironmentVar] = struct{}{}
		path := filepath.Join(inputRoot, string(environment)+".plist")
		writeFireAuthTestFile(t, path, firebaseTestPlist(
			firebase.ProjectID,
			firebase.GoogleAppID,
			firebase.BundleID,
			apiKey,
		))
		t.Setenv(firebase.ValidationInputEnvironmentVar, path)
	}
}

func firebaseTestPlist(projectID, googleAppID, bundleID, apiKey string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PROJECT_ID</key><string>%s</string>
<key>GOOGLE_APP_ID</key><string>%s</string>
<key>BUNDLE_ID</key><string>%s</string>
<key>API_KEY</key><string>%s</string>
</dict></plist>
`, projectID, googleAppID, bundleID, apiKey)
}

func snapshotProjectTree(t *testing.T, root string) string {
	t.Helper()
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, relative)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s) error = %v", root, err)
	}
	sort.Strings(paths)
	var snapshot strings.Builder
	for _, relative := range paths {
		snapshot.WriteString("== " + filepath.ToSlash(relative) + " ==\n")
		snapshot.WriteString(readFireAuthTestFile(t, filepath.Join(root, relative)))
		snapshot.WriteString("\n")
	}
	return snapshot.String()
}

func writeFireAuthTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func readFireAuthTestFile(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(payload)
}

func assertFireAuthProcessSelection(
	t *testing.T,
	probeBinary string,
	arguments []string,
	environmentValue *string,
	want string,
) {
	t.Helper()
	output, err := runFireAuthProcessSelectionProbe(probeBinary, arguments, environmentValue)
	if err != nil {
		t.Fatalf("process selection probe error = %v\n%s", err, output)
	}
	if got := strings.TrimSpace(output); got != want {
		t.Fatalf("process selection = %q, want %q", got, want)
	}
}

func runFireAuthProcessSelectionProbe(
	probeBinary string,
	arguments []string,
	environmentValue *string,
) (string, error) {
	command := exec.Command(probeBinary, arguments...)
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, processSelectionEnvironmentKey+"=") {
			environment = append(environment, entry)
		}
	}
	if environmentValue != nil {
		environment = append(environment, processSelectionEnvironmentKey+"="+*environmentValue)
	}
	command.Env = environment
	output, err := command.CombinedOutput()
	return string(output), err
}
