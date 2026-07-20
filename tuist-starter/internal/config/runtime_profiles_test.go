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
	production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
	staging := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
	if production.Firebase.IdentitySharingGroup == "" || production.Firebase.IdentitySharingGroup != staging.Firebase.IdentitySharingGroup {
		t.Fatalf("generic example sharing groups = production %q staging %q, want one explicit shared group", production.Firebase.IdentitySharingGroup, staging.Firebase.IdentitySharingGroup)
	}
	if differences := firebaseIdentityDifferences(*production.Firebase, *staging.Firebase); len(differences) != 0 {
		t.Fatalf("generic example shared Firebase metadata differs: %v", differences)
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
	if !strings.Contains(string(payload), `"identity_sharing_group"`) {
		t.Fatal("runtime profile schema does not expose identity_sharing_group")
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
			name: "API origin collision",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				environment.APIOrigin = "https://api.example.com/"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = environment
			},
			wantErr: "APIOrigin \"https://api.example.com\" collides with production",
		},
		{
			name: "API origin default HTTPS port collision",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				environment.APIOrigin = "https://api.example.com:443"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = environment
			},
			wantErr: "APIOrigin \"https://api.example.com\" collides with production",
		},
		{
			name: "API origin zero-padded default HTTPS port collision",
			mutate: func(cfg *ProjectConfig) {
				environment := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				environment.APIOrigin = "https://api.example.com:0443"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = environment
			},
			wantErr: "APIOrigin \"https://api.example.com\" collides with production",
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

func TestRuntimeProfilesFirebaseDuplicatesRemainRejectedWithoutSharingGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(staging *FirebaseClientConfig, production FirebaseClientConfig)
		field  string
	}{
		{
			name: "project ID",
			mutate: func(staging *FirebaseClientConfig, production FirebaseClientConfig) {
				staging.ProjectID = production.ProjectID
			},
			field: "ProjectID",
		},
		{
			name: "Google App ID",
			mutate: func(staging *FirebaseClientConfig, production FirebaseClientConfig) {
				staging.GoogleAppID = production.GoogleAppID
			},
			field: "GoogleAppID",
		},
		{
			name: "resource name",
			mutate: func(staging *FirebaseClientConfig, production FirebaseClientConfig) {
				staging.ResourceName = production.ResourceName
			},
			field: "ResourceName",
		},
		{
			name: "validation hook",
			mutate: func(staging *FirebaseClientConfig, production FirebaseClientConfig) {
				staging.ValidationInputEnvironmentVar = production.ValidationInputEnvironmentVar
			},
			field: "ValidationInputEnvironmentVariable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := validRuntimeProjectConfig()
			production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
			staging := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
			tt.mutate(staging.Firebase, *production.Firebase)
			cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = staging

			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.field) || !strings.Contains(err.Error(), "identity_sharing_group") {
				t.Fatalf("Validate() error = %v, want fail-closed %s collision with sharing guidance", err, tt.field)
			}
		})
	}
}

func TestRuntimeProfilesAllowsExplicitSharedFirebaseIdentity(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		targets []BackendEnvironment
	}{
		{name: "two participants", targets: []BackendEnvironment{BackendEnvironmentStaging}},
		{name: "three participants", targets: []BackendEnvironment{BackendEnvironmentStaging, BackendEnvironmentDevelopment}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := validRuntimeProjectConfig()
			for _, target := range tt.targets {
				shareFirebaseIdentity(&cfg, BackendEnvironmentProduction, target, "shared-public-client")
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() shared Firebase identity error = %v", err)
			}

			production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
			for _, targetEnvironment := range tt.targets {
				target := cfg.RuntimeProfiles.BackendEnvironments[targetEnvironment]
				if production.APIOrigin == target.APIOrigin {
					t.Fatalf("shared identity API origins = %q, want environment-specific origins", production.APIOrigin)
				}
				for name, values := range map[string][2]string{
					"auth":    {production.AuthNamespace, target.AuthNamespace},
					"storage": {production.StorageNamespace, target.StorageNamespace},
					"grant":   {production.GrantNamespace, target.GrantNamespace},
					"quota":   {production.QuotaNamespace, target.QuotaNamespace},
				} {
					if values[0] == values[1] {
						t.Fatalf("shared identity %s namespaces = %q, want environment-specific namespaces", name, values[0])
					}
				}
			}
		})
	}
}

func TestCanonicalAPIOriginNormalizesEffectivePorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "HTTPS omitted port", raw: "https://API.Example.COM", want: "https://api.example.com"},
		{name: "HTTPS uppercase scheme", raw: "HTTPS://API.Example.COM", want: "https://api.example.com"},
		{name: "HTTPS default port", raw: "https://API.Example.COM:443", want: "https://api.example.com"},
		{name: "HTTPS zero-padded default port", raw: "https://API.Example.COM:0443", want: "https://api.example.com"},
		{name: "HTTPS zero-padded non-default port", raw: "https://API.Example.COM:08443", want: "https://api.example.com:8443"},
		{name: "HTTP omitted port", raw: "http://127.0.0.1", want: "http://127.0.0.1"},
		{name: "HTTP default port", raw: "http://127.0.0.1:80", want: "http://127.0.0.1"},
		{name: "HTTP zero-padded default port", raw: "http://127.0.0.1:0080", want: "http://127.0.0.1"},
		{name: "IPv6 HTTPS default port", raw: "https://[::1]:0443", want: "https://[::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := canonicalAPIOrigin(tt.raw); got != tt.want {
				t.Fatalf("canonicalAPIOrigin(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRuntimeProfilesRejectsInvalidFirebaseIdentitySharing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*ProjectConfig)
		wantErr string
	}{
		{
			name: "partial metadata match",
			mutate: func(cfg *ProjectConfig) {
				production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
				staging := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				production.Firebase.IdentitySharingGroup = "shared-public-client"
				staging.Firebase.IdentitySharingGroup = "shared-public-client"
				staging.Firebase.ProjectID = production.Firebase.ProjectID
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction] = production
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = staging
			},
			wantErr: "conflicting public registration metadata",
		},
		{
			name: "cross-group collision",
			mutate: func(cfg *ProjectConfig) {
				shareFirebaseIdentity(cfg, BackendEnvironmentProduction, BackendEnvironmentStaging, "group-one")
				staging := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				staging.Firebase.IdentitySharingGroup = "group-two"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = staging
			},
			wantErr: "same non-empty identity_sharing_group",
		},
		{
			name: "missing participant declaration",
			mutate: func(cfg *ProjectConfig) {
				shareFirebaseIdentity(cfg, BackendEnvironmentProduction, BackendEnvironmentStaging, "shared-public-client")
				staging := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				staging.Firebase.IdentitySharingGroup = ""
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = staging
			},
			wantErr: "every participant",
		},
		{
			name: "conflicting metadata within group",
			mutate: func(cfg *ProjectConfig) {
				shareFirebaseIdentity(cfg, BackendEnvironmentProduction, BackendEnvironmentStaging, "shared-public-client")
				staging := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging]
				staging.Firebase.ResourceName = "GoogleService-Info-conflict.plist"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentStaging] = staging
			},
			wantErr: "resource_name must match exactly",
		},
		{
			name: "singleton declaration",
			mutate: func(cfg *ProjectConfig) {
				production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
				production.Firebase.IdentitySharingGroup = "shared-public-client"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction] = production
			},
			wantErr: "must include at least two non-fixture backend environments",
		},
		{
			name: "invalid group identifier",
			mutate: func(cfg *ProjectConfig) {
				production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
				production.Firebase.IdentitySharingGroup = "Shared Group"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction] = production
			},
			wantErr: "must be a lowercase kebab-case identifier",
		},
		{
			name: "fixture participant",
			mutate: func(cfg *ProjectConfig) {
				production := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction]
				production.Firebase.IdentitySharingGroup = "shared-public-client"
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentProduction] = production
				fixture := cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentFixture]
				firebase := *production.Firebase
				fixture.Firebase = &firebase
				cfg.RuntimeProfiles.BackendEnvironments[BackendEnvironmentFixture] = fixture
			},
			wantErr: "IdentitySharingGroup is forbidden for fixture",
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

func shareFirebaseIdentity(
	cfg *ProjectConfig,
	sourceEnvironment BackendEnvironment,
	targetEnvironment BackendEnvironment,
	group FirebaseIdentitySharingGroup,
) {
	source := cfg.RuntimeProfiles.BackendEnvironments[sourceEnvironment]
	target := cfg.RuntimeProfiles.BackendEnvironments[targetEnvironment]
	shared := *source.Firebase
	shared.IdentitySharingGroup = group
	source.Firebase.IdentitySharingGroup = group
	target.Firebase = &shared
	cfg.RuntimeProfiles.BackendEnvironments[sourceEnvironment] = source
	cfg.RuntimeProfiles.BackendEnvironments[targetEnvironment] = target
}
