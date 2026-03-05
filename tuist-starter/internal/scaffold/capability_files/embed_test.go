package capability_files

import (
	"strings"
	"testing"
)

func TestCapabilityFilesEmbed(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		contains string
	}{
		{
			name:     "Capability.swift is non-empty and contains type declaration",
			file:     "Capability.swift",
			contains: "public struct Capability",
		},
		{
			name:     "EntitlementsFactory.swift is non-empty and contains type declaration",
			file:     "EntitlementsFactory.swift",
			contains: "public enum EntitlementsFactory",
		},
		{
			name:     "PortalCapability file contains enum",
			file:     "Capability+PortalCapability.swift",
			contains: "PortalCapability",
		},
		{
			name:     "AppleSupportedCapabilities.swift is non-empty and contains type declaration",
			file:     "AppleSupportedCapabilities.swift",
			contains: "public enum AppleSupportedCapabilities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := CapabilityFiles.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("ReadFile(%q) error: %v", tt.file, err)
			}
			if len(data) == 0 {
				t.Fatalf("ReadFile(%q) returned empty content", tt.file)
			}
			content := string(data)
			if !strings.Contains(content, tt.contains) {
				t.Errorf("ReadFile(%q) does not contain %q", tt.file, tt.contains)
			}
		})
	}
}

func TestCapabilitySwiftNoNamespacing(t *testing.T) {
	data, err := CapabilityFiles.ReadFile("Capability.swift")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	content := string(data)

	forbidden := []string{
		"Namespacing",
		"environmentSuffix",
		"ConfigurationHelper",
		"sharedRoot",
		"case shared",
		"IdentifierSegments",
	}

	for _, s := range forbidden {
		if strings.Contains(content, s) {
			t.Errorf("Capability.swift should not contain %q (removed reference)", s)
		}
	}
}

func TestEntitlementsFactoryNoNamespacing(t *testing.T) {
	data, err := CapabilityFiles.ReadFile("EntitlementsFactory.swift")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	content := string(data)

	forbidden := []string{
		"Namespacing",
		"applyNamespacing",
		"ConfigurationHelper",
		"sharedRoot",
		"resolveSharedRoot",
		"Environment.coreRoot",
		"Environment.sharedRoot",
		"TUIST_BUNDLE_ID_SUFFIX",
		"IdentifierSegments",
	}

	for _, s := range forbidden {
		if strings.Contains(content, s) {
			t.Errorf("EntitlementsFactory.swift should not contain %q (removed reference)", s)
		}
	}
}
