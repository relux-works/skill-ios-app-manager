package modules

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetModuleType(t *testing.T) {
	t.Parallel()

	expected := expectedDescriptors()
	tests := []struct {
		name  string
		input string
		want  ModuleTypeDescriptor
	}{
		{
			name:  "feature",
			input: "feature",
			want:  expected[ModuleTypeFeature],
		},
		{
			name:  "kit",
			input: " kit ",
			want:  expected[ModuleTypeKit],
		},
		{
			name:  "shared",
			input: "SHARED",
			want:  expected[ModuleTypeShared],
		},
		{
			name:  "ui",
			input: "Ui",
			want:  expected[ModuleTypeUI],
		},
		{
			name:  "utility",
			input: "utility",
			want:  expected[ModuleTypeUtility],
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetModuleType(tc.input)
			if err != nil {
				t.Fatalf("GetModuleType(%q) error = %v", tc.input, err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("GetModuleType(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}

func TestGetModuleTypeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty",
			input:   "",
			wantErr: "module type is required",
		},
		{
			name:    "spaces",
			input:   "   ",
			wantErr: "module type is required",
		},
		{
			name:    "unknown",
			input:   "product",
			wantErr: `unknown module type "product"`,
		},
		{
			name:    "relux-feature is blueprint-only",
			input:   "relux-feature",
			wantErr: `unknown module type "relux-feature"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := GetModuleType(tc.input)
			if err == nil {
				t.Fatalf("GetModuleType(%q) error = nil, want %q", tc.input, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("GetModuleType(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateModuleType(t *testing.T) {
	t.Parallel()

	validInputs := []string{"feature", "kit", "shared", "ui", "utility"}
	for _, input := range validInputs {
		if err := ValidateModuleType(input); err != nil {
			t.Fatalf("ValidateModuleType(%q) error = %v", input, err)
		}
	}

	if err := ValidateModuleType("invalid"); err == nil {
		t.Fatal("ValidateModuleType(invalid) error = nil, want non-nil")
	}
}

func TestAllModuleTypes(t *testing.T) {
	t.Parallel()

	got := AllModuleTypes()
	expected := expectedDescriptors()
	want := []ModuleTypeDescriptor{
		expected[ModuleTypeFeature],
		expected[ModuleTypeKit],
		expected[ModuleTypeShared],
		expected[ModuleTypeUI],
		expected[ModuleTypeUtility],
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllModuleTypes() = %#v, want %#v", got, want)
	}
}

func TestTemplateMappingsAndCounts(t *testing.T) {
	t.Parallel()

	templateSet := []string{"namespace", "module", "interface", "impl"}
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantSet   []string
	}{
		{
			name:      "feature has 4 templates",
			input:     "feature",
			wantCount: 4,
			wantSet:   templateSet,
		},
		{
			name:      "kit has 4 templates",
			input:     "kit",
			wantCount: 4,
			wantSet:   templateSet,
		},
		{
			name:      "shared has 4 templates",
			input:     "shared",
			wantCount: 4,
			wantSet:   templateSet,
		},
		{
			name:      "ui has 4 templates",
			input:     "ui",
			wantCount: 4,
			wantSet:   templateSet,
		},
		{
			name:      "utility has 0 templates",
			input:     "utility",
			wantCount: 0,
			wantSet:   []string{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			descriptor, err := GetModuleType(tc.input)
			if err != nil {
				t.Fatalf("GetModuleType(%q) error = %v", tc.input, err)
			}
			if len(descriptor.TemplateSet) != tc.wantCount {
				t.Fatalf("GetModuleType(%q) template count = %d, want %d", tc.input, len(descriptor.TemplateSet), tc.wantCount)
			}
			if !reflect.DeepEqual(descriptor.TemplateSet, tc.wantSet) {
				t.Fatalf("GetModuleType(%q) templates = %#v, want %#v", tc.input, descriptor.TemplateSet, tc.wantSet)
			}
		})
	}
}

func TestGetModuleTypeTemplateSetIsCloned(t *testing.T) {
	t.Parallel()

	first, err := GetModuleType("feature")
	if err != nil {
		t.Fatalf("GetModuleType(feature) error = %v", err)
	}
	first.TemplateSet[0] = "mutated"

	second, err := GetModuleType("feature")
	if err != nil {
		t.Fatalf("GetModuleType(feature) error = %v", err)
	}
	if second.TemplateSet[0] != "namespace" {
		t.Fatalf("template registry mutated unexpectedly, first template = %q, want %q", second.TemplateSet[0], "namespace")
	}
}

func expectedDescriptors() map[ModuleType]ModuleTypeDescriptor {
	templateSet := []string{"namespace", "module", "interface", "impl"}
	return map[ModuleType]ModuleTypeDescriptor{
		ModuleTypeFeature: {
			Type:                  ModuleTypeFeature,
			HasInterfaceImplSplit: true,
			HasRelux:              true,
			HasUI:                 true,
			TemplateSet:           templateSet,
			Description:           "Full module with UI (interface + implementation split).",
		},
		ModuleTypeKit: {
			Type:                  ModuleTypeKit,
			HasInterfaceImplSplit: true,
			HasRelux:              true,
			HasUI:                 false,
			TemplateSet:           templateSet,
			Description:           "Logic module without UI (interface + implementation split).",
		},
		ModuleTypeShared: {
			Type:                  ModuleTypeShared,
			HasInterfaceImplSplit: true,
			HasRelux:              true,
			HasUI:                 false,
			TemplateSet:           templateSet,
			Description:           "Shared services module (interface + implementation split).",
		},
		ModuleTypeUI: {
			Type:                  ModuleTypeUI,
			HasInterfaceImplSplit: true,
			HasRelux:              true,
			HasUI:                 true,
			TemplateSet:           templateSet,
			Description:           "SwiftUI component module (interface + implementation split).",
		},
		ModuleTypeUtility: {
			Type:                  ModuleTypeUtility,
			HasInterfaceImplSplit: false,
			HasRelux:              false,
			HasUI:                 false,
			TemplateSet:           []string{},
			Description:           "Utility-only module (single package, no templates).",
		},
	}
}
