package e2e

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/cli"
	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestPipelineInitModuleCreateAndValidation(t *testing.T) {
	repoRoot := findRepoRoot(t)

	fixturePath := filepath.Join(repoRoot, "testdata", "xflow-config.json")
	cfg, err := config.LoadConfig(fixturePath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", fixturePath, err)
	}

	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	if err := config.WriteProjectConfig(configPath, cfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	initOutput, err := executeRootCommand("init", "--config", configPath, "--output", projectRoot)
	if err != nil {
		t.Fatalf("executeRootCommand(init) error = %v", err)
	}
	if !strings.Contains(initOutput, "scaffolded") {
		t.Fatalf("init output = %q, want scaffold confirmation", initOutput)
	}

	verifyScaffoldOutput(t, projectRoot, cfg)

	moduleOutput, err := executeRootCommand("--config", configPath, "module", "create", "TodoList", "--type", "feature")
	if err != nil {
		t.Fatalf("executeRootCommand(module create) error = %v", err)
	}
	if !strings.Contains(moduleOutput, `created module "TodoList" of type "feature"`) {
		t.Fatalf("module create output = %q, want create confirmation", moduleOutput)
	}

	verifyTodoListModule(t, projectRoot, cfg.ModulesPath)

	// Create a relux-feature module to verify full pipeline.
	reluxOutput, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "relux-feature")
	if err != nil {
		t.Fatalf("executeRootCommand(module create relux-feature) error = %v", err)
	}
	if !strings.Contains(reluxOutput, `created module "Auth" of type "relux-feature"`) {
		t.Fatalf("relux-feature module create output = %q, want create confirmation", reluxOutput)
	}

	verifyReluxFeatureModule(t, projectRoot, cfg.ModulesPath)

	// Set up IoC first (required for token-provider to update Registry).
	iocOutput, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes")
	if err != nil {
		t.Fatalf("executeRootCommand(ioc setup) error = %v", err)
	}
	if !strings.Contains(iocOutput, "IoC setup complete") {
		t.Fatalf("ioc setup output = %q, want completion message", iocOutput)
	}

	// Set up TokenProvider module.
	tpOutput, err := executeRootCommand("--config", configPath, "token-provider", "setup", "--yes")
	if err != nil {
		t.Fatalf("executeRootCommand(token-provider setup) error = %v", err)
	}
	if !strings.Contains(tpOutput, "TokenProvider setup complete") {
		t.Fatalf("token-provider setup output = %q, want completion message", tpOutput)
	}

	verifyTokenProviderModule(t, projectRoot, cfg.ModulesPath)

	// Set up HttpClient (IoC registration via registry).
	hcOutput, err := executeRootCommand("--config", configPath, "http-client", "setup", "--yes")
	if err != nil {
		t.Fatalf("executeRootCommand(http-client setup) error = %v", err)
	}
	if !strings.Contains(hcOutput, "HttpClient setup complete") {
		t.Fatalf("http-client setup output = %q, want completion message", hcOutput)
	}

	verifyHttpClientSetup(t, projectRoot, cfg)
	verifyNoTemplateArtifactsInSwiftFiles(t, projectRoot)
	verifyGoBuild(t, repoRoot)
}

func TestPipelineAllModuleTypesRegression(t *testing.T) {
	repoRoot := findRepoRoot(t)

	fixturePath := filepath.Join(repoRoot, "testdata", "xflow-config.json")
	cfg, err := config.LoadConfig(fixturePath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", fixturePath, err)
	}

	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	if err := config.WriteProjectConfig(configPath, cfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	_, err = executeRootCommand("init", "--config", configPath, "--output", projectRoot)
	if err != nil {
		t.Fatalf("executeRootCommand(init) error = %v", err)
	}

	modulesRoot := filepath.Join(projectRoot, cfg.ModulesPath)

	// Create one module of each type and verify generated files.
	type moduleCase struct {
		name       string
		moduleType string
		hasImpl    bool
	}
	cases := []moduleCase{
		{"Profile", "feature", true},
		{"Analytics", "kit", true},
		{"Cache", "shared", true},
		{"Theme", "ui", true},
		{"Logger", "utility", false},
		{"Payment", "relux-feature", true},
	}

	for _, mc := range cases {
		output, createErr := executeRootCommand("--config", configPath, "module", "create", mc.name, "--type", mc.moduleType)
		if createErr != nil {
			t.Fatalf("module create %s (%s) error = %v", mc.name, mc.moduleType, createErr)
		}
		if !strings.Contains(output, mc.name) {
			t.Fatalf("output missing module name %q: %s", mc.name, output)
		}
	}

	// Verify standard module types (feature, kit, shared, ui) produce 4 files each.
	for _, mc := range cases {
		if !mc.hasImpl || mc.moduleType == "relux-feature" {
			continue
		}

		t.Run(mc.moduleType+"_files", func(t *testing.T) {
			interfaceSrc := filepath.Join(modulesRoot, mc.name, "Sources", mc.name)
			implSrc := filepath.Join(modulesRoot, mc.name+"Impl", "Sources", mc.name+"Impl")

			requireDir(t, filepath.Join(modulesRoot, mc.name))
			requireDir(t, filepath.Join(modulesRoot, mc.name+"Impl"))
			requireFile(t, filepath.Join(modulesRoot, mc.name, "Package.swift"))
			requireFile(t, filepath.Join(modulesRoot, mc.name+"Impl", "Package.swift"))

			requireFile(t, filepath.Join(interfaceSrc, mc.name+".swift"))
			requireFile(t, filepath.Join(interfaceSrc, "Module", mc.name+".Module.swift"))
			requireFile(t, filepath.Join(interfaceSrc, "Module", mc.name+".Module+Interface.swift"))
			requireFile(t, filepath.Join(implSrc, "Module", mc.name+".Module+Impl.swift"))

			// Standard modules should not have Business/ subdirectory.
			if _, bizErr := os.Stat(filepath.Join(interfaceSrc, "Business")); bizErr == nil {
				t.Fatalf("standard module %q should not have Business/ directory", mc.name)
			}

			// Standard modules should not have swift-relux dependency.
			manifest := readFile(t, filepath.Join(modulesRoot, mc.name, "Package.swift"))
			if strings.Contains(manifest, "swift-relux") {
				t.Fatalf("%s (%s) Package.swift should not have swift-relux", mc.name, mc.moduleType)
			}
		})
	}

	// Verify utility module has no Impl package.
	t.Run("utility_no_impl", func(t *testing.T) {
		requireDir(t, filepath.Join(modulesRoot, "Logger"))
		if _, err := os.Stat(filepath.Join(modulesRoot, "LoggerImpl")); err == nil {
			t.Fatal("utility module Logger should not have LoggerImpl package")
		}
	})

	// Verify relux-feature module has all 8 files + swift-relux dep.
	t.Run("relux-feature_files", func(t *testing.T) {
		interfaceSrc := filepath.Join(modulesRoot, "Payment", "Sources", "Payment")
		implSrc := filepath.Join(modulesRoot, "PaymentImpl", "Sources", "PaymentImpl")

		requireFile(t, filepath.Join(interfaceSrc, "Payment.swift"))
		requireFile(t, filepath.Join(interfaceSrc, "Module", "Payment.Module.swift"))
		requireFile(t, filepath.Join(interfaceSrc, "Module", "Payment.Module+Interface.swift"))
		requireFile(t, filepath.Join(interfaceSrc, "Business", "Payment.Business+Action.swift"))
		requireFile(t, filepath.Join(interfaceSrc, "Business", "Payment.Business+Effect.swift"))
		requireFile(t, filepath.Join(implSrc, "Module", "Payment.Module+Impl.swift"))
		requireFile(t, filepath.Join(implSrc, "Business", "Payment.Business+State.swift"))
		requireFile(t, filepath.Join(implSrc, "Business", "Payment.Business+Flow.swift"))

		manifest := readFile(t, filepath.Join(modulesRoot, "Payment", "Package.swift"))
		if !strings.Contains(manifest, "swift-relux") {
			t.Fatalf("Payment Package.swift missing swift-relux dependency:\n%s", manifest)
		}

		implManifest := readFile(t, filepath.Join(modulesRoot, "PaymentImpl", "Package.swift"))
		if !strings.Contains(implManifest, "swift-relux") {
			t.Fatalf("PaymentImpl Package.swift missing swift-relux dependency:\n%s", implManifest)
		}
	})

	verifyNoTemplateArtifactsInSwiftFiles(t, projectRoot)
}

func verifyScaffoldOutput(t *testing.T, projectRoot string, cfg config.ProjectConfig) {
	t.Helper()

	requiredDirs := []string{
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Resources"),
		filepath.Join(projectRoot, cfg.ModulesPath),
	}
	for _, dir := range requiredDirs {
		requireDir(t, dir)
	}

	assetsPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Resources", "Assets.xcassets")
	requiredFiles := []string{
		filepath.Join(projectRoot, "Tuist.swift"),
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Workspace.swift"),
		filepath.Join(projectRoot, "Package.swift"),
		filepath.Join(projectRoot, "Makefile"),
		filepath.Join(projectRoot, ".gitignore"),
		filepath.Join(projectRoot, ".swiftlint.yml"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Configuration.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Configuration+AppGroups.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Bundle+InfoPlist.swift"),
		filepath.Join(assetsPath, "Contents.json"),
		filepath.Join(assetsPath, "AppIcon.appiconset", "Contents.json"),
		filepath.Join(assetsPath, "AppIcon.appiconset", "AppIcon.png"),
	}
	for _, path := range requiredFiles {
		requireFile(t, path)
	}

	makefile := readFile(t, filepath.Join(projectRoot, "Makefile"))
	phonyTargets := parsePhonyTargets(t, makefile)
	for _, target := range phonyTargets {
		if !strings.Contains(makefile, target+":") {
			t.Fatalf("Makefile missing target %q listed in .PHONY:\n%s", target, makefile)
		}
	}
}

func verifyTodoListModule(t *testing.T, projectRoot string, modulesPath string) {
	t.Helper()

	modulesRoot := filepath.Join(projectRoot, modulesPath)
	requireDir(t, filepath.Join(modulesRoot, "TodoList"))
	requireDir(t, filepath.Join(modulesRoot, "TodoListImpl"))

	requireFile(t, filepath.Join(modulesRoot, "TodoList", "Package.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TodoListImpl", "Package.swift"))

	interfaceSources := filepath.Join(modulesRoot, "TodoList", "Sources", "TodoList")
	implSources := filepath.Join(modulesRoot, "TodoListImpl", "Sources", "TodoListImpl")

	requireFile(t, filepath.Join(interfaceSources, "TodoList.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Module", "TodoList.Module.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Module", "TodoList.Module+Interface.swift"))

	requireFile(t, filepath.Join(implSources, "Module", "TodoList.Module+Impl.swift"))
}

func verifyReluxFeatureModule(t *testing.T, projectRoot string, modulesPath string) {
	t.Helper()

	modulesRoot := filepath.Join(projectRoot, modulesPath)
	requireDir(t, filepath.Join(modulesRoot, "Auth"))
	requireDir(t, filepath.Join(modulesRoot, "AuthImpl"))

	requireFile(t, filepath.Join(modulesRoot, "Auth", "Package.swift"))
	requireFile(t, filepath.Join(modulesRoot, "AuthImpl", "Package.swift"))

	interfaceSources := filepath.Join(modulesRoot, "Auth", "Sources", "Auth")
	implSources := filepath.Join(modulesRoot, "AuthImpl", "Sources", "AuthImpl")

	// Interface package: namespace, module, interface, action, effect
	requireFile(t, filepath.Join(interfaceSources, "Auth.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Module", "Auth.Module.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Module", "Auth.Module+Interface.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Business", "Auth.Business+Action.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Business", "Auth.Business+Effect.swift"))

	// Impl package: impl, state, flow
	requireFile(t, filepath.Join(implSources, "Module", "Auth.Module+Impl.swift"))
	requireFile(t, filepath.Join(implSources, "Business", "Auth.Business+State.swift"))
	requireFile(t, filepath.Join(implSources, "Business", "Auth.Business+Flow.swift"))

	// Verify Package.swift files contain swift-relux dependency
	interfaceManifest := readFile(t, filepath.Join(modulesRoot, "Auth", "Package.swift"))
	if !strings.Contains(interfaceManifest, "swift-relux") {
		t.Fatalf("Auth interface Package.swift missing swift-relux dependency:\n%s", interfaceManifest)
	}
	if !strings.Contains(interfaceManifest, `"Relux"`) {
		t.Fatalf("Auth interface Package.swift missing Relux product:\n%s", interfaceManifest)
	}

	implManifest := readFile(t, filepath.Join(modulesRoot, "AuthImpl", "Package.swift"))
	if !strings.Contains(implManifest, "swift-relux") {
		t.Fatalf("Auth impl Package.swift missing swift-relux dependency:\n%s", implManifest)
	}
	if !strings.Contains(implManifest, `"Relux"`) {
		t.Fatalf("Auth impl Package.swift missing Relux product:\n%s", implManifest)
	}

	// Verify root Package.swift has swift-relux
	rootManifest := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootManifest, "swift-relux") {
		t.Fatalf("root Package.swift missing swift-relux dependency:\n%s", rootManifest)
	}

	// Verify template content is correct (no template artifacts, has Relux imports)
	actionContent := readFile(t, filepath.Join(interfaceSources, "Business", "Auth.Business+Action.swift"))
	if !strings.Contains(actionContent, "import Relux") {
		t.Fatalf("Auth action file missing 'import Relux':\n%s", actionContent)
	}

	stateContent := readFile(t, filepath.Join(implSources, "Business", "Auth.Business+State.swift"))
	if !strings.Contains(stateContent, "import Relux") {
		t.Fatalf("Auth state file missing 'import Relux':\n%s", stateContent)
	}
	if !strings.Contains(stateContent, "import Auth") {
		t.Fatalf("Auth state file missing 'import Auth':\n%s", stateContent)
	}
}

func verifyTokenProviderModule(t *testing.T, projectRoot string, modulesPath string) {
	t.Helper()

	modulesRoot := filepath.Join(projectRoot, modulesPath)
	requireDir(t, filepath.Join(modulesRoot, "TokenProvider"))
	requireDir(t, filepath.Join(modulesRoot, "TokenProviderImpl"))

	requireFile(t, filepath.Join(modulesRoot, "TokenProvider", "Package.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProviderImpl", "Package.swift"))

	interfaceSources := filepath.Join(modulesRoot, "TokenProvider", "Sources", "TokenProvider")
	implSources := filepath.Join(modulesRoot, "TokenProviderImpl", "Sources", "TokenProviderImpl")

	requireFile(t, filepath.Join(interfaceSources, "TokenProvider.swift"))
	requireFile(t, filepath.Join(interfaceSources, "TokenProvider.AuthData.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Module", "TokenProvider.Module.swift"))
	requireFile(t, filepath.Join(interfaceSources, "Module", "TokenProvider.Module+Interface.swift"))
	requireFile(t, filepath.Join(implSources, "Module", "TokenProvider.Module+Impl.swift"))

	// Verify impl contains actor.
	implContent := readFile(t, filepath.Join(implSources, "Module", "TokenProvider.Module+Impl.swift"))
	if !strings.Contains(implContent, "public actor Impl") {
		t.Fatalf("TokenProvider impl missing actor declaration:\n%s", implContent)
	}

	// Verify protocol contains expected methods.
	protoContent := readFile(t, filepath.Join(interfaceSources, "Module", "TokenProvider.Module+Interface.swift"))
	if !strings.Contains(protoContent, "func setAuthData") {
		t.Fatalf("TokenProvider interface missing setAuthData:\n%s", protoContent)
	}
}

func verifyHttpClientSetup(t *testing.T, projectRoot string, cfg config.ProjectConfig) {
	t.Helper()

	// Verify Configuration+HttpClient.swift exists.
	configPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Configuration+HttpClient.swift")
	requireFile(t, configPath)
	configContent := readFile(t, configPath)
	if !strings.Contains(configContent, "timeoutForResponse") {
		t.Fatalf("Configuration+HttpClient.swift missing timeoutForResponse:\n%s", configContent)
	}

	// Verify Package.swift has swift-httpclient.
	packageContent := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(packageContent, "swift-httpclient") {
		t.Fatalf("Package.swift missing swift-httpclient:\n%s", packageContent)
	}

	// Verify Project.swift has HttpClient.
	projectContent := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectContent, "HttpClient") {
		t.Fatalf("Project.swift missing HttpClient:\n%s", projectContent)
	}

	// Verify Registry.swift has HttpClient registration and builder.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	registryContent := readFile(t, registryPath)
	for _, expected := range []string{
		"import HttpClient",
		"IRpcAsyncClient.self",
		"buildHttpClient",
	} {
		if !strings.Contains(registryContent, expected) {
			t.Fatalf("Registry.swift missing %q:\n%s", expected, registryContent)
		}
	}
}

func verifyNoTemplateArtifactsInSwiftFiles(t *testing.T, root string) {
	t.Helper()

	swiftFiles := make([]string, 0, 16)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".swift" {
			swiftFiles = append(swiftFiles, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkDir(%q) error = %v", root, err)
	}

	if len(swiftFiles) == 0 {
		t.Fatalf("no Swift files found under %q", root)
	}

	for _, path := range swiftFiles {
		content := readFile(t, path)
		for _, token := range []string{"{{", "}}", "{%", "%}", "<#"} {
			if strings.Contains(content, token) {
				t.Fatalf("Swift file %q contains template artifact %q", path, token)
			}
		}
	}
}

func verifyGoBuild(t *testing.T, repoRoot string) {
	t.Helper()

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./... failed: %v\noutput:\n%s", err, string(output))
	}
}

func executeRootCommand(args ...string) (string, error) {
	root := cli.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader(""))
	root.SetArgs(args)

	err := root.Execute()
	return out.String(), err
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to find repo root from %q", dir)
		}
		dir = parent
	}
}

func requireDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("path %q is not a directory", path)
	}
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

func parsePhonyTargets(t *testing.T, makefile string) []string {
	t.Helper()

	for _, line := range strings.Split(makefile, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ".PHONY:") {
			continue
		}

		targets := strings.Fields(strings.TrimSpace(strings.TrimPrefix(trimmed, ".PHONY:")))
		if len(targets) == 0 {
			t.Fatalf(".PHONY line exists but has no targets:\n%s", makefile)
		}
		return targets
	}

	t.Fatalf("Makefile missing .PHONY declaration:\n%s", makefile)
	return nil
}
