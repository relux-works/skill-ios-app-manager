package config

import (
	"strings"
	"testing"
)

func TestProjectConfigValidateValid(t *testing.T) {
	t.Parallel()

	cfg := validProjectConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProjectConfigValidateMissingRequiredFieldsAllReturned(t *testing.T) {
	t.Parallel()

	cfg := ProjectConfig{
		AppName:          " ",
		BundleID:         "",
		TeamID:           "",
		SwiftVersion:     "",
		MinTarget:        "",
		MarketingVersion: "",
		ProjectVersion:   "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation errors")
	}

	msg := err.Error()
	required := []string{
		"AppName is required",
		"BundleID is required",
		"TeamID is required",
		"SwiftVersion is required",
		"MinTarget is required",
		"MarketingVersion is required",
		"ProjectVersion is required",
	}
	for _, want := range required {
		if !strings.Contains(msg, want) {
			t.Fatalf("Validate() error = %q, want %q", msg, want)
		}
	}

	validationErr, ok := err.(*ValidationErrors)
	if !ok {
		t.Fatalf("Validate() error type = %T, want *ValidationErrors", err)
	}
	if got, want := len(validationErr.Issues), len(required); got != want {
		t.Fatalf("len(ValidationErrors.Issues) = %d, want %d", got, want)
	}
}

func TestProjectConfigValidateInvalidFormatsAllReturned(t *testing.T) {
	t.Parallel()

	cfg := validProjectConfig()
	cfg.BundleID = "invalid_bundle_id"
	cfg.SwiftVersion = "6"
	cfg.MinTarget = "17"
	cfg.ProjectSettings.Swift.LanguageMode = "6"
	cfg.ProjectSettings.Swift.StrictMemorySafety = "later"
	cfg.ProjectSettings.Swift.Concurrency.StrictChecking = "hard"
	cfg.ProjectSettings.Swift.Concurrency.MemberImportVisibility = "later"
	cfg.ProjectSettings.Swift.Concurrency.ExistentialAny = "never"
	cfg.SharedConfig.ModuleName = "Shared-Config"
	cfg.AppGroups = []string{"", "group.com.example.demo"}
	cfg.Configurations = []string{"Debug", ""}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation errors")
	}

	msg := err.Error()
	wantIssues := []string{
		"BundleID must use reverse-domain format",
		"SwiftVersion must use major.minor format",
		"MinTarget must use major.minor format",
		"ProjectSettings.Swift.LanguageMode must use SwiftPM format",
		"ProjectSettings.Swift.StrictMemorySafety must be yes, migrate, or no",
		"ProjectSettings.Swift.Concurrency.StrictChecking must be minimal, targeted, or complete",
		"ProjectSettings.Swift.Concurrency.MemberImportVisibility must be yes, migrate, or no",
		"ProjectSettings.Swift.Concurrency.ExistentialAny must be yes, migrate, or no",
		"SharedConfig.ModuleName must be a valid Swift module identifier",
		"AppGroups[0] must not be empty",
		"Configurations[1] must not be empty",
	}
	for _, want := range wantIssues {
		if !strings.Contains(msg, want) {
			t.Fatalf("Validate() error = %q, want %q", msg, want)
		}
	}
}
