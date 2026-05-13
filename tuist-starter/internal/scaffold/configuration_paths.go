package scaffold

import (
	"path/filepath"
)

func configurationFilePath(root string, appName string, fileName string) string {
	rootCandidates := configurationRootFileCandidatePaths(root, fileName)
	for _, candidate := range rootCandidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	if len(rootCandidates) > 0 {
		return rootCandidates[0]
	}

	for _, candidate := range configurationFileCandidatePaths(root, appName, fileName) {
		if fileExists(candidate) {
			return candidate
		}
	}
	return defaultConfigurationFilePath(root, appName, fileName)
}

func staleConfigurationFilePaths(root string, appName string, fileName string, selectedPath string) []string {
	stalePaths := make([]string, 0)
	for _, candidate := range configurationFileCandidatePaths(root, appName, fileName) {
		if candidate == selectedPath || !fileExists(candidate) {
			continue
		}
		stalePaths = append(stalePaths, candidate)
	}
	return stalePaths
}

func configurationFileCandidatePaths(root string, appName string, fileName string) []string {
	candidates := make([]string, 0)
	candidates = append(candidates, configurationRootFileCandidatePaths(root, fileName)...)
	candidates = append(candidates,
		filepath.Join(root, "Targets", appName, "Sources", "Configuration", "Runtime", fileName),
		defaultConfigurationFilePath(root, appName, fileName),
	)
	for _, pattern := range []string{
		filepath.Join(root, "Targets", "*", "Sources", "Configuration", "Runtime", fileName),
		filepath.Join(root, "Targets", "*", "Sources", "Configuration", fileName),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		candidates = append(candidates, matches...)
	}
	return appendUniqueStrings(nil, candidates...)
}

func configurationRootFileCandidatePaths(root string, fileName string) []string {
	matches, err := filepath.Glob(filepath.Join(root, "Targets", "*", "Sources", "Configuration", "Configuration.swift"))
	if err != nil {
		return nil
	}

	candidates := make([]string, 0, len(matches)*2)
	for _, configurationSwiftPath := range matches {
		configurationDir := filepath.Dir(configurationSwiftPath)
		runtimePath := filepath.Join(configurationDir, "Runtime", fileName)
		if fileExists(runtimePath) {
			candidates = append(candidates, runtimePath)
			continue
		}
		candidates = append(candidates, filepath.Join(configurationDir, fileName))
	}
	return appendUniqueStrings(nil, candidates...)
}

func defaultConfigurationFilePath(root string, appName string, fileName string) string {
	return filepath.Join(root, "Targets", appName, "Sources", "Configuration", fileName)
}
