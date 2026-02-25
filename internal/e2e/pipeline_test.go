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
	verifyNoTemplateArtifactsInSwiftFiles(t, projectRoot)
	verifyGoBuild(t, repoRoot)
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

	requiredFiles := []string{
		filepath.Join(projectRoot, "Tuist.swift"),
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Workspace.swift"),
		filepath.Join(projectRoot, "Package.swift"),
		filepath.Join(projectRoot, "Makefile"),
		filepath.Join(projectRoot, ".gitignore"),
		filepath.Join(projectRoot, ".swiftlint.yml"),
		filepath.Join(projectRoot, cfg.AppName+".entitlements"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App.swift"),
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
	requireFile(t, filepath.Join(interfaceSources, "TodoList.Module.swift"))
	requireFile(t, filepath.Join(interfaceSources, "TodoList.Module+Interface.swift"))

	requireFile(t, filepath.Join(implSources, "TodoList.Module+Impl.swift"))
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
