package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/spf13/cobra"
)

func fakeModule() *registry.Module {
	return &registry.Module{
		ID:          "fake-mod",
		Name:        "FakeMod",
		Description: "A fake module for testing",
		Category:    registry.Foundation,

		Plan: func(input registry.SetupInput) (string, error) {
			return "Will create FakeMod files in " + input.ProjectRoot, nil
		},
		Setup: func(input registry.SetupInput) error {
			return nil
		},
		UsageGuide: "Use FakeMod like this.",

		CLIUse:     "fake-mod",
		CLIShort:   "Manage FakeMod module",
		SetupShort: "Create FakeMod module",
	}
}

func fakeModuleWithExtraFlags() *registry.Module {
	mod := fakeModule()
	mod.ExtraFlags = []registry.ExtraFlag{
		{Name: "access-group", Usage: "app group for access", Required: true, ArgKey: "access-group"},
		{Name: "region", Usage: "deployment region", Required: false, ArgKey: "region"},
	}
	mod.Setup = func(input registry.SetupInput) error {
		// Store extra args for verification — we check output instead.
		return nil
	}
	return mod
}

func fakeDepModule() *registry.Module {
	return &registry.Module{
		ID:   "fake-dep",
		Name: "FakeDep",
	}
}

func fakeModuleWithDeps() *registry.Module {
	mod := fakeModule()
	mod.Dependencies = []registry.ModuleID{"fake-dep"}
	return mod
}

func executeSetupCommandWithProject(
	t *testing.T,
	mod *registry.Module,
	input string,
	prepareProject func(projectRoot string, cfg config.ProjectConfig),
	extraArgs ...string,
) (string, error) {
	t.Helper()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	if prepareProject != nil {
		prepareProject(projectRoot, cfg)
	}
	configPath := writeModuleConfig(t, projectRoot, cfg)

	opts := &RootOptions{ConfigPath: configPath}
	cmd := NewSetupCommand(mod, opts)

	root := &cobra.Command{Use: "test"}
	root.PersistentFlags().StringVarP(&opts.ConfigPath, "config", "c", config.DefaultConfigPath, "")

	root.AddCommand(cmd)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader(input))

	args := []string{"--config", configPath, mod.CLIUse, "setup"}
	args = append(args, extraArgs...)
	root.SetArgs(args)

	err := root.Execute()
	return out.String(), err
}

func executeSetupCommand(t *testing.T, mod *registry.Module, input string, extraArgs ...string) (string, error) {
	t.Helper()
	return executeSetupCommandWithProject(t, mod, input, nil, extraArgs...)
}

func registerTestModulesWithDeps(t *testing.T, mod *registry.Module) {
	t.Helper()
	saved := registry.All() // reference to old map (Reset creates a new one)
	registry.Reset()
	// Re-register all real modules so other tests aren't affected.
	for _, m := range saved {
		registry.Register(m)
	}
	registry.Register(fakeDepModule())
	registry.Register(mod)
	t.Cleanup(func() {
		registry.Reset()
		for _, m := range saved {
			registry.Register(m)
		}
	})
}

func writeRegistrySwift(t *testing.T, projectRoot string, appName string, content string) {
	t.Helper()
	registryDir := filepath.Join(projectRoot, "Targets", appName, "Sources", "App")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", registryDir, err)
	}
	registryPath := filepath.Join(registryDir, appName+".Registry.swift")
	if err := os.WriteFile(registryPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", registryPath, err)
	}
}

func TestSetupCommandDryRun(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	setupCalled := false
	mod.Setup = func(_ registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommand(t, mod, "", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run error = %v", err)
	}

	if setupCalled {
		t.Fatal("Setup() was called during dry-run, should not be")
	}

	if !strings.Contains(output, "Will create FakeMod files") {
		t.Fatalf("dry-run output missing plan text:\n%s", output)
	}

	if !strings.Contains(output, "Use FakeMod like this.") {
		t.Fatalf("dry-run output missing usage guide:\n%s", output)
	}

	if strings.Contains(output, "setup complete") {
		t.Fatalf("dry-run output should not contain 'setup complete':\n%s", output)
	}
}

func TestSetupCommandYes(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	setupCalled := false
	mod.Setup = func(_ registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommand(t, mod, "", "--yes")
	if err != nil {
		t.Fatalf("--yes error = %v", err)
	}

	if !setupCalled {
		t.Fatal("Setup() was not called with --yes")
	}

	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}

	if strings.Contains(output, "Proceed?") {
		t.Fatalf("--yes should skip confirmation prompt:\n%s", output)
	}
}

func TestSetupCommandYesShortFlag(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	setupCalled := false
	mod.Setup = func(_ registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommand(t, mod, "", "-y")
	if err != nil {
		t.Fatalf("-y error = %v", err)
	}

	if !setupCalled {
		t.Fatal("Setup() was not called with -y")
	}

	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}
}

func TestSetupCommandConfirmYes(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	setupCalled := false
	mod.Setup = func(_ registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommand(t, mod, "y\n")
	if err != nil {
		t.Fatalf("confirm=y error = %v", err)
	}

	if !setupCalled {
		t.Fatal("Setup() was not called after confirming 'y'")
	}

	if !strings.Contains(output, "Proceed? [y/N]") {
		t.Fatalf("output missing confirmation prompt:\n%s", output)
	}

	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}
}

func TestSetupCommandConfirmNo(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	setupCalled := false
	mod.Setup = func(_ registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommand(t, mod, "n\n")
	if err != nil {
		t.Fatalf("confirm=n error = %v", err)
	}

	if setupCalled {
		t.Fatal("Setup() was called after declining, should not be")
	}

	if strings.Contains(output, "setup complete") {
		t.Fatalf("output should not contain 'setup complete' after decline:\n%s", output)
	}
}

func TestSetupCommandDepsNotMet(t *testing.T) {
	mod := fakeModuleWithDeps()
	registerTestModulesWithDeps(t, mod)

	planCalled := false
	setupCalled := false
	mod.Plan = func(input registry.SetupInput) (string, error) {
		planCalled = true
		return "plan", nil
	}
	mod.Setup = func(input registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	_, err := executeSetupCommand(t, mod, "", "--yes")
	if err == nil {
		t.Fatal("expected error when Registry.swift is missing, got nil")
	}
	if !strings.Contains(err.Error(), "Registry.swift not found") {
		t.Fatalf("error = %q, want missing Registry.swift message", err)
	}
	if planCalled {
		t.Fatal("Plan() should not run when dependency check fails")
	}
	if setupCalled {
		t.Fatal("Setup() should not run when dependency check fails")
	}
}

func TestSetupCommandDepsMet(t *testing.T) {
	mod := fakeModuleWithDeps()
	registerTestModulesWithDeps(t, mod)

	planCalled := false
	setupCalled := false
	mod.Plan = func(input registry.SetupInput) (string, error) {
		planCalled = true
		return "plan", nil
	}
	mod.Setup = func(input registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommandWithProject(
		t,
		mod,
		"",
		func(projectRoot string, cfg config.ProjectConfig) {
			writeRegistrySwift(t, projectRoot, cfg.AppName, "FakeDep")
		},
		"--yes",
	)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !planCalled {
		t.Fatal("Plan() was not called when dependencies were met")
	}
	if !setupCalled {
		t.Fatal("Setup() was not called when dependencies were met")
	}
	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}
}

func TestSetupCommandDepsPartiallyMet(t *testing.T) {
	mod := fakeModuleWithDeps()
	registerTestModulesWithDeps(t, mod)

	planCalled := false
	setupCalled := false
	mod.Plan = func(input registry.SetupInput) (string, error) {
		planCalled = true
		return "plan", nil
	}
	mod.Setup = func(input registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	_, err := executeSetupCommandWithProject(
		t,
		mod,
		"",
		func(projectRoot string, cfg config.ProjectConfig) {
			writeRegistrySwift(t, projectRoot, cfg.AppName, "SomeOtherModule")
		},
		"--yes",
	)
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "missing dependencies") {
		t.Fatalf("error = %q, want missing dependencies message", err)
	}
	if planCalled {
		t.Fatal("Plan() should not run when dependencies are missing")
	}
	if setupCalled {
		t.Fatal("Setup() should not run when dependencies are missing")
	}
}

func TestSetupCommandNoDepsSkipsCheck(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	planCalled := false
	setupCalled := false
	mod.Plan = func(input registry.SetupInput) (string, error) {
		planCalled = true
		return "plan", nil
	}
	mod.Setup = func(input registry.SetupInput) error {
		setupCalled = true
		return nil
	}

	output, err := executeSetupCommand(t, mod, "", "--yes")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !planCalled {
		t.Fatal("Plan() was not called for module without dependencies")
	}
	if !setupCalled {
		t.Fatal("Setup() was not called for module without dependencies")
	}
	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}
}

func TestSetupCommandExtraFlags(t *testing.T) {
	t.Parallel()

	mod := fakeModuleWithExtraFlags()
	var capturedInput registry.SetupInput
	mod.Setup = func(input registry.SetupInput) error {
		capturedInput = input
		return nil
	}

	output, err := executeSetupCommand(t, mod, "", "--yes", "--access-group", "group.com.test", "--region", "us-east")
	if err != nil {
		t.Fatalf("extra flags error = %v", err)
	}

	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}

	if capturedInput.ExtraArgs["access-group"] != "group.com.test" {
		t.Fatalf("ExtraArgs[access-group] = %q, want %q", capturedInput.ExtraArgs["access-group"], "group.com.test")
	}

	if capturedInput.ExtraArgs["region"] != "us-east" {
		t.Fatalf("ExtraArgs[region] = %q, want %q", capturedInput.ExtraArgs["region"], "us-east")
	}
}

func TestSetupCommandExtraFlagRequired(t *testing.T) {
	t.Parallel()

	mod := fakeModuleWithExtraFlags()

	// Don't provide required --access-group flag.
	_, err := executeSetupCommand(t, mod, "", "--yes")
	if err == nil {
		t.Fatal("expected error for missing required flag, got nil")
	}
}

func TestSetupCommandExtraFlagOptionalOmitted(t *testing.T) {
	t.Parallel()

	mod := fakeModuleWithExtraFlags()
	var capturedInput registry.SetupInput
	mod.Setup = func(input registry.SetupInput) error {
		capturedInput = input
		return nil
	}

	// Provide required flag, omit optional --region.
	output, err := executeSetupCommand(t, mod, "", "--yes", "--access-group", "group.com.test")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(output, "FakeMod setup complete") {
		t.Fatalf("output missing completion message:\n%s", output)
	}

	if _, ok := capturedInput.ExtraArgs["region"]; ok {
		t.Fatalf("ExtraArgs should not contain 'region' when omitted, got %q", capturedInput.ExtraArgs["region"])
	}
}

func TestSetupCommandNoConfig(t *testing.T) {
	t.Parallel()

	mod := fakeModule()

	opts := &RootOptions{ConfigPath: "/nonexistent/path/ios-app-manager.json"}
	cmd := NewSetupCommand(mod, opts)

	root := &cobra.Command{Use: "test"}
	root.PersistentFlags().StringVarP(&opts.ConfigPath, "config", "c", "", "")
	root.AddCommand(cmd)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--config", "/nonexistent/path/ios-app-manager.json", mod.CLIUse, "setup", "--yes"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when config missing, got nil")
	}
}

func TestSetupCommandHelp(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	opts := &RootOptions{}
	cmd := NewSetupCommand(mod, opts)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute()

	output := out.String()
	if !strings.Contains(output, "setup") {
		t.Fatalf("help output missing 'setup':\n%s", output)
	}
}

func TestSetupCommandPlanPassesCorrectInput(t *testing.T) {
	t.Parallel()

	mod := fakeModule()
	var capturedInput registry.SetupInput
	mod.Plan = func(input registry.SetupInput) (string, error) {
		capturedInput = input
		return "plan", nil
	}

	_, err := executeSetupCommand(t, mod, "", "--dry-run")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if capturedInput.AppName == "" {
		t.Fatal("Plan received empty AppName")
	}

	if capturedInput.ProjectRoot == "" {
		t.Fatal("Plan received empty ProjectRoot")
	}
}
