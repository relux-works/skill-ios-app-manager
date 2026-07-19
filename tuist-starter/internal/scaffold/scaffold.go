package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold/capability_files"
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
	if cfg.HasRuntimeProfiles() {
		if err := ValidateFirebaseClientConfigurationInputs(root, cfg); err != nil {
			return nil, fmt.Errorf("validate Firebase client configuration inputs: %w", err)
		}
	}

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
	files[filepath.Join(root, "Targets", appName, "Sources", "App.swift")] = GenerateAppStub(cfg)
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration.swift")] = GenerateConfiguration()
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration+ApplicationConfiguration.swift")] = GenerateConfigurationApplicationConfiguration(cfg)
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration+Keychain.swift")] = GenerateConfigurationKeychain(cfg)
	files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Bundle+InfoPlist.swift")] = GenerateInfoPlistHelper()

	packageSwift, err := GenerateSharedConfigurationPackageSwift(cfg)
	if err != nil {
		return nil, fmt.Errorf("generate shared configuration Package.swift: %w", err)
	}
	files[sharedConfigurationPackageSwiftPath(root, cfg)] = packageSwift
	files[sharedConfigurationInfoPlistReadingSourcePath(root, cfg)] = GenerateSharedConfigurationInfoPlistReadingSwift(cfg)
	files[applicationConfigurationSharedConfigurationSourcePath(root, cfg)] = GenerateApplicationConfigurationSharedConfigurationSwift(cfg)
	if cfg.HasRuntimeProfiles() {
		files[runtimeProfilesSwiftPath(root, appName)] = GenerateRuntimeProfilesSwift(cfg)
		files[runtimeProfilesProjectDescriptionPath(root)] = GenerateRuntimeProfilesProjectDescriptionSwift(cfg)
	}

	rootPackagePath := filepath.Join(root, "Package.swift")
	if content, ok := files[rootPackagePath]; ok {
		updated, _, err := syncRootPackageSharedConfigurationDependencyContent(content, cfg)
		if err != nil {
			return nil, err
		}
		updated, err = syncRuntimeProfilePackageManifestContent(updated, cfg, cfg.HasRuntimeProfiles())
		if err != nil {
			return nil, fmt.Errorf("sync runtime profiles in root Package.swift: %w", err)
		}
		files[rootPackagePath] = updated
	}

	projectPath := filepath.Join(root, "Project.swift")
	if content, ok := files[projectPath]; ok {
		updated, _, err := syncProjectManifestExternalDependencyContent(
			content,
			appGroupSharedConfigurationModuleName(cfg),
		)
		if err != nil {
			return nil, err
		}
		files[projectPath] = updated
	}

	if len(cfg.AppGroups) > 0 {
		files[filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Configuration+AppGroups.swift")] = GenerateConfigurationAppGroups(cfg)

		packageSwift, err := GenerateAppGroupSharedConfigurationPackageSwift(cfg)
		if err != nil {
			return nil, fmt.Errorf("generate app-group shared configuration Package.swift: %w", err)
		}
		files[appGroupSharedConfigurationPackageSwiftPath(root, cfg)] = packageSwift
		files[appGroupSharedConfigurationSourcePath(root, cfg)] = GenerateAppGroupSharedConfigurationSwift(cfg)
	}

	assetsPath := filepath.Join(root, "Targets", appName, "Resources", "Assets.xcassets")
	files[filepath.Join(assetsPath, "Contents.json")] = assetCatalogContentsJSON()
	files[filepath.Join(assetsPath, "AppIcon.appiconset", "Contents.json")] = appIconsetContentsJSON()

	// Copy capability DSL files into ProjectDescriptionHelpers.
	helpersDir := filepath.Join(root, "Tuist", "ProjectDescriptionHelpers")
	err = fs.WalkDir(capability_files.CapabilityFiles, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".swift") {
			return nil
		}
		data, readErr := capability_files.CapabilityFiles.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", path, readErr)
		}
		files[filepath.Join(helpersDir, path)] = string(data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("embed capability files: %w", err)
	}

	// Generate initial AppCapabilities.swift.
	files[filepath.Join(helpersDir, "AppCapabilities.swift")] = GenerateAppCapabilitiesForConfig(cfg)

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
