# skill-ios-app-manager

Skill + Go CLI tool that scaffolds and manages Tuist-based iOS projects with Relux state management architecture.

## Structure

```
skill-ios-app-manager/
├── SKILL.md              # Skill definition
├── blueprints/           # Versioned example blueprints for relux-feature modules
├── references/           # CLI, DSL, workflow docs
├── diagrams/             # Architecture diagrams
├── tuist-starter/        # Go CLI tool source
│   ├── cmd/, internal/, pkg/
│   ├── go.mod, Makefile
│   └── testdata/
└── .temp/                # Generated demo projects (gitignored)
```

## Skill Setup

```bash
git clone git@github.com:relux-works/skill-ios-app-manager.git
cd skill-ios-app-manager
./scripts/setup.sh
```

Setup behavior:
- Builds the CLI in `tuist-starter/`
- Copies the skill runtime into `~/.agents/skills/ios-app-manager`
- Degitizes that installed copy after sync
- Symlinks `~/.claude/skills/ios-app-manager` and `~/.codex/skills/ios-app-manager` to the installed copy
- Symlinks `~/.local/bin/ios-app-manager` to the built binary

## Build & Run

```bash
cd tuist-starter
make setup    # create local Go cache dirs
make build    # build binary -> ./ios-app-manager
make test     # run all tests
make lint     # go vet
```

## Generate plugins

`generate` is plugin-based. Each `generate <artifact>` entrypoint is a separate scaffold generator with its own responsibility and dependency contract.

Current generators:
- `generate project-config`
- `generate makefile`
- `generate swiftlint`
- `generate versions`
- `generate min-target`
- `generate team-id`
- `generate app-capabilities`
- `generate build-flags`
- `generate package-strictness`

`generate project-config` is the orchestration entrypoint for manifest config sync. Today it runs:
- `generate versions` — syncs `marketing_version` and `project_version`
- `generate min-target` — syncs `min_target` into `deploymentTargets` and `IPHONEOS_DEPLOYMENT_TARGET`
- `generate team-id` — syncs `team_id` into `developmentTeam` constants and `DEVELOPMENT_TEAM` build settings
- `generate app-capabilities` — syncs host app capabilities from config
- `generate build-flags` — syncs app/extension Swift language, strict memory safety, and concurrency settings from `project_settings.swift`
- `generate package-strictness` — syncs root/module `Package.swift` strictness from the same `project_settings.swift`

These generators depend on the `init` scaffold shape. `versions`, `min-target`, `team-id`, and `build-flags` update the host app `Project.swift` plus every `Extensions/*/Project.swift`. `app-capabilities` syncs capability-owned manifest slices. `package-strictness` updates root `Package.swift` plus every module `Packages/*/Package.swift`.

Swift strictness is config-driven. Declare it in `ios-app-manager.json` under `project_settings.swift`; if you omit that block, defaults are derived from `swift_version` and the scaffold's current strict baseline.

Example:

```json
{
  "project_settings": {
    "swift": {
      "language_mode": "v6",
      "strict_memory_safety": "yes",
      "concurrency": {
        "approachable": false,
        "default_actor_isolation": "nonisolated",
        "strict_checking": "complete",
        "concise_magic_file": true,
        "disable_outward_actor_isolation": true,
        "global_actor_isolated_types_usability": true,
        "infer_isolated_conformances": true,
        "infer_sendable_from_captures": true,
        "global_concurrency": true,
        "member_import_visibility": "yes",
        "nonfrozen_enum_exhaustivity": true,
        "region_based_isolation": true,
        "existential_any": "yes",
        "nonisolated_nonsending_by_default": true
      }
    }
  }
}
```

Generated Makefiles use `tuist generate --no-open` by default. To auto-open Xcode explicitly, run `tuist generate --open` yourself or override the generated Makefile call with `make generate TUIST_GENERATE_FLAGS=--open`.

Generated `make build` and `make test` run `tuist generate` before `xcodebuild` and reuse `make clean-package-artifacts` on exit. The cleanup hook is a no-op by default (`PACKAGE_ARTIFACT_CLEANUP_CMD ?= :`); projects that need checkout-specific cleanup should override that variable in the Makefile custom section. Build uses a generic iOS Simulator destination by default, while test keeps the configured concrete simulator destination unless overridden.

Recommended config sync flow:

```bash
# 1. bump project config in ios-app-manager.json
$EDITOR ios-app-manager.json

# 2. sync host app + extension + package manifests
./ios-app-manager generate project-config

# 3. regenerate Tuist project artifacts
tuist generate --no-open
```

If you only want one slice, the leaf plugins still work directly:

```bash
./ios-app-manager generate versions
./ios-app-manager generate min-target
./ios-app-manager generate team-id
./ios-app-manager generate app-capabilities
./ios-app-manager generate build-flags
./ios-app-manager generate package-strictness
```

### App Groups capability

`generate app-capabilities` currently runs the `app-groups` subplugin. Configure it with `app_groups` and, optionally, `shared_config.module_name` in `ios-app-manager.json`:

```json
{
  "bundle_id": "com.example.demo.app",
  "app_groups": [
    "group.com.example.demo.app.shared",
    "group.com.example.demo.app.sso"
  ],
  "modules_path": "Packages",
  "shared_config": {
    "module_name": "SharedConfig"
  }
}
```

Generated output:
- `Tuist/ProjectDescriptionHelpers/AppCapabilities.swift` gets `.appGroups(...)` declarations used by generated entitlements.
- Every generated target Info.plist slice gets a single `AppGroups` dictionary, not root-level `APP_GROUP_*` keys.
- `${modules_path}/${shared_config.module_name}` is generated as a shared package and linked to app/test targets.
- `Configuration+AppGroups.swift` reads typed values through that shared package instead of hardcoding identifiers.

The Info.plist dictionary key is generated from the app group identifier relative to `bundle_id`:
- `group.<bundle_id>` maps to `main`.
- `group.<bundle_id>.<suffix>` maps from `<suffix>`.
- Other `group.*` values drop only the `group.` prefix.
- The remaining value is sanitized to a lowerCamelCase Swift/dictionary key; collisions fail validation.

For the config above, the generated Info.plist shape is:

```swift
"AppGroups": .dictionary([
    "shared": .string("group.com.example.demo.app.shared"),
    "sso": .string("group.com.example.demo.app.sso"),
])
```

The same generated keys are used by `SharedConfig` reader APIs such as `DemoAppGroups.read(from:)` and `DemoAppGroupSlot.dictionaryKey`.

## What it does

Generates a complete iOS project structure from a JSON config file:

- **Tuist manifests**: Project.swift, Package.swift, Workspace.swift, Tuist.swift
- **IoC container**: SwiftIoC-based Registry.swift with auto-discovery
- **Relux state management**: Actions, Effects, State, Flow scaffolding
- **Foundation modules**: SecureStore (Keychain), TokenProvider, Utilities, FoundationPlus, SwiftUIPlus
- **Widget extensions**: WidgetKit (static + interactive), Live Activity + Dynamic Island, App Intents
- **Network**: HttpClient IoC registration with swift-httpclient
- **AppConfig**: Environment switching with ApiConfigurator

## Pipeline

```bash
cd tuist-starter
./ios-app-manager init
./ios-app-manager ioc setup
./ios-app-manager relux setup
./ios-app-manager secure-store setup --access-group <group>
./ios-app-manager token-provider setup
./ios-app-manager utilities setup
./ios-app-manager foundation-plus setup
./ios-app-manager swiftui-plus setup
./ios-app-manager app-extensions setup
./ios-app-manager widget-base setup
./ios-app-manager app-intents setup
./ios-app-manager static-widget setup
./ios-app-manager live-activity setup
./ios-app-manager module create --from <name>.blueprint.json
./ios-app-manager app-config setup
./ios-app-manager http-client setup
```

Order matters -- each command depends on prerequisites from earlier steps.

## Dynamic product policy

- Scaffold-generated local package products are emitted as dynamic libraries by default.
- External Swift package products added through setup plugins, module blueprints, or `dep add-external` are automatically forced to `.framework` in root `Package.swift` via Tuist `PackageSettings.productTypes`. Pass `--product <name>` when the Swift product name differs from the package name, `--app-target` when the product must be linked into the host app target, and `--target-setting KEY=VALUE` when a product needs a Tuist `PackageSettings.targetSettings` override.
- Removing an external dependency through `dep remove-external` also removes the matching framework override.
- If a remote package is added by hand outside `ios-app-manager`, this policy does not apply automatically.

## Blueprint examples

Versioned relux-feature blueprints live under [blueprints/](blueprints/README.md).
Current examples come from the XFlow iOS migration surface:

- [Auth.blueprint.json](blueprints/xflow-ios/Auth.blueprint.json)
- [CRM.blueprint.json](blueprints/xflow-ios/CRM.blueprint.json)
- [Onboarding.blueprint.json](blueprints/xflow-ios/Onboarding.blueprint.json)
- [Organizations.blueprint.json](blueprints/xflow-ios/Organizations.blueprint.json)
- [Profile.blueprint.json](blueprints/xflow-ios/Profile.blueprint.json)

## Module types

| Type | Description |
|------|-------------|
| `feature` | UI module with interface/impl split |
| `relux-feature` | Blueprint-only: full Relux business logic (Business/Data/UI layers) |
| `kit` | Business logic library with interface/impl split |
| `shared` | Shared state/services with interface/impl split |
| `ui` | Pure UI components with interface/impl split |
| `utility` | Single-package utility (no interface/impl split) |

## Tools

| Tool | Purpose | Command |
|------|---------|---------|
| `make build` | Build CLI binary | `cd tuist-starter && go build -o ios-app-manager ./cmd/ios-app-manager` |
| `make test` | Run all Go tests | `cd tuist-starter && go test ./...` |
| `make test-update` | Rebuild golden files + run tests | `cd tuist-starter && go test ./internal/testutil -update && go test ./...` |
| `make lint` | Lint Go code | `cd tuist-starter && go vet ./...` |
| `plantuml` | Render dependency diagrams | `plantuml -tpng diagrams/*.puml -o diagrams/` |

## CI/CD (STUB)

CI entry points live in `.github/workflows/`. Currently stubs, not wired to a real project.

| Workflow | What it runs |
|----------|-------------|
| `build.yml` | `make setup` + `make build` |
| `test.yml` | `make setup` + `make test` |
| `lint.yml` | `make lint` |

Adding new CI steps:
- All workflows run on `macos-latest`
- Signing requires `TEAM_ID`, `PROVISIONING_PROFILE_SPECIFIER`, and `PROVISIONING_PROFILE_BASE64` secrets (not yet configured)

## Dependencies

Go: Cobra (CLI framework) + gopkg.in/yaml.v3.

Swift packages managed in generated projects: SwiftIoC, swiftui-relux, swift-relux, swift-httpclient.
