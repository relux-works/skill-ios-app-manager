package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templaterenderer "github.com/relux-works/ios-app-manager/internal/template"
)

const (
	defaultOutputDir  = "."
	defaultModulesDir = "Packages"
)

// Renderer defines the template rendering dependency used by Scaffolder.
type Renderer interface {
	Render(cfg config.ProjectConfig) (map[string]string, error)
}

// Scaffolder creates a full project scaffold on disk.
type Scaffolder struct {
	renderer  Renderer
	mkdirAll  func(path string, perm os.FileMode) error
	writeFile func(name string, data []byte, perm os.FileMode) error
	readFile  func(name string) ([]byte, error)
	stat      func(name string) (os.FileInfo, error)
}

// New creates a Scaffolder. If renderer is nil, a default template renderer is used.
func New(renderer Renderer) *Scaffolder {
	if renderer == nil {
		renderer = templaterenderer.NewRenderer()
	}

	return &Scaffolder{
		renderer:  renderer,
		mkdirAll:  os.MkdirAll,
		writeFile: os.WriteFile,
		readFile:  os.ReadFile,
		stat:      os.Stat,
	}
}

// Scaffold renders and writes a full Tuist project structure to outputDir.
// Returns absolute paths for all written files.
func (s *Scaffolder) Scaffold(cfg config.ProjectConfig, outputDir string, force bool) ([]string, error) {
	if s == nil {
		return nil, errors.New("scaffolder is required")
	}
	if s.renderer == nil {
		return nil, errors.New("template renderer is required")
	}

	root := normalizeOutputDir(outputDir)
	appName := normalizeAppName(cfg.AppName)
	modulesPath := normalizeModulesPath(cfg.ModulesPath)

	filesToWrite, err := s.planFiles(cfg, root, appName)
	if err != nil {
		return nil, err
	}

	if !force {
		existing, err := s.findExistingFiles(filesToWrite)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			return nil, fmt.Errorf(
				"output directory %q already contains scaffold files (%s); pass --force to overwrite",
				root,
				strings.Join(existing, ", "),
			)
		}
	}

	requiredDirs := []string{
		root,
		filepath.Join(root, "Targets", appName, "Sources"),
		filepath.Join(root, "Targets", appName, "Resources"),
		filepath.Join(root, modulesPath),
	}

	for _, path := range requiredDirs {
		if err := s.mkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %q: %w", path, err)
		}
	}

	written := make([]string, 0, len(filesToWrite)+1)
	paths := make([]string, 0, len(filesToWrite))
	for path := range filesToWrite {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		content := filesToWrite[path]

		parent := filepath.Dir(path)
		if err := s.mkdirAll(parent, 0o755); err != nil {
			return nil, fmt.Errorf("create parent directory %q: %w", parent, err)
		}

		if err := s.writeFile(path, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write file %q: %w", path, err)
		}
		written = append(written, path)
	}

	iconPath := filepath.Join(root, "Targets", appName, "Resources", "Assets.xcassets", "AppIcon.appiconset", "AppIcon.png")
	iconData, err := generatePlaceholderIcon()
	if err != nil {
		return nil, err
	}
	if err := s.writeFile(iconPath, iconData, 0o644); err != nil {
		return nil, fmt.Errorf("write file %q: %w", iconPath, err)
	}
	written = append(written, iconPath)

	return written, nil
}

func (s *Scaffolder) planFiles(cfg config.ProjectConfig, root string, appName string) (map[string]string, error) {
	rendered, err := s.renderer.Render(cfg)
	if err != nil {
		return nil, fmt.Errorf("render Tuist templates: %w", err)
	}

	files := make(map[string]string, len(rendered)+6)
	for name, content := range rendered {
		switch strings.TrimSpace(name) {
		case "Tuist.swift":
			files[filepath.Join(root, "Tuist.swift")] = content
		default:
			files[filepath.Join(root, name)] = content
		}
	}

	makefilePath := filepath.Join(root, "Makefile")
	makefileContent := GenerateMakefile(cfg)
	if s.readFile != nil {
		existing, err := s.readFile(makefilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("read existing makefile %q: %w", makefilePath, err)
			}
		} else {
			makefileContent = GenerateMakefilePreservingCustom(cfg, string(existing))
		}
	}

	files[makefilePath] = makefileContent
	files[filepath.Join(root, ".periphery.yml")] = GeneratePeripheryConfig(cfg)
	files[filepath.Join(root, ".swiftlint.yml")] = GenerateSwiftLintConfig(cfg)
	files[filepath.Join(root, ".gitignore")] = GenerateGitignore()
	files[filepath.Join(root, appName+".entitlements")] = GenerateEntitlements(cfg)
	files[filepath.Join(root, "Targets", appName, "Sources", "App.swift")] = GenerateAppStub(cfg)
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration.swift")] = GenerateConfiguration()
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration+Keychain.swift")] = GenerateConfigurationKeychain(cfg)
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Bundle+InfoPlist.swift")] = GenerateInfoPlistHelper()

	if len(cfg.AppGroups) > 0 {
		files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration+AppGroups.swift")] = GenerateConfigurationAppGroups(cfg)
	}

	assetsPath := filepath.Join(root, "Targets", appName, "Resources", "Assets.xcassets")
	files[filepath.Join(assetsPath, "Contents.json")] = assetCatalogContentsJSON()
	files[filepath.Join(assetsPath, "AppIcon.appiconset", "Contents.json")] = appIconsetContentsJSON()

	return files, nil
}

func (s *Scaffolder) findExistingFiles(files map[string]string) ([]string, error) {
	existing := make([]string, 0)
	for path := range files {
		if _, err := s.stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat %q: %w", path, err)
		}
		existing = append(existing, path)
	}

	sort.Strings(existing)
	return existing, nil
}

func normalizeOutputDir(outputDir string) string {
	path := strings.TrimSpace(outputDir)
	if path == "" {
		path = defaultOutputDir
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Clean(absolutePath)
}

func normalizeAppName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "App"
	}
	return name
}

func normalizeModulesPath(raw string) string {
	modulesPath := strings.TrimSpace(raw)
	if modulesPath == "" {
		modulesPath = defaultModulesDir
	}

	modulesPath = filepath.Clean(filepath.FromSlash(modulesPath))
	if modulesPath == "." {
		return defaultModulesDir
	}

	if filepath.IsAbs(modulesPath) {
		modulesPath = strings.TrimPrefix(modulesPath, string(filepath.Separator))
	}

	if modulesPath == "" {
		return defaultModulesDir
	}

	return modulesPath
}
