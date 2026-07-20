package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const RuntimeProfilesSchemaVersion = 1

// DistributionProfile is an immutable artifact/release profile. It is
// deliberately independent from BackendEnvironment, which is selected at
// runtime within the profile's allowlist.
type DistributionProfile string

const (
	DistributionProfilePilotTestFlight DistributionProfile = "pilotTestFlight"
	DistributionProfileAppStore        DistributionProfile = "appStore"
	DistributionProfileInternal        DistributionProfile = "internal"
	DistributionProfileTests           DistributionProfile = "tests"
)

// BackendEnvironment identifies one persistent backend realm. Arbitrary
// environment identifiers are rejected so generated Swift remains exhaustive.
type BackendEnvironment string

const (
	BackendEnvironmentProduction  BackendEnvironment = "production"
	BackendEnvironmentStaging     BackendEnvironment = "staging"
	BackendEnvironmentDevelopment BackendEnvironment = "development"
	BackendEnvironmentFixture     BackendEnvironment = "fixture"
)

// FirebaseIdentitySharingGroup names an explicit Firebase public-client trust
// boundary. It never changes backend API or runtime-state namespaces.
type FirebaseIdentitySharingGroup string

type BuildConfigurationKind string

const (
	BuildConfigurationDebug   BuildConfigurationKind = "debug"
	BuildConfigurationRelease BuildConfigurationKind = "release"
)

type EnvironmentMenuPolicy string

const (
	EnvironmentMenuHidden  EnvironmentMenuPolicy = "hidden"
	EnvironmentMenuVisible EnvironmentMenuPolicy = "visible"
)

type SelectionPersistencePolicy string

const (
	SelectionPersistenceDisabled SelectionPersistencePolicy = "disabled"
	SelectionPersistenceEnabled  SelectionPersistencePolicy = "enabled"
)

type NonProductionMarkerPolicy string

const (
	NonProductionMarkerNone       NonProductionMarkerPolicy = "none"
	NonProductionMarkerPersistent NonProductionMarkerPolicy = "persistent"
)

type EphemeralInjectionPolicy string

const (
	EphemeralInjectionForbidden EphemeralInjectionPolicy = "forbidden"
	EphemeralInjectionAllowed   EphemeralInjectionPolicy = "allowed"
)

var (
	allDistributionProfiles = []DistributionProfile{
		DistributionProfilePilotTestFlight,
		DistributionProfileAppStore,
		DistributionProfileInternal,
		DistributionProfileTests,
	}
	allBackendEnvironments = []BackendEnvironment{
		BackendEnvironmentProduction,
		BackendEnvironmentStaging,
		BackendEnvironmentDevelopment,
		BackendEnvironmentFixture,
	}
	firebaseProjectIDPattern            = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	googleAppIDPattern                  = regexp.MustCompile(`^\d+:[A-Za-z0-9-]+:ios:[A-Za-z0-9]+$`)
	environmentVariablePattern          = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	firebaseIdentitySharingGroupPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
)

type RuntimeProfilesConfig struct {
	SchemaVersion        int                                               `json:"schema_version,omitempty"`
	TestAction           RuntimeProfileTestActionConfig                    `json:"test_action"`
	DistributionProfiles map[DistributionProfile]DistributionProfileConfig `json:"distribution_profiles"`
	BackendEnvironments  map[BackendEnvironment]BackendEnvironmentConfig   `json:"backend_environments"`
}

// RuntimeProfileTestActionConfig maps the Tests distribution profile to
// concrete Tuist test targets and optional non-secret launch flags.
type RuntimeProfileTestActionConfig struct {
	Targets         []string `json:"targets"`
	LaunchArguments []string `json:"launch_arguments,omitempty"`
}

type DistributionProfileConfig struct {
	BuildConfiguration   string                     `json:"build_configuration"`
	BuildKind            BuildConfigurationKind     `json:"build_kind"`
	DefaultEnvironment   BackendEnvironment         `json:"default_environment"`
	AllowedEnvironments  []BackendEnvironment       `json:"allowed_environments"`
	EnvironmentMenu      EnvironmentMenuPolicy      `json:"environment_menu"`
	SelectionPersistence SelectionPersistencePolicy `json:"selection_persistence"`
	NonProductionMarker  NonProductionMarkerPolicy  `json:"non_production_marker"`
	EphemeralInjection   EphemeralInjectionPolicy   `json:"ephemeral_injection"`
}

type BackendEnvironmentConfig struct {
	APIOrigin        string                `json:"api_origin"`
	AuthNamespace    string                `json:"auth_namespace"`
	StorageNamespace string                `json:"storage_namespace"`
	GrantNamespace   string                `json:"grant_namespace"`
	QuotaNamespace   string                `json:"quota_namespace"`
	Firebase         *FirebaseClientConfig `json:"firebase,omitempty"`
}

// FirebaseClientConfig contains public registration metadata and a safe hook
// name. The hook value is an environment variable whose value points at the
// operator-supplied plist; neither that path nor the plist is serialized.
type FirebaseClientConfig struct {
	ProjectID                     string                       `json:"project_id"`
	GoogleAppID                   string                       `json:"google_app_id"`
	BundleID                      string                       `json:"bundle_id"`
	ResourceName                  string                       `json:"resource_name"`
	ValidationInputEnvironmentVar string                       `json:"validation_input_environment_variable"`
	IdentitySharingGroup          FirebaseIdentitySharingGroup `json:"identity_sharing_group,omitempty"`
}

// UnmarshalJSON rejects accidental secret/path material in Firebase config
// instead of silently ignoring it through encoding/json's default behavior.
func (c *FirebaseClientConfig) UnmarshalJSON(data []byte) error {
	type alias FirebaseClientConfig
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	allowed := map[string]struct{}{
		"project_id":                            {},
		"google_app_id":                         {},
		"bundle_id":                             {},
		"resource_name":                         {},
		"validation_input_environment_variable": {},
		"identity_sharing_group":                {},
	}
	for field := range fields {
		if _, ok := allowed[field]; ok {
			continue
		}
		return fmt.Errorf("unsupported Firebase client field %q; persist only public metadata and the validation hook name", field)
	}

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = FirebaseClientConfig(decoded)
	return nil
}

func (c *ProjectConfig) applyRuntimeProfileDefaults() {
	legacyConfigured := len(c.LegacyDistributionProfiles) > 0 || len(c.LegacyBackendEnvironments) > 0
	if c.RuntimeProfiles == nil && legacyConfigured {
		c.RuntimeProfiles = &RuntimeProfilesConfig{
			SchemaVersion: RuntimeProfilesSchemaVersion,
			TestAction: RuntimeProfileTestActionConfig{
				Targets: []string{strings.TrimSpace(c.AppName) + "Tests", strings.TrimSpace(c.AppName) + "UITests"},
			},
			DistributionProfiles: c.LegacyDistributionProfiles,
			BackendEnvironments:  c.LegacyBackendEnvironments,
		}
		c.LegacyDistributionProfiles = nil
		c.LegacyBackendEnvironments = nil
	}
	if c.RuntimeProfiles == nil {
		return
	}

	if c.RuntimeProfiles.SchemaVersion == 0 {
		c.RuntimeProfiles.SchemaVersion = RuntimeProfilesSchemaVersion
	}
	for index, target := range c.RuntimeProfiles.TestAction.Targets {
		c.RuntimeProfiles.TestAction.Targets[index] = strings.TrimSpace(target)
	}
	for index, argument := range c.RuntimeProfiles.TestAction.LaunchArguments {
		c.RuntimeProfiles.TestAction.LaunchArguments[index] = strings.TrimSpace(argument)
	}

	for profile, raw := range c.RuntimeProfiles.DistributionProfiles {
		raw.BuildConfiguration = strings.TrimSpace(raw.BuildConfiguration)
		raw.BuildKind = BuildConfigurationKind(strings.ToLower(strings.TrimSpace(string(raw.BuildKind))))
		raw.DefaultEnvironment = BackendEnvironment(strings.TrimSpace(string(raw.DefaultEnvironment)))
		raw.EnvironmentMenu = EnvironmentMenuPolicy(strings.ToLower(strings.TrimSpace(string(raw.EnvironmentMenu))))
		raw.SelectionPersistence = SelectionPersistencePolicy(strings.ToLower(strings.TrimSpace(string(raw.SelectionPersistence))))
		raw.NonProductionMarker = NonProductionMarkerPolicy(strings.ToLower(strings.TrimSpace(string(raw.NonProductionMarker))))
		raw.EphemeralInjection = EphemeralInjectionPolicy(strings.ToLower(strings.TrimSpace(string(raw.EphemeralInjection))))
		for index, environment := range raw.AllowedEnvironments {
			raw.AllowedEnvironments[index] = BackendEnvironment(strings.TrimSpace(string(environment)))
		}
		sort.SliceStable(raw.AllowedEnvironments, func(i, j int) bool {
			return backendEnvironmentOrder(raw.AllowedEnvironments[i]) < backendEnvironmentOrder(raw.AllowedEnvironments[j])
		})
		c.RuntimeProfiles.DistributionProfiles[profile] = raw
	}

	for environment, raw := range c.RuntimeProfiles.BackendEnvironments {
		raw.APIOrigin = strings.TrimSpace(raw.APIOrigin)
		raw.AuthNamespace = strings.TrimSpace(raw.AuthNamespace)
		raw.StorageNamespace = strings.TrimSpace(raw.StorageNamespace)
		raw.GrantNamespace = strings.TrimSpace(raw.GrantNamespace)
		raw.QuotaNamespace = strings.TrimSpace(raw.QuotaNamespace)
		if raw.Firebase != nil {
			raw.Firebase.ProjectID = strings.TrimSpace(raw.Firebase.ProjectID)
			raw.Firebase.GoogleAppID = strings.TrimSpace(raw.Firebase.GoogleAppID)
			raw.Firebase.BundleID = strings.TrimSpace(raw.Firebase.BundleID)
			raw.Firebase.ResourceName = strings.TrimSpace(raw.Firebase.ResourceName)
			raw.Firebase.ValidationInputEnvironmentVar = strings.TrimSpace(raw.Firebase.ValidationInputEnvironmentVar)
			raw.Firebase.IdentitySharingGroup = FirebaseIdentitySharingGroup(strings.TrimSpace(string(raw.Firebase.IdentitySharingGroup)))
		}
		c.RuntimeProfiles.BackendEnvironments[environment] = raw
	}
}

func (c ProjectConfig) HasRuntimeProfiles() bool {
	return c.RuntimeProfiles != nil
}

func AllDistributionProfiles() []DistributionProfile {
	return append([]DistributionProfile(nil), allDistributionProfiles...)
}

func AllBackendEnvironments() []BackendEnvironment {
	return append([]BackendEnvironment(nil), allBackendEnvironments...)
}

func (c ProjectConfig) OrderedDistributionProfiles() []DistributionProfile {
	if c.RuntimeProfiles == nil {
		return nil
	}
	result := make([]DistributionProfile, 0, len(c.RuntimeProfiles.DistributionProfiles))
	for _, profile := range allDistributionProfiles {
		if _, ok := c.RuntimeProfiles.DistributionProfiles[profile]; ok {
			result = append(result, profile)
		}
	}
	return result
}

func (c ProjectConfig) OrderedBackendEnvironments() []BackendEnvironment {
	if c.RuntimeProfiles == nil {
		return nil
	}
	result := make([]BackendEnvironment, 0, len(c.RuntimeProfiles.BackendEnvironments))
	for _, environment := range allBackendEnvironments {
		if _, ok := c.RuntimeProfiles.BackendEnvironments[environment]; ok {
			result = append(result, environment)
		}
	}
	return result
}

func validateRuntimeProfiles(c ProjectConfig, issues *[]string) {
	legacyConfigured := len(c.LegacyDistributionProfiles) > 0 || len(c.LegacyBackendEnvironments) > 0
	if c.RuntimeProfiles != nil && legacyConfigured {
		*issues = append(*issues, "RuntimeProfiles cannot be combined with deprecated top-level distribution_profiles/backend_environments")
	}
	if c.RuntimeProfiles == nil {
		return
	}

	runtime := c.RuntimeProfiles
	if runtime.SchemaVersion != RuntimeProfilesSchemaVersion {
		*issues = append(*issues, fmt.Sprintf("RuntimeProfiles.SchemaVersion must be %d", RuntimeProfilesSchemaVersion))
	}
	validateRuntimeProfileTestAction(runtime.TestAction, issues)

	for profile := range runtime.DistributionProfiles {
		if !isDistributionProfile(profile) {
			*issues = append(*issues, fmt.Sprintf("RuntimeProfiles.DistributionProfiles contains unsupported profile %q", profile))
		}
	}
	for _, profile := range allDistributionProfiles {
		profileConfig, ok := runtime.DistributionProfiles[profile]
		if !ok {
			*issues = append(*issues, fmt.Sprintf("RuntimeProfiles.DistributionProfiles.%s is required", profile))
			continue
		}
		validateDistributionProfile(profile, profileConfig, runtime.BackendEnvironments, issues)
	}

	for environment := range runtime.BackendEnvironments {
		if !isBackendEnvironment(environment) {
			*issues = append(*issues, fmt.Sprintf("RuntimeProfiles.BackendEnvironments contains unsupported environment %q", environment))
		}
	}
	validateBackendEnvironments(c.BundleID, runtime.BackendEnvironments, issues)
	validateUniqueBuildConfigurations(runtime.DistributionProfiles, issues)
}

func validateRuntimeProfileTestAction(testAction RuntimeProfileTestActionConfig, issues *[]string) {
	const field = "RuntimeProfiles.TestAction"
	if len(testAction.Targets) == 0 {
		*issues = append(*issues, field+".Targets must not be empty")
	}

	seenTargets := make(map[string]struct{}, len(testAction.Targets))
	for index, rawTarget := range testAction.Targets {
		target := strings.TrimSpace(rawTarget)
		if target == "" {
			*issues = append(*issues, fmt.Sprintf("%s.Targets[%d] must not be empty", field, index))
			continue
		}
		if strings.ContainsAny(target, "\x00\r\n") {
			*issues = append(*issues, fmt.Sprintf("%s.Targets[%d] must be a single-line value", field, index))
		}
		if _, exists := seenTargets[target]; exists {
			*issues = append(*issues, fmt.Sprintf("%s.Targets contains duplicate target %q", field, target))
			continue
		}
		seenTargets[target] = struct{}{}
	}

	seenArguments := make(map[string]struct{}, len(testAction.LaunchArguments))
	for index, rawArgument := range testAction.LaunchArguments {
		argument := strings.TrimSpace(rawArgument)
		if argument == "" {
			*issues = append(*issues, fmt.Sprintf("%s.LaunchArguments[%d] must not be empty", field, index))
			continue
		}
		if strings.ContainsAny(argument, "\x00\r\n") {
			*issues = append(*issues, fmt.Sprintf("%s.LaunchArguments[%d] must be a single-line value", field, index))
		}
		if _, exists := seenArguments[argument]; exists {
			*issues = append(*issues, fmt.Sprintf("%s.LaunchArguments contains duplicate argument %q", field, argument))
			continue
		}
		seenArguments[argument] = struct{}{}
	}
}

func validateDistributionProfile(
	profile DistributionProfile,
	profileConfig DistributionProfileConfig,
	environments map[BackendEnvironment]BackendEnvironmentConfig,
	issues *[]string,
) {
	field := "RuntimeProfiles.DistributionProfiles." + string(profile)
	if profileConfig.BuildConfiguration == "" {
		*issues = append(*issues, field+".BuildConfiguration is required")
	}
	if profileConfig.BuildKind != BuildConfigurationDebug && profileConfig.BuildKind != BuildConfigurationRelease {
		*issues = append(*issues, field+".BuildKind must be debug or release")
	}
	if !isBackendEnvironment(profileConfig.DefaultEnvironment) {
		*issues = append(*issues, field+".DefaultEnvironment must be production, staging, development, or fixture")
	}
	if len(profileConfig.AllowedEnvironments) == 0 {
		*issues = append(*issues, field+".AllowedEnvironments must not be empty")
	}
	seen := make(map[BackendEnvironment]struct{}, len(profileConfig.AllowedEnvironments))
	defaultAllowed := false
	for index, environment := range profileConfig.AllowedEnvironments {
		if !isBackendEnvironment(environment) {
			*issues = append(*issues, fmt.Sprintf("%s.AllowedEnvironments[%d] contains unsupported environment %q", field, index, environment))
			continue
		}
		if _, ok := seen[environment]; ok {
			*issues = append(*issues, fmt.Sprintf("%s.AllowedEnvironments contains duplicate environment %q", field, environment))
			continue
		}
		seen[environment] = struct{}{}
		if environment == profileConfig.DefaultEnvironment {
			defaultAllowed = true
		}
		if _, ok := environments[environment]; !ok {
			*issues = append(*issues, fmt.Sprintf("%s allows %q but no backend descriptor is configured", field, environment))
		}
	}
	if !defaultAllowed {
		*issues = append(*issues, field+".DefaultEnvironment must be present in AllowedEnvironments")
	}
	if profileConfig.EnvironmentMenu != EnvironmentMenuHidden && profileConfig.EnvironmentMenu != EnvironmentMenuVisible {
		*issues = append(*issues, field+".EnvironmentMenu must be hidden or visible")
	}
	if profileConfig.SelectionPersistence != SelectionPersistenceDisabled && profileConfig.SelectionPersistence != SelectionPersistenceEnabled {
		*issues = append(*issues, field+".SelectionPersistence must be disabled or enabled")
	}
	if profileConfig.NonProductionMarker != NonProductionMarkerNone && profileConfig.NonProductionMarker != NonProductionMarkerPersistent {
		*issues = append(*issues, field+".NonProductionMarker must be none or persistent")
	}
	if profileConfig.EphemeralInjection != EphemeralInjectionForbidden && profileConfig.EphemeralInjection != EphemeralInjectionAllowed {
		*issues = append(*issues, field+".EphemeralInjection must be forbidden or allowed")
	}

	validateApprovedProfileBoundary(profile, profileConfig, seen, issues)
}

func validateApprovedProfileBoundary(
	profile DistributionProfile,
	profileConfig DistributionProfileConfig,
	allowed map[BackendEnvironment]struct{},
	issues *[]string,
) {
	field := "RuntimeProfiles.DistributionProfiles." + string(profile)
	containsOnly := func(values ...BackendEnvironment) bool {
		permitted := make(map[BackendEnvironment]struct{}, len(values))
		for _, value := range values {
			permitted[value] = struct{}{}
		}
		for value := range allowed {
			if _, ok := permitted[value]; !ok {
				return false
			}
		}
		return true
	}

	switch profile {
	case DistributionProfilePilotTestFlight:
		if profileConfig.BuildKind != BuildConfigurationRelease {
			*issues = append(*issues, field+" must be Release-like")
		}
		if profileConfig.DefaultEnvironment != BackendEnvironmentProduction {
			*issues = append(*issues, field+" must default to production")
		}
		_, hasProduction := allowed[BackendEnvironmentProduction]
		_, hasStaging := allowed[BackendEnvironmentStaging]
		if !hasProduction || !hasStaging || len(allowed) != 2 || !containsOnly(BackendEnvironmentProduction, BackendEnvironmentStaging) {
			*issues = append(*issues, field+" must allow exactly production plus staging")
		}
		if profileConfig.EnvironmentMenu != EnvironmentMenuVisible {
			*issues = append(*issues, field+" must expose the environment menu")
		}
		if profileConfig.SelectionPersistence != SelectionPersistenceEnabled {
			*issues = append(*issues, field+" must persist an allowed environment selection")
		}
		if profileConfig.NonProductionMarker != NonProductionMarkerPersistent {
			*issues = append(*issues, field+" must require a persistent marker for staging")
		}
		if profileConfig.EphemeralInjection != EphemeralInjectionForbidden {
			*issues = append(*issues, field+" must forbid ephemeral injection")
		}
	case DistributionProfileAppStore:
		if profileConfig.BuildKind != BuildConfigurationRelease ||
			profileConfig.DefaultEnvironment != BackendEnvironmentProduction ||
			len(allowed) != 1 {
			*issues = append(*issues, field+" must be Release-like and allow only production")
		}
		if _, ok := allowed[BackendEnvironmentProduction]; !ok {
			*issues = append(*issues, field+" must allow production")
		}
		if profileConfig.EnvironmentMenu != EnvironmentMenuHidden ||
			profileConfig.SelectionPersistence != SelectionPersistenceDisabled ||
			profileConfig.NonProductionMarker != NonProductionMarkerNone ||
			profileConfig.EphemeralInjection != EphemeralInjectionForbidden {
			*issues = append(*issues, field+" must hide the menu, disable selection persistence, omit the non-production marker, and forbid ephemeral injection")
		}
	case DistributionProfileInternal:
		if profileConfig.DefaultEnvironment != BackendEnvironmentStaging {
			*issues = append(*issues, field+" must default to staging")
		}
		if _, ok := allowed[BackendEnvironmentStaging]; !ok || !containsOnly(BackendEnvironmentDevelopment, BackendEnvironmentStaging, BackendEnvironmentProduction) {
			*issues = append(*issues, field+" may allow only development, staging, and production and must include staging")
		}
		if len(allowed) > 1 && profileConfig.EnvironmentMenu != EnvironmentMenuVisible {
			*issues = append(*issues, field+" must expose the environment menu when multiple environments are allowed")
		}
		if profileConfig.NonProductionMarker != NonProductionMarkerPersistent {
			*issues = append(*issues, field+" must require a persistent non-production marker")
		}
		if profileConfig.EphemeralInjection != EphemeralInjectionForbidden {
			*issues = append(*issues, field+" must forbid ephemeral injection")
		}
	case DistributionProfileTests:
		if profileConfig.BuildKind != BuildConfigurationDebug ||
			profileConfig.DefaultEnvironment != BackendEnvironmentFixture ||
			len(allowed) != 1 {
			*issues = append(*issues, field+" must be Debug-like and use only fixture by default")
		}
		if _, ok := allowed[BackendEnvironmentFixture]; !ok {
			*issues = append(*issues, field+" must allow fixture")
		}
		if profileConfig.EnvironmentMenu != EnvironmentMenuHidden ||
			profileConfig.SelectionPersistence != SelectionPersistenceDisabled ||
			profileConfig.NonProductionMarker != NonProductionMarkerPersistent ||
			profileConfig.EphemeralInjection != EphemeralInjectionAllowed {
			*issues = append(*issues, field+" must hide the menu, disable persistence, require a persistent marker, and allow explicit ephemeral injection")
		}
	}
}

func validateBackendEnvironments(
	appBundleID string,
	environments map[BackendEnvironment]BackendEnvironmentConfig,
	issues *[]string,
) {
	apiOriginOwners := map[string]BackendEnvironment{}
	namespaceOwners := map[string]map[string]BackendEnvironment{
		"AuthNamespace":    {},
		"StorageNamespace": {},
		"GrantNamespace":   {},
		"QuotaNamespace":   {},
	}
	firebaseOwners := map[string]map[string]firebaseIdentityParticipant{
		"ProjectID":                          {},
		"GoogleAppID":                        {},
		"ResourceName":                       {},
		"ValidationInputEnvironmentVariable": {},
	}
	firebaseParticipants := make([]firebaseIdentityParticipant, 0, len(environments))
	for _, environment := range allBackendEnvironments {
		descriptor, ok := environments[environment]
		if !ok {
			continue
		}
		field := "RuntimeProfiles.BackendEnvironments." + string(environment)
		validateAPIOrigin(environment, descriptor.APIOrigin, field+".APIOrigin", issues)
		validateEnvironmentScopedValue(field+".APIOrigin", canonicalAPIOrigin(descriptor.APIOrigin), environment, apiOriginOwners, issues)
		validateNamespace(field+".AuthNamespace", descriptor.AuthNamespace, environment, namespaceOwners["AuthNamespace"], issues)
		validateNamespace(field+".StorageNamespace", descriptor.StorageNamespace, environment, namespaceOwners["StorageNamespace"], issues)
		validateNamespace(field+".GrantNamespace", descriptor.GrantNamespace, environment, namespaceOwners["GrantNamespace"], issues)
		validateNamespace(field+".QuotaNamespace", descriptor.QuotaNamespace, environment, namespaceOwners["QuotaNamespace"], issues)

		if environment == BackendEnvironmentFixture && descriptor.Firebase == nil {
			continue
		}
		if descriptor.Firebase == nil {
			*issues = append(*issues, field+".Firebase is required outside fixture")
			continue
		}
		validateFirebaseClientConfig(appBundleID, field+".Firebase", *descriptor.Firebase, issues)
		if environment == BackendEnvironmentFixture && descriptor.Firebase.IdentitySharingGroup != "" {
			*issues = append(*issues, field+".Firebase.IdentitySharingGroup is forbidden for fixture")
		}
		participant := firebaseIdentityParticipant{Environment: environment, Firebase: *descriptor.Firebase}
		firebaseParticipants = append(firebaseParticipants, participant)
		validateFirebaseEnvironmentScopedValue(field+".Firebase.ProjectID", descriptor.Firebase.ProjectID, participant, firebaseOwners["ProjectID"], issues)
		validateFirebaseEnvironmentScopedValue(field+".Firebase.GoogleAppID", descriptor.Firebase.GoogleAppID, participant, firebaseOwners["GoogleAppID"], issues)
		validateFirebaseEnvironmentScopedValue(field+".Firebase.ResourceName", descriptor.Firebase.ResourceName, participant, firebaseOwners["ResourceName"], issues)
		validateFirebaseEnvironmentScopedValue(field+".Firebase.ValidationInputEnvironmentVariable", descriptor.Firebase.ValidationInputEnvironmentVar, participant, firebaseOwners["ValidationInputEnvironmentVariable"], issues)
	}
	validateFirebaseIdentitySharingGroups(firebaseParticipants, issues)
}

func validateAPIOrigin(environment BackendEnvironment, raw string, field string, issues *[]string) {
	if raw == "" {
		*issues = append(*issues, field+" is required")
		return
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		*issues = append(*issues, field+" must be an absolute API origin")
		return
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		*issues = append(*issues, field+" must be an exact origin without credentials, path, query, or fragment")
	}
	if environment == BackendEnvironmentFixture {
		if parsed.Scheme == "https" {
			return
		}
		host := parsed.Hostname()
		if parsed.Scheme != "http" || (host != "localhost" && net.ParseIP(host) == nil) || !isLoopbackHost(host) {
			*issues = append(*issues, field+" must use HTTPS or an HTTP loopback host for fixture")
		}
		return
	}
	if parsed.Scheme != "https" {
		*issues = append(*issues, field+" must use HTTPS outside fixture")
	}
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func canonicalAPIOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimSpace(raw)
	}

	scheme := strings.ToLower(parsed.Scheme)
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if numericPort, err := strconv.ParseUint(port, 10, 64); err == nil {
		port = strconv.FormatUint(numericPort, 10)
	}
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}

	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return scheme + "://" + host
}

func validateNamespace(field string, value string, environment BackendEnvironment, owners map[string]BackendEnvironment, issues *[]string) {
	if value == "" {
		*issues = append(*issues, field+" is required")
		return
	}
	if strings.ContainsAny(value, "\r\n\t /\\") {
		*issues = append(*issues, field+" must be a single path-safe namespace token")
	}
	if owner, exists := owners[value]; exists && owner != environment {
		*issues = append(*issues, fmt.Sprintf("%s %q collides with %s", field, value, owner))
		return
	}
	owners[value] = environment
}

func validateEnvironmentScopedValue(field string, value string, environment BackendEnvironment, owners map[string]BackendEnvironment, issues *[]string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if owner, exists := owners[value]; exists && owner != environment {
		*issues = append(*issues, fmt.Sprintf("%s %q collides with %s", field, value, owner))
		return
	}
	owners[value] = environment
}

type firebaseIdentityParticipant struct {
	Environment BackendEnvironment
	Firebase    FirebaseClientConfig
}

func validateFirebaseEnvironmentScopedValue(
	field string,
	value string,
	participant firebaseIdentityParticipant,
	owners map[string]firebaseIdentityParticipant,
	issues *[]string,
) {
	if strings.TrimSpace(value) == "" {
		return
	}
	owner, exists := owners[value]
	if !exists {
		owners[value] = participant
		return
	}
	if owner.Environment == participant.Environment {
		return
	}
	if firebaseIdentityGroupsMatch(owner.Firebase, participant.Firebase) {
		return
	}
	*issues = append(*issues, fmt.Sprintf(
		"%s %q collides with %s; duplicate Firebase public identity values require every participant to declare the same non-empty identity_sharing_group and all public registration metadata to match exactly",
		field,
		value,
		owner.Environment,
	))
}

func firebaseIdentityGroupsMatch(left FirebaseClientConfig, right FirebaseClientConfig) bool {
	return left.IdentitySharingGroup != "" &&
		left.IdentitySharingGroup == right.IdentitySharingGroup
}

func validateFirebaseIdentitySharingGroups(participants []firebaseIdentityParticipant, issues *[]string) {
	groups := make(map[FirebaseIdentitySharingGroup][]firebaseIdentityParticipant)
	for _, participant := range participants {
		group := participant.Firebase.IdentitySharingGroup
		if group == "" || participant.Environment == BackendEnvironmentFixture {
			continue
		}
		groups[group] = append(groups[group], participant)
	}

	groupNames := make([]string, 0, len(groups))
	for group := range groups {
		groupNames = append(groupNames, string(group))
	}
	sort.Strings(groupNames)
	for _, rawGroup := range groupNames {
		group := FirebaseIdentitySharingGroup(rawGroup)
		members := groups[group]
		if len(members) < 2 {
			*issues = append(*issues, fmt.Sprintf(
				"RuntimeProfiles Firebase identity_sharing_group %q must include at least two non-fixture backend environments; found only %s",
				group,
				members[0].Environment,
			))
			continue
		}
		canonical := members[0]
		for _, member := range members[1:] {
			differences := firebaseIdentityDifferences(canonical.Firebase, member.Firebase)
			if len(differences) == 0 {
				continue
			}
			*issues = append(*issues, fmt.Sprintf(
				"RuntimeProfiles Firebase identity_sharing_group %q has conflicting public registration metadata between %s and %s: %s must match exactly",
				group,
				canonical.Environment,
				member.Environment,
				strings.Join(differences, ", "),
			))
		}
	}
}

func firebaseIdentityDifferences(left FirebaseClientConfig, right FirebaseClientConfig) []string {
	differences := make([]string, 0, 5)
	if left.ProjectID != right.ProjectID {
		differences = append(differences, "project_id")
	}
	if left.GoogleAppID != right.GoogleAppID {
		differences = append(differences, "google_app_id")
	}
	if left.BundleID != right.BundleID {
		differences = append(differences, "bundle_id")
	}
	if left.ResourceName != right.ResourceName {
		differences = append(differences, "resource_name")
	}
	if left.ValidationInputEnvironmentVar != right.ValidationInputEnvironmentVar {
		differences = append(differences, "validation_input_environment_variable")
	}
	return differences
}

func validateFirebaseClientConfig(appBundleID string, field string, firebase FirebaseClientConfig, issues *[]string) {
	if firebase.ProjectID == "" || !firebaseProjectIDPattern.MatchString(firebase.ProjectID) {
		*issues = append(*issues, field+".ProjectID must be a Firebase project identifier")
	}
	if firebase.GoogleAppID == "" || !googleAppIDPattern.MatchString(firebase.GoogleAppID) {
		*issues = append(*issues, field+".GoogleAppID must be an iOS Google App ID")
	}
	if firebase.BundleID == "" || !bundleIDPattern.MatchString(firebase.BundleID) {
		*issues = append(*issues, field+".BundleID must use reverse-domain format")
	} else if strings.TrimSpace(appBundleID) != "" && firebase.BundleID != strings.TrimSpace(appBundleID) {
		*issues = append(*issues, field+".BundleID must match the project BundleID")
	}
	if firebase.ResourceName == "" || filepath.Base(firebase.ResourceName) != firebase.ResourceName || filepath.Ext(firebase.ResourceName) != ".plist" {
		*issues = append(*issues, field+".ResourceName must be a plist resource file name without a path")
	}
	if !environmentVariablePattern.MatchString(firebase.ValidationInputEnvironmentVar) {
		*issues = append(*issues, field+".ValidationInputEnvironmentVariable must be an uppercase environment variable name")
	}
	if firebase.IdentitySharingGroup != "" && !firebaseIdentitySharingGroupPattern.MatchString(string(firebase.IdentitySharingGroup)) {
		*issues = append(*issues, field+".IdentitySharingGroup must be a lowercase kebab-case identifier")
	}
}

func validateUniqueBuildConfigurations(profiles map[DistributionProfile]DistributionProfileConfig, issues *[]string) {
	owners := make(map[string]DistributionProfile, len(profiles))
	for _, profile := range allDistributionProfiles {
		profileConfig, ok := profiles[profile]
		if !ok || profileConfig.BuildConfiguration == "" {
			continue
		}
		key := strings.ToLower(profileConfig.BuildConfiguration)
		if owner, exists := owners[key]; exists {
			*issues = append(*issues, fmt.Sprintf("RuntimeProfiles build configuration %q is shared by %s and %s", profileConfig.BuildConfiguration, owner, profile))
			continue
		}
		owners[key] = profile
	}
}

func isDistributionProfile(value DistributionProfile) bool {
	for _, allowed := range allDistributionProfiles {
		if value == allowed {
			return true
		}
	}
	return false
}

func isBackendEnvironment(value BackendEnvironment) bool {
	for _, allowed := range allBackendEnvironments {
		if value == allowed {
			return true
		}
	}
	return false
}

func backendEnvironmentOrder(value BackendEnvironment) int {
	for index, environment := range allBackendEnvironments {
		if value == environment {
			return index
		}
	}
	return len(allBackendEnvironments)
}
