package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRuntimeProfilesApprovedMatrixValidates(t *testing.T) {
	t.Parallel()

	cfg := validRuntimeProjectConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if got, want := cfg.OrderedDistributionProfiles(), AllDistributionProfiles(); !reflect.DeepEqual(got, want) {
		t.Fatalf("OrderedDistributionProfiles() = %#v, want %#v", got, want)
	}
	if got, want := cfg.OrderedBackendEnvironments(), AllBackendEnvironments(); !reflect.DeepEqual(got, want) {
		t.Fatalf("OrderedBackendEnvironments() = %#v, want %#v", got, want)
	}
}

func TestLoadRuntimeProfilesGenericExample(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "runtime-profiles-config.json")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", path, err)
	}
	if !cfg.HasRuntimeProfiles() {
		t.Fatal("HasRuntimeProfiles() = false, want true")
	}
	if got := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight].AllowedEnvironments; !reflect.DeepEqual(got, []BackendEnvironment{BackendEnvironmentProduction, BackendEnvironmentStaging}) {
		t.Fatalf("pilot allowed environments = %#v", got)
	}
	if got := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileTests].SelectionPersistence; got != SelectionPersistenceDisabled {
		t.Fatalf("tests selection persistence = %q, want disabled", got)
	}
}

func TestRuntimeProfilesJSONSchemaIsWellFormed(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "..", "references", "runtime-profiles.schema.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(payload, &schema); err != nil {
		t.Fatalf("runtime profile schema is invalid JSON: %v", err)
	}
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("$schema = %#v, want draft 2020-12", schema["$schema"])
	}
}

func TestRuntimeProfilesNormalizeAllowedEnvironmentOrder(t *testing.T) {
	t.Parallel()

	cfg := validRuntimeProjectConfig()
	internal := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileInternal]
	internal.AllowedEnvironments = []BackendEnvironment{
		BackendEnvironmentStaging,
		BackendEnvironmentProduction,
		BackendEnvironmentDevelopment,
	}
	cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileInternal] = internal
	cfg.applyDefaults()

	want := []BackendEnvironment{
		BackendEnvironmentProduction,
		BackendEnvironmentStaging,
		BackendEnvironmentDevelopment,
	}
	got := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileInternal].AllowedEnvironments
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllowedEnvironments = %#v, want canonical %#v", got, want)
	}
}

func TestRuntimeProfilesRejectForbiddenProfileCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*ProjectConfig)
		wantErr string
	}{
		{
			name: "pilot development",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight]
				profile.AllowedEnvironments = append(profile.AllowedEnvironments, BackendEnvironmentDevelopment)
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight] = profile
			},
			wantErr: "must allow exactly production plus staging",
		},
		{
			name: "pilot missing staging",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight]
				profile.AllowedEnvironments = []BackendEnvironment{BackendEnvironmentProduction}
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight] = profile
			},
			wantErr: "must allow exactly production plus staging",
		},
		{
			name: "pilot persistence disabled",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight]
				profile.SelectionPersistence = SelectionPersistenceDisabled
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight] = profile
			},
			wantErr: "must persist an allowed environment selection",
		},
		{
			name: "app store staging",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileAppStore]
				profile.AllowedEnvironments = append(profile.AllowedEnvironments, BackendEnvironmentStaging)
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileAppStore] = profile
			},
			wantErr: "allow only production",
		},
		{
			name: "internal fixture",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileInternal]
				profile.AllowedEnvironments = append(profile.AllowedEnvironments, BackendEnvironmentFixture)
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileInternal] = profile
			},
			wantErr: "may allow only development, staging, and production",
		},
		{
			name: "tests persist production",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileTests]
				profile.AllowedEnvironments = []BackendEnvironment{BackendEnvironmentFixture, BackendEnvironmentProduction}
				profile.SelectionPersistence = SelectionPersistenceEnabled
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfileTests] = profile
			},
			wantErr: "disable persistence",
		},
		{
			name: "pilot debug build",
			mutate: func(cfg *ProjectConfig) {
				profile := cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight]
				profile.BuildKind = BuildConfigurationDebug
				cfg.RuntimeProfiles.DistributionProfiles[DistributionProfilePilotTestFlight] = profile
			},
			wantErr: "Release-like",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := validRuntimeProjectConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRuntimeProfilesRejectInvalidEnvironmentDescriptors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*ProjectConfig)
		wantErr string
	}{
		{
			name: "shipping http",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
				environment.APIOrigin = "http://api.example.com"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction] = environment
			},
			wantErr: "must use HTTPS outside fixture",
		},
		{
			name: "origin path",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				environment.APIOrigin = "https://staging-api.example.com/api/v1"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = environment
			},
			wantErr: "exact origin without credentials, path, query, or fragment",
		},
		{
			name: "namespace collision",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				environment.StorageNamespace = "runtime-production"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = environment
			},
			wantErr: "collides with production",
		},
		{
			name: "firebase bundle mismatch",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
				environment.Firebase.BundleID = "com.example.other"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction] = environment
			},
			wantErr: "BundleID must match the project BundleID",
		},
		{
			name: "firebase hook collision",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				environment.Firebase.ValidationInputEnvironmentVar = "FIREBASE_PRODUCTION_PLIST"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = environment
			},
			wantErr: "ValidationInputEnvironmentVariable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := validRuntimeProjectConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigMigratesLegacyRuntimeProfileMaps(t *testing.T) {
	t.Parallel()

	cfg := validRuntimeProjectConfig()
	payload := struct {
		ProjectConfig
		RuntimeProfiles *RuntimeProfilesConfig `json:"runtime_profiles,omitempty"`
	}{ProjectConfig: cfg}
	payload.RuntimeProfiles = nil
	payload.ProjectConfig.RuntimeProfiles = nil
	payload.ProjectConfig.LegacyDistributionProfiles = cfg.RuntimeProfiles.DistributionProfiles
	payload.ProjectConfig.LegacyBackendEnvironments = cfg.RuntimeProfiles.BackendEnvironments

	raw, err := json.Marshal(payload.ProjectConfig)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.RuntimeProfiles == nil {
		t.Fatal("RuntimeProfiles = nil, want migrated legacy maps")
	}
	if loaded.RuntimeProfiles.SchemaVersion != RuntimeProfilesSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.RuntimeProfiles.SchemaVersion, RuntimeProfilesSchemaVersion)
	}
	if loaded.LegacyDistributionProfiles != nil || loaded.LegacyBackendEnvironments != nil {
		t.Fatalf("legacy maps were not cleared: %#v %#v", loaded.LegacyDistributionProfiles, loaded.LegacyBackendEnvironments)
	}
}

func TestWriteProjectConfigConvergesLegacyRuntimeProfileMaps(t *testing.T) {
	t.Parallel()

	runtime := validRuntimeProjectConfig().RuntimeProfiles
	cfg := validProjectConfig()
	cfg.LegacyDistributionProfiles = runtime.DistributionProfiles
	cfg.LegacyBackendEnvironments = runtime.BackendEnvironments

	path := filepath.Join(t.TempDir(), "config.json")
	if err := WriteProjectConfig(path, cfg); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, `"runtime_profiles"`) {
		t.Fatalf("written config missing runtime_profiles:\n%s", content)
	}
	if strings.Contains(content, `\n  "distribution_profiles"`) || strings.Contains(content, `\n  "backend_environments"`) {
		t.Fatalf("written config retained deprecated top-level aliases:\n%s", content)
	}
}

func TestLoadConfigWithoutRuntimeProfilesRemainsBackwardCompatible(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := WriteProjectConfig(path, validProjectConfig()); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.RuntimeProfiles != nil || loaded.HasRuntimeProfiles() {
		t.Fatalf("RuntimeProfiles = %#v, want nil for legacy config", loaded.RuntimeProfiles)
	}
	if !reflect.DeepEqual(loaded.Configurations, []string{"Debug", "Release"}) {
		t.Fatalf("Configurations = %#v, want legacy Debug/Release", loaded.Configurations)
	}
}

func TestFirebaseClientConfigRejectsSecretOrPathFields(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"api_key", "plist_path", "credential"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			var firebase FirebaseClientConfig
			err := json.Unmarshal([]byte(`{"project_id":"example-prod","`+field+`":"must-not-persist"}`), &firebase)
			if err == nil || !strings.Contains(err.Error(), "persist only public metadata") {
				t.Fatalf("json.Unmarshal() error = %v, want safe-field rejection", err)
			}
		})
	}
}

func validRuntimeProjectConfig() ProjectConfig {
	cfg := validProjectConfig()
	cfg.RuntimeProfiles = &RuntimeProfilesConfig{
		SchemaVersion: RuntimeProfilesSchemaVersion,
		DistributionProfiles: map[DistributionProfile]DistributionProfileConfig{
			DistributionProfilePilotTestFlight: {
				BuildConfiguration:   "PilotTestFlight",
				BuildKind:            BuildConfigurationRelease,
				DefaultEnvironment:   BackendEnvironmentProduction,
				AllowedEnvironments:  []BackendEnvironment{BackendEnvironmentProduction, BackendEnvironmentStaging},
				EnvironmentMenu:      EnvironmentMenuVisible,
				SelectionPersistence: SelectionPersistenceEnabled,
				NonProductionMarker:  NonProductionMarkerPersistent,
				EphemeralInjection:   EphemeralInjectionForbidden,
			},
			DistributionProfileAppStore: {
				BuildConfiguration:   "AppStore",
				BuildKind:            BuildConfigurationRelease,
				DefaultEnvironment:   BackendEnvironmentProduction,
				AllowedEnvironments:  []BackendEnvironment{BackendEnvironmentProduction},
				EnvironmentMenu:      EnvironmentMenuHidden,
				SelectionPersistence: SelectionPersistenceDisabled,
				NonProductionMarker:  NonProductionMarkerNone,
				EphemeralInjection:   EphemeralInjectionForbidden,
			},
			DistributionProfileInternal: {
				BuildConfiguration:   "Internal",
				BuildKind:            BuildConfigurationDebug,
				DefaultEnvironment:   BackendEnvironmentStaging,
				AllowedEnvironments:  []BackendEnvironment{BackendEnvironmentProduction, BackendEnvironmentStaging, BackendEnvironmentDevelopment},
				EnvironmentMenu:      EnvironmentMenuVisible,
				SelectionPersistence: SelectionPersistenceEnabled,
				NonProductionMarker:  NonProductionMarkerPersistent,
				EphemeralInjection:   EphemeralInjectionForbidden,
			},
			DistributionProfileTests: {
				BuildConfiguration:   "Tests",
				BuildKind:            BuildConfigurationDebug,
				DefaultEnvironment:   BackendEnvironmentFixture,
				AllowedEnvironments:  []BackendEnvironment{BackendEnvironmentFixture},
				EnvironmentMenu:      EnvironmentMenuHidden,
				SelectionPersistence: SelectionPersistenceDisabled,
				NonProductionMarker:  NonProductionMarkerPersistent,
				EphemeralInjection:   EphemeralInjectionAllowed,
			},
		},
		BackendEnvironments: map[BackendEnvironment]BackendEnvironmentConfig{
			BackendEnvironmentProduction:  runtimeBackendEnvironment("production", "https://api.example.com", "FIREBASE_PRODUCTION_PLIST"),
			BackendEnvironmentStaging:     runtimeBackendEnvironment("staging", "https://staging-api.example.com", "FIREBASE_STAGING_PLIST"),
			BackendEnvironmentDevelopment: runtimeBackendEnvironment("development", "https://development-api.example.com", "FIREBASE_DEVELOPMENT_PLIST"),
			BackendEnvironmentFixture: {
				APIOrigin:        "http://127.0.0.1:8080",
				AuthNamespace:    "auth-fixture",
				StorageNamespace: "runtime-fixture",
				GrantNamespace:   "grants-fixture",
				QuotaNamespace:   "quota-fixture",
			},
		},
	}
	return cfg
}

func runtimeBackendEnvironment(name string, origin string, inputEnvironmentVariable string) BackendEnvironmentConfig {
	return BackendEnvironmentConfig{
		APIOrigin:        origin,
		AuthNamespace:    "auth-" + name,
		StorageNamespace: "runtime-" + name,
		GrantNamespace:   "grants-" + name,
		QuotaNamespace:   "quota-" + name,
		Firebase: &FirebaseClientConfig{
			ProjectID:                     "example-" + name,
			GoogleAppID:                   "1:example-" + name + ":ios:abc123",
			BundleID:                      "com.example.demo",
			ResourceName:                  "GoogleService-Info-" + name + ".plist",
			ValidationInputEnvironmentVar: inputEnvironmentVariable,
		},
	}
}
