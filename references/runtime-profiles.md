# Runtime profiles

`runtime_profiles` separates immutable distribution artifacts from the backend environment selected at runtime. The generator owns the typed Swift descriptors, Tuist build configurations and schemes, application configuration field, and the policy-aware AppConfig templates.

Use the complete generic example at [`tuist-starter/testdata/runtime-profiles-config.json`](../tuist-starter/testdata/runtime-profiles-config.json). The machine-readable block schema is [`runtime-profiles.schema.json`](runtime-profiles.schema.json).

## Model

The schema has two independent typed axes:

- `distribution_profiles`: `pilotTestFlight`, `appStore`, `internal`, and `tests` describe an artifact and its runtime policy.
- `backend_environments`: `production`, `staging`, `development`, and `fixture` describe exact backend realms and public client-registration metadata.

A distribution profile never contains an API URL. It refers to backend environment identifiers through `default_environment` and `allowed_environments`. A backend descriptor contains one exact `api_origin`; the generator never appends a path such as `/api/v1`.

`test_action` maps the `tests` profile to one or more existing Tuist unit/UI test target names. `targets` is required and must not be empty; invalid, empty, or duplicate Swift target identifiers fail validation rather than generating an unusable empty Test action. Optional `launch_arguments` are static, non-secret flags applied only to the generated Tests action. They are not inherited by Run, Profile, or Archive actions.

```json
{
  "runtime_profiles": {
    "schema_version": 1,
    "test_action": {
      "targets": [
        "DemoAppTests",
        "DemoAppUITests"
      ],
      "launch_arguments": [
        "--demo-hosted-tests"
      ]
    }
  }
}
```

The mapped targets must exist in the root Tuist project. For generated targets, use the same names with `ios-app-manager test-targets setup --unit-target DemoAppTests --ui-target DemoAppUITests --yes` before running `tuist generate`.

## Approved policy matrix

| Profile | Build kind | Default | Allowed | Menu | Selection persistence | Non-production marker | Ephemeral injection |
|---|---|---|---|---|---|---|---|
| `pilotTestFlight` | `release` | `production` | exactly `production`, `staging` | `visible` | `enabled` | `persistent` | `forbidden` |
| `appStore` | `release` | `production` | exactly `production` | `hidden` | `disabled` | `none` | `forbidden` |
| `internal` | `debug` or `release` | `staging` | a subset of `development`, `staging`, `production` that includes `staging` | visible when more than one environment is allowed | configurable | `persistent` | `forbidden` |
| `tests` | `debug` | `fixture` | exactly `fixture` | `hidden` | `disabled` | `persistent` | `allowed` |

The `tests` profile accepts an explicit ephemeral `BackendEnvironmentDescriptor` only when its typed identifier is `fixture`. It never loads or writes a persisted environment choice, so a production selection cannot carry into tests.

Every profile also declares a unique `build_configuration` name. These names become Tuist configurations and shared schemes; they are independent from the typed profile identifiers.

## Backend descriptors

Each configured environment declares:

| Field | Contract |
|---|---|
| `api_origin` | Absolute origin only and unique across environments. Credentials, non-root paths, query strings, and fragments are rejected. HTTPS is required except that `fixture` may use HTTP on a loopback host. Ownership comparison lowercases scheme/hostname and normalizes the effective numeric port, so omitted and explicit default ports collide; generated Swift preserves the exact configured origin. |
| `auth_namespace` | Non-empty, path-safe authentication realm identifier; unique across environments. |
| `storage_namespace` | Non-empty, path-safe client storage realm identifier; unique across environments. |
| `grant_namespace` | Non-empty, path-safe access-grant realm identifier; unique across environments. |
| `quota_namespace` | Non-empty, path-safe quota realm identifier; unique across environments. |
| `firebase` | Required outside `fixture`; contains public registration metadata and a validation hook name. |

An environment allowed by any profile must have a descriptor. The generated `BackendEnvironment` enum remains exhaustive even when optional `development` is not enabled in an internal profile.

## Firebase public-client inputs

The persisted Firebase block is deliberately limited to:

- `project_id`
- `google_app_id`
- `bundle_id`
- `resource_name`
- `validation_input_environment_variable`
- optional `identity_sharing_group`

`bundle_id` must match the project bundle identifier. `resource_name` is a filename such as `GoogleService-Info-staging.plist`, never a path. Secret- or path-bearing fields such as `api_key`, `credential`, and `plist_path` are rejected by the config loader.

### Explicit identity sharing

Firebase identifiers stay environment-scoped and duplicate project IDs, Google App IDs, resource names, and validation hooks are rejected by default. Two or more non-fixture backend environments may intentionally reuse one public-client identity only when every participant declares the same lowercase kebab-case `identity_sharing_group` and these values match exactly across the whole group:

- `project_id`
- `google_app_id`
- `bundle_id`
- `resource_name`
- `validation_input_environment_variable`

Partial matches, different groups, an undeclared participant, a singleton group, conflicting group metadata, and fixture participation fail validation with the involved environments and fields. API origins and auth, storage, grant, and quota namespaces remain unique per backend environment.

The generated `FirebaseIdentitySharingGroup` value exposes this public-client trust boundary to Swift consumers. It does not merge backend realms or runtime state. Firebase tokens identify the configured Firebase project/app and carry no backend-environment claim, so the active `BackendEnvironmentDescriptor` still chooses the trusted identity and isolates API, auth, storage, grant, and quota state through its own exact origin and namespaces.

The value of `validation_input_environment_variable` is the name of an operator-provided process environment variable. Its value points to a local XML plist only for the current command:

```bash
export IOS_APP_MANAGER_FIREBASE_SHARED_PLIST="$PWD/.local/firebase/shared.plist"
export IOS_APP_MANAGER_FIREBASE_DEVELOPMENT_PLIST="$PWD/.local/firebase/development.plist"

ios-app-manager generate runtime-profiles
```

The validation hook checks that the plist contains `PROJECT_ID`, `GOOGLE_APP_ID`, `BUNDLE_ID`, and `API_KEY`, then compares only the configured public identifiers. The generator does not serialize, copy, print, or retain the local path or API key. Supplying the runtime resource to the application bundle remains an explicit build/deployment responsibility under the declared public `resource_name`.

## Generated output

`ios-app-manager init` and `generate runtime-profiles` converge these owned slices:

- `Targets/<AppName>/Sources/Configuration/Runtime/RuntimeProfiles.swift`
  - typed distribution and backend enums;
  - policy and backend descriptor types;
  - deterministic dictionaries in canonical order;
  - exact `URL` origins, public Firebase metadata, and an optional typed identity-sharing group that does not alter environment-specific namespaces.
- `Tuist/ProjectDescriptionHelpers/RuntimeProfiles.swift`
  - typed profile-to-configuration mapping;
  - debug/release configurations with `DISTRIBUTION_PROFILE` settings;
  - one shared scheme per distribution profile;
  - a non-empty Tests action containing the configured testables and optional hosted-test launch flags.
- `Project.swift`
  - managed configuration and scheme blocks;
  - replacement of the legacy app scheme when its Debug/Release actions no longer name generated configurations; unrelated custom schemes remain intact;
  - project configuration settings;
  - the `ApplicationConfiguration.distributionProfile` build setting.
- `Package.swift`
  - the same configurations for Tuist-generated package projects, replacing an existing `PackageSettings` configurations argument instead of adding a duplicate.
- `SharedConfig/Sources/ApplicationConfiguration.swift`
  - typed reading of the selected distribution profile.

When `app-config setup` runs with `runtime_profiles`, its managed templates use the generated allowlist, default, menu, persistence, marker, and ephemeral-injection policy. Existing handwritten AppConfig manager behavior is not overwritten: the command returns an actionable merge error unless the file is an unchanged legacy template or a managed runtime-profile template.

## Commands and convergence

Initialize a project with validation inputs present:

```bash
ios-app-manager init --config ios-app-manager.json --output .
ios-app-manager test-targets setup --unit-target DemoAppTests --ui-target DemoAppUITests --yes
tuist install
tuist generate --no-open
```

Regenerate only the runtime domain:

```bash
ios-app-manager generate runtime-profiles
```

Regenerate the full project configuration and the AppConfig consumer:

```bash
ios-app-manager secure-store setup --access-group <group> --yes
ios-app-manager generate project-config
ios-app-manager app-config setup --yes
```

On a mature project with an existing custom Registry, SecureStore and AppConfig add only their owned imports, registrations, and builders. They do not require or invoke full Registry regeneration.

Rerunning either generator with unchanged config produces no file changes. Changes to allowed environments, configuration names, public metadata, or descriptors replace the managed output rather than appending another variant.

## Removal and compatibility

To remove runtime profiles, delete the `runtime_profiles` block and run:

```bash
ios-app-manager generate project-config
ios-app-manager app-config setup --yes
```

Managed Swift and Tuist runtime files are removed, managed manifest blocks return to the legacy `configurations` list, `distributionProfile` is removed from application configuration, and managed AppConfig templates return to legacy behavior.

Configs without `runtime_profiles` retain the previous `Debug`/`Release` and AppConfig behavior. Nested runtime-profile configs require explicit `test_action.targets`. The deprecated top-level `distribution_profiles` and `backend_environments` maps are read as a version-1 runtime block, receive conventional `<AppName>Tests` and `<AppName>UITests` mappings for compatibility, and converge to nested `runtime_profiles` the next time `WriteProjectConfig` writes the file. A config cannot combine the legacy aliases with the nested block.
