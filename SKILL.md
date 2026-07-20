---
name: ios-app-manager
description: iOS app project management CLI for Tuist-based projects with Relux architecture
triggers:
  - ios project
  - tuist
  - relux module
  - ios-app-manager
globs:
  - ios-app-manager.json
  - "**/Tuist/**"
---

# ios-app-manager

Agent-facing workflow for managing Tuist-based iOS projects that follow Relux module conventions.

## Setup

```bash
# Clone and run setup (builds CLI, installs runtime skill copy, symlinks binary)
git clone git@github.com:relux-works/skill-ios-app-manager.git
cd skill-ios-app-manager
./scripts/setup.sh

# Optional one-time full Xcode selection for iOS simulator/build/profile workflows
./scripts/setup.sh --print-privileged-commands
./scripts/setup.sh --select-xcode
```

Setup does:
- Builds Go CLI binary in `tuist-starter/`
- Copies the skill repo into `~/.agents/skills/ios-app-manager`
- Degitizes the installed copy after sync (`.git`, `.gitignore`, `.gitattributes`, `.gitmodules` are removed)
- Symlinks `~/.claude/skills/ios-app-manager` and `~/.codex/skills/ios-app-manager` to the installed copy in `~/.agents/skills/ios-app-manager`
- Symlinks binary to `~/.local/bin/ios-app-manager`
- `--select-xcode` runs only a hardcoded privileged list: `sudo -v` and `sudo xcode-select -s /Applications/Xcode.app/Contents/Developer`

## Repository structure

```
skill-ios-app-manager/
├── SKILL.md              # This file (skill definition)
├── blueprints/           # Versioned example relux-feature blueprints
├── references/           # CLI, DSL, workflow docs
├── diagrams/             # Architecture diagrams (PlantUML)
├── scripts/setup.sh      # Global skill installer
└── tuist-starter/        # Go CLI source
    ├── cmd/, internal/, pkg/
    ├── go.mod, Makefile
    └── testdata/
```

## Quick start

1. Initialize from config:
   - `ios-app-manager init --config ios-app-manager.json --output .`
2. Generate helper targets:
   - `ios-app-manager generate makefile`
   - `ios-app-manager generate project-config`
3. Run local cycle:
   - `make setup && make build && make test`

Use `--force` when you intentionally want to overwrite scaffold files:
- `ios-app-manager init --config ios-app-manager.json --output . --force`

## Scaffold Change Policy

Follow this policy when changing the scaffold itself or when a project needs scaffold-owned output that the current CLI cannot express.

Do not manually patch files owned by `ios-app-manager` scaffolding or generators (`Project.swift`, `Workspace.swift`, `Package.swift`, `Tuist/ProjectDescriptionHelpers/*`, generated app configuration, module scaffolds) when a required capability is missing from the tool.

If the CLI cannot express the needed setup, document the scaffold gap first: the blocked user intent, the generated files that would otherwise need changes, and the `ios-app-manager` command/generator that should own the behavior. Then decide whether to extend `ios-app-manager`; do not hide the gap with one-off scaffold edits.

Scaffold extensions must follow the plugin architecture. If the behavior belongs to an existing scaffold/setup plugin, extend that plugin. If it is a distinct scaffold concern, add a new pluggable scaffold module/generator and register it through the relevant registry instead of wiring ad hoc logic into unrelated commands.

For broad scaffold domains, keep the top-level plugin as an orchestrator and put concrete concerns into subplugins. Example: `generate app-capabilities` owns the host app capabilities domain, while `app-groups` is a capability subplugin with `init` dependency; future capabilities must be added as separate subplugins rather than collected into one large capability function.

The same plugin/subplugin rule applies to test targets and app extensions:
- `test-targets` is the orchestration plugin; unit-test-target and UI-test-target behavior belongs in separate subplugins with explicit target-name inputs.
- `app-extensions` owns only shared extension infrastructure and registry contracts.
- Every concrete extension type is its own plugin, for example widget-base/static-widget/live-activity/notification-service.
- Do not collect unrelated test target or extension target behavior into one large generator.

Extension targets must keep testable internals out of the `.appex` wrapper. A concrete extension plugin must generate a thin extension target plus a dedicated SwiftPM package named `<ExtensionName>Core`; business logic, payload parsing, timeline/content builders, attachment handling, and other internals go into that Core package. The extension target links the Core product and owns only the platform runtime entrypoint glue. Tests should target the Core package or a generated unit-test target, not the `.appex` target directly.

Scaffold code, templates, docs, and tests must stay organization-agnostic. Use generic examples such as `com.example.*` / `group.com.example.*`; the only non-generic organization reference allowed in this skill is Relux Works ownership/module metadata. Do not add external organization, client, product, Jira, GitLab, or project-specific identifiers to scaffold sources.

Scaffold plugins and subplugins must be idempotent. Re-running the same command against the same config must not duplicate Swift declarations, manifest entries, generated files, dependencies, entitlements, or package settings. When config values change, the plugin should converge generated output to the current config rather than append another variant.

## Command groups

### Project lifecycle

- Init scaffold: `ios-app-manager init --config <path> --output <dir> [--force]`
- Generate Makefile: `ios-app-manager generate makefile`
- Generate SwiftLint config: `ios-app-manager generate swiftlint`
- Sync project manifest config: `ios-app-manager generate project-config`
- Sync generic app runtime config: `ios-app-manager generate application-configuration`
- Sync typed distribution/backend runtime policy: `ios-app-manager generate runtime-profiles`
- Sync host app capabilities from config: `ios-app-manager generate app-capabilities`
- Generate app/extension bundle ids: `ios-app-manager generate bundle-id`
- Generate app/extension versions: `ios-app-manager generate versions`
- Generate app/extension min target: `ios-app-manager generate min-target`
- Generate app/extension signing team id: `ios-app-manager generate team-id`
- Generate host app background modes config: `ios-app-manager generate background-modes-config`
- Generate host app theme/orientation presentation config: `ios-app-manager generate presentation-config`
- Generate host app export compliance config: `ios-app-manager generate export-compliance-config`
- Generate host app privacy usage descriptions config: `ios-app-manager generate privacy-usage-descriptions-config`
- Generate app/extension Swift strictness: `ios-app-manager generate build-flags`
- Generate root/module package strictness: `ios-app-manager generate package-strictness`
- Clean artifacts: `ios-app-manager clean [--deep] [--kill-xcode]`
- Status: `ios-app-manager status`
- Diagram: `ios-app-manager diagram` — generates PlantUML module dependency diagram
- Build profile: `ios-app-manager profile build`
- Rendered layout helper: `ios-app-manager profile layout scaffold`
- Rendered layout XML analysis: `ios-app-manager profile layout analyze --input <xml-or-log>`
- Runtime profile helper: `ios-app-manager profile runtime scaffold`
- Runtime profile log analysis: `ios-app-manager profile runtime analyze --input <log>`
- Runtime error log analysis: `ios-app-manager profile runtime errors [--input <log>]`
- Add test targets: `ios-app-manager test-targets setup --unit-target <UnitTestsName> --ui-target <UITestsName>`

Generate commands are scaffold generator plugins:
- Each `generate <artifact>` entrypoint is a separate scaffold plugin with its own responsibility and dependency contract.
- Use this pattern for scaffold-only sync tasks instead of overloading `init`.
- `generate project-config` is the orchestration entrypoint for project manifest sync and currently runs `generate bundle-id`, `generate versions`, `generate min-target`, `generate team-id`, `generate platform-destinations`, `generate background-modes-config`, `generate presentation-config`, `generate export-compliance-config`, `generate privacy-usage-descriptions-config`, `generate application-configuration`, `generate runtime-profiles`, `generate app-capabilities`, `generate build-flags`, and `generate package-strictness`.
- `generate bundle-id` depends on the `init` scaffold shape and syncs `bundle_id` from `ios-app-manager.json` into the host app `Project.swift` and every `Extensions/*/Project.swift`. Extensions keep their own suffixes, while the containing host bundle root converges to the configured app bundle id.
- `generate versions` depends on the `init` scaffold shape and syncs both `marketing_version` and `project_version` from `ios-app-manager.json` into the host app `Project.swift` and every `Extensions/*/Project.swift`.
- `generate min-target` depends on the same scaffold shape and syncs `min_target` into both `deploymentTargets` and `IPHONEOS_DEPLOYMENT_TARGET` for the host app and extensions.
- `generate team-id` depends on the same scaffold shape and syncs `team_id` into `developmentTeam` constants and `DEVELOPMENT_TEAM` build settings for the host app, app-like targets, test targets, and extensions.
- `generate background-modes-config` depends on the same scaffold shape and syncs host app `background_modes` values into `UIBackgroundModes`. Values are validated against Apple's documented `UIBackgroundModes` strings; common values include `audio` for Audio, AirPlay, and Picture in Picture, `remote-notification` for APNs background notification delivery, and explicit `voip`. Empty or omitted values remove the scaffold-owned key.
- `generate presentation-config` depends on the same scaffold shape and syncs host app presentation keys from `theme` and `orientation` in `ios-app-manager.json`. Supported values: `theme` is `automatic`, `light`, or `dark`; `orientation` is `automatic`, `portrait`, or `landscape`. Automatic values remove the owned Info.plist keys and let iOS use defaults.
- `generate export-compliance-config` depends on the same scaffold shape and syncs host app export compliance from `uses_non_exempt_encryption` in `ios-app-manager.json`. An explicit `false` writes `ITSAppUsesNonExemptEncryption` as `.boolean(false)`; omitting the field removes the owned Info.plist key.
- `generate privacy-usage-descriptions-config` depends on the same scaffold shape and syncs host app `privacy_usage_descriptions` values into Info.plist usage description strings such as `NSBluetoothAlwaysUsageDescription`, `NSBluetoothPeripheralUsageDescription`, `NSCameraUsageDescription`, `NSMicrophoneUsageDescription`, and `NSLocalNetworkUsageDescription`. Empty values remove scaffold-owned keys.
- `generate application-configuration` depends on the same scaffold shape and syncs product-level runtime identity from `ios-app-manager.json` into an `ApplicationConfiguration` Info.plist dictionary for the host app, app-like targets, and extensions. It also generates the `SharedConfig` reader source and app-target `Configuration+ApplicationConfiguration.swift` facade. This dictionary is distinct from target identity keys such as `CFBundleIdentifier`.
- `generate runtime-profiles` owns typed distribution/backend descriptors, fail-closed Firebase public-input validation, explicit public-identity sharing groups, Tuist configurations/schemes, and policy-aware AppConfig integration. Read [`references/runtime-profiles.md`](references/runtime-profiles.md) before adding or changing runtime-profile config.
- `generate app-capabilities` depends on the same scaffold shape and orchestrates host app capability subplugins.
- `app-groups` is an app capability subplugin with `init` dependency; it syncs configured `app_groups` into `Tuist/ProjectDescriptionHelpers/AppCapabilities.swift`, host/test target `Project.swift` Info.plist `AppGroups` dictionaries, generated `SharedConfig`, and `Configuration+AppGroups.swift`. Generated app-group code reads product-level service identity through `Configuration.ApplicationConfiguration`, so prefer `generate project-config` when syncing app groups.
- `generate build-flags` depends on the same scaffold shape and syncs Swift language, strict memory safety, strict concurrency, approachable concurrency, default actor isolation, and upcoming-feature restriction settings from `project_settings.swift` into the host app and extensions.
- `generate package-strictness` syncs the same `project_settings.swift` Swift language/concurrency settings into root `Package.swift` and every module `Packages/*/Package.swift`.
- When `project_settings.swift` is omitted, Swift strictness defaults are derived from `swift_version` and the scaffold's current strict baseline.
- Generated Makefiles use `tuist generate --no-open` by default. To auto-open Xcode explicitly, run `tuist generate --open` yourself or override the generated Makefile call with `make generate TUIST_GENERATE_FLAGS=--open`.
- Generated `make build` and `make test` run `tuist generate` before `xcodebuild` and reuse `make clean-package-artifacts` on exit. The cleanup hook is a no-op by default (`PACKAGE_ARTIFACT_CLEANUP_CMD ?= :`); projects that need checkout-specific cleanup should override that variable in the Makefile custom section. Build uses a generic iOS Simulator destination by default, while test keeps the configured concrete simulator destination unless overridden.
- Projects may configure lifecycle scripts in `ios-app-manager.json` under `scripts.pre_tuist_generate`. Each script node has `path`, `language`, and optional `description`; paths are relative to the project root and cannot escape it. Supported `language` values are `bash`, `swift`, `go`, and `executable`. Generated Makefiles run these scripts after `tuist install` and before `tuist generate`, which is the safe point for patching SwiftPM checkouts before Tuist converts them into Xcode projects.

App Groups capability contract:
- Configure groups in `ios-app-manager.json` with `app_groups`. Configure the generated shared config package name with `shared_config.module_name`; it defaults to `SharedConfig`.
- `app-groups` emits entitlement declarations, root `Package.swift` dependency/product type entries, app/test `.external(name: "<shared_config.module_name>")` dependencies, generated shared package sources, and app/test Info.plist values.
- Info.plist values are grouped under one dictionary key: `"AppGroups": .dictionary([...])`. Do not emit root-level `APP_GROUP_*` keys for current scaffold output.
- The dictionary key for each app group is derived from the identifier, not hardcoded. `group.<bundle_id>` maps to `main`; `group.<bundle_id>.<suffix>` maps from `<suffix>`; other `group.*` identifiers drop only the `group.` prefix.
- Dictionary keys are sanitized to lowerCamelCase Swift identifiers and reused as `AppGroupSlot.dictionaryKey` in the generated shared package. Validation must reject collisions between generated keys.
- Example: with `bundle_id = "com.example.demo.app"`, `group.com.example.demo.app.shared` maps to `AppGroups.shared`, and `group.com.example.demo.app.sso` maps to `AppGroups.sso`.
- App code, tests, and extensions should read app groups through generated APIs such as `DemoAppGroups.read(from:)`; do not hardcode group identifiers or duplicate dictionary-key derivation in handwritten code.
- App-group service identity should come from generated `Configuration.ApplicationConfiguration.current.applicationBundleIdentifier`, not duplicated `bundle_id` constants in app code.

Project config sync workflow:
```bash
# 1. bump project config in ios-app-manager.json
$EDITOR ios-app-manager.json

# 2. restick manifest config into app + extensions + packages
ios-app-manager generate project-config

# 3. regenerate Tuist project artifacts
tuist generate --no-open
```

Swift strictness lives in `ios-app-manager.json` under `project_settings.swift`, for example:
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
        "member_import_visibility": "yes",
        "existential_any": "yes"
      }
    }
  }
}
```

Leaf workflows remain available:
```bash
ios-app-manager generate versions
ios-app-manager generate min-target
ios-app-manager generate team-id
ios-app-manager generate background-modes-config
ios-app-manager generate presentation-config
ios-app-manager generate export-compliance-config
ios-app-manager generate privacy-usage-descriptions-config
ios-app-manager generate application-configuration
ios-app-manager generate runtime-profiles
ios-app-manager generate app-capabilities
ios-app-manager generate build-flags
ios-app-manager generate package-strictness
```

### Profile diagnostics

Use `profile` when an agent needs evidence about build time, runtime slowness, startup latency, runtime failures, or the UI hierarchy that actually rendered on simulator/device.

Decision table:

| Question | Command | Primary output |
| --- | --- | --- |
| Why is the build slow? | `ios-app-manager profile build` | slow commands, target work, critical path estimate |
| What did an existing build log spend time on? | `ios-app-manager profile build --log <xcodebuild.log> --skip-graph` | parsed timing summary without rebuilding |
| Which SwiftUI/function calls are slow or repeated? | `ios-app-manager profile runtime scaffold` then `profile runtime analyze --input <log>` | `IAM_PROFILE` aggregation |
| How long until first meaningful render? | `PerformanceProbe.markAppStart()` + `.firstRenderProfiled("RootView")` then `profile runtime analyze` | app-start-to-first-render duration |
| What runtime errors/faults happened? | `ios-app-manager profile runtime errors [--input <log>]` | grouped errors, crash/exception/hang hints |
| What UI hierarchy actually rendered? | `ios-app-manager profile layout scaffold` then `profile layout analyze --input <xml-or-log>` | rendered accessibility XML tree and layout issues |

Command map:

```bash
# Build timing + target graph critical path
ios-app-manager profile build
ios-app-manager profile build --jobs 8 --configuration Debug
ios-app-manager profile build --log .temp/build-profile/xcodebuild.log --skip-graph

# Runtime view/function/startup instrumentation
ios-app-manager profile runtime scaffold
ios-app-manager profile runtime analyze --input .temp/runtime-profile.log

# Runtime errors/faults/crash hints
ios-app-manager profile runtime errors --input .temp/runtime-errors.log
ios-app-manager profile runtime errors --simulator --device booted --process DemoApp

# Rendered UI hierarchy for agents
ios-app-manager profile layout scaffold
ios-app-manager profile layout analyze --input .temp/layout/feed.xml
ios-app-manager profile layout analyze --input .temp/layout/ui-test.log --format json
```

Feature contracts:

| Feature | Instrumentation | Input | Diagnostics |
| --- | --- | --- | --- |
| Build profiling | none required | `xcodebuild -showBuildTimingSummary`, Tuist `legacyJSON` graph | top commands, target work, ideal parallelism ceiling, critical path estimate |
| Runtime profiling | generated `PerformanceProbe.swift` in app target | app logs containing `IAM_PROFILE` JSON lines | counts, total/average/max duration, main-thread slow calls, repeated-call warnings |
| Startup timing | `PerformanceProbe.markAppStart()` and `.firstRenderProfiled(...)` | same `IAM_PROFILE` log | app-start-to-first-render duration |
| Runtime errors | optional `PerformanceProbe.error(...)`; unified logs also supported | `IAM_ERROR`, unified-log NDJSON/plain text | grouped signatures and hints for crash, exception, hang, thread checker, SwiftUI background publish |
| Rendered layout XML | generated `LayoutHierarchyProbe.swift` in UI test target | XCTest XML, Appium/WDA page source, or `IAM_LAYOUT_XML_*` log block | agent-readable tree, duplicate identities, missing interactive identity, tiny tap targets, offscreen frames |

Runtime instrumentation usage:

```swift
@main
struct DemoApp: App {
    init() {
        PerformanceProbe.markAppStart()
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .profiled("RootView")
                .firstRenderProfiled("RootView")
        }
    }
}

let items = PerformanceProbe.measure("Feed.filter") {
    allItems.filter(\.isVisible)
}
```

Rendered layout usage:

```swift
final class FeedUITests: XCTestCase {
    @MainActor
    func testFeedLayoutDump() {
        let app = XCUIApplication()
        app.launch()
        attachLayoutHierarchyXML(app, name: "feed", screenName: "Feed")
    }
}
```

Artifacts and references:

| Artifact | Location |
| --- | --- |
| Build profiling research | `references/build-profiling-research.md` |
| Runtime profiling research | `references/runtime-profiling-research.md` |
| Runtime error diagnostics research | `references/runtime-error-diagnostics-research.md` |
| Rendered layout hierarchy research | `references/layout-hierarchy-diagnostics-research.md` |
| Architecture overview | `references/profile-diagnostics-architecture.md` |
| Architecture diagram | `diagrams/profile-diagnostics-architecture.puml` |

Implementation notes:

- `scripts/setup.sh` verifies Go and Tuist, then reports Xcode/profile-tool readiness for `xcodebuild`, `xcrun/simctl`, macOS `log`, and optional `xctrace`; missing Xcode tools are warnings so the CLI can still be installed.
- `./scripts/setup.sh --select-xcode` is the only setup path that uses sudo; it prints the exact hardcoded command list first and does not evaluate user-provided shell strings.
- Build profiling is local-first and works without Tuist Cloud.
- Runtime helpers are debug-only and use public APIs/signposts plus structured log lines.
- Rendered layout diagnostics use XCTest/accessibility hierarchy, not private UIKit/SwiftUI view graph scraping.
- Prefer JSON output when another agent or CI step will consume a report.

### Physical iOS launch with logs

When a user asks to start the app on all connected physical iPhones/iPads, with or without rebuild, and keep runtime logs attached, use the workflow in [`references/physical-ios-launch-log-workflow.md`](references/physical-ios-launch-log-workflow.md).

Key rules:
- Keep the workflow generic: discover USB devices or accept explicit UDIDs; never bake local device names or identifiers into reusable scripts.
- Put project-specific app defaults, bundle id, environment, and launch flags in the generated Makefile's preserved custom section.
- Start log capture before launching the app so startup, Bluetooth, and permission/runtime events are not missed.
- Keep the capture command in the foreground until the user says to stop, then terminate child log streams and leave raw plus filtered artifacts under `.temp/`.
- Use `devicectl device process launch` for CoreDevice-compatible devices and `ios-deploy --bundle <app> --noinstall --justlaunch` as a legacy fallback when CoreDevice cannot launch a connected device.

### Infrastructure setup (run in order)

- IoC container: `ios-app-manager ioc setup`
- Relux state management: `ios-app-manager relux setup`
- Keychain wrapper: `ios-app-manager secure-store setup --access-group <group>`
- Token provider: `ios-app-manager token-provider setup`
- Utilities: `ios-app-manager utilities setup`
- FoundationPlus (re-export + helpers): `ios-app-manager foundation-plus setup`
- SwiftUIPlus (re-export + components): `ios-app-manager swiftui-plus setup`
- Test targets: `ios-app-manager test-targets setup --unit-target <UnitTestsName> --ui-target <UITestsName>`
- App extensions base: `ios-app-manager app-extensions setup`
- Notification Service Extension: `ios-app-manager notification-service setup [--extension-target <Name>] [--bundle-id-suffix <suffix>]`
- Widget base (WidgetBundle + WidgetKit): `ios-app-manager widget-base setup`
- App Intents (interactive widgets): `ios-app-manager app-intents setup`
- Static widget (timeline widget): `ios-app-manager static-widget setup`
- Live Activity (ActivityKit + Dynamic Island): `ios-app-manager live-activity setup`
- HTTP client: `ios-app-manager http-client setup`
- AppConfig (env switching + ApiConfigurator): `ios-app-manager app-config setup`

Pipeline order matters — each command has prerequisites:
```
init → ioc → relux → secure-store → token-provider → utilities
     → foundation-plus → swiftui-plus → test-targets
     → app-extensions → notification-service
                     → widget-base → app-intents → static-widget
                                  → live-activity
     → module create (blueprints)
     → app-config → http-client
```

### Module management

- Create module: `ios-app-manager module create <name> --type <feature|kit|shared|ui|utility>`
- Create relux module from blueprint: `ios-app-manager module create --from <blueprint.json>`
- Generate blueprint template: `ios-app-manager module blueprint <Name>`
- List modules: `ios-app-manager module list`
- Delete module: `ios-app-manager module delete <name> [--force]`

Versioned blueprint examples:
- [`blueprints/xflow-ios/Auth.blueprint.json`](blueprints/xflow-ios/Auth.blueprint.json)
- [`blueprints/xflow-ios/CRM.blueprint.json`](blueprints/xflow-ios/CRM.blueprint.json)
- [`blueprints/xflow-ios/Onboarding.blueprint.json`](blueprints/xflow-ios/Onboarding.blueprint.json)
- [`blueprints/xflow-ios/Organizations.blueprint.json`](blueprints/xflow-ios/Organizations.blueprint.json)
- [`blueprints/xflow-ios/Profile.blueprint.json`](blueprints/xflow-ios/Profile.blueprint.json)

Module type guidance:
- `feature`: UI module, interface/implementation split (no Relux business logic)
- `relux-feature`: Blueprint-only. Use `module blueprint <Name>` to generate template, then `module create --from <file>.blueprint.json`
- `kit`: Business logic library with interface/implementation split
- `shared`: Shared state/services with interface/implementation split
- `ui`: Pure UI components with interface/implementation split
- `utility`: Single-package utility module (no interface/impl split)

### Dependency management

- Internal add: `ios-app-manager dep add <module> --depends-on <other>`
- Internal remove: `ios-app-manager dep remove <module> --depends-on <other>`
- External add: `ios-app-manager dep add-external --url <git_url> --version <ver> [--module <target>] [--product <product>] [--target-setting KEY=VALUE] [--app-target]`
- External remove: `ios-app-manager dep remove-external --package <name>`
- List dependencies: `ios-app-manager dep list [<module>]`

Framework policy:
- Scaffold-generated local package products are emitted as dynamic libraries by default.
- External Swift package products added through `setup` plugins, `module create --from <blueprint>`, or `dep add-external` are automatically forced to `.framework` in root `Package.swift` under Tuist `PackageSettings.productTypes`. Use `--product` when the Swift product name differs from the package name. Use `--target-setting KEY=VALUE` for product-level Tuist `PackageSettings.targetSettings` overrides such as package min-target fixes.
- `dep remove-external` also removes the matching framework override from root `Package.swift`.
- If someone adds a remote package by hand outside `ios-app-manager`, this policy is not applied until the manifest is brought back through the tool flow or patched manually.

### Entitlements

- List keys: `ios-app-manager entitlements list`
- Optional explicit plist path: `--path <entitlements_file>`

### Push tooling

- Get token: `ios-app-manager push token`
- Send push: `ios-app-manager push send --token <token> [--env dev|prod] [--payload <file>]`

### DSL entrypoints

- Query: `ios-app-manager q '<query>'`
- Mutation: `ios-app-manager m '<mutation>'`

### Diagnostics

- Diagram: `ios-app-manager diagram` — generates PlantUML dependency graph of all modules in the project

## Scaffolding pipeline — dependency graph

See [`diagrams/scaffolding-pipeline.puml`](diagrams/scaffolding-pipeline.puml) for the visual dependency graph. Always consult this diagram to understand what depends on what before running setup commands.

### Pipeline elements

| Element | CLI command | What it creates | Prerequisites |
|---------|-------------|-----------------|---------------|
| **init** | `init` | Project scaffold: Tuist manifests, App.swift, Info.plist, entitlements, Configuration, Assets | Config file |
| **ioc** | `ioc setup` | SwiftIoC integration: Registry.swift, App.swift init injection, SwiftIoC dependency | init |
| **relux** | `relux setup` | Relux state management: ReluxLogger, Registry infra, swift-relux + swiftui-relux deps | ioc |
| **secure-store** | `secure-store setup --access-group <group>` | SecureStore + SecureStoreImpl plus a focused, non-destructive Registry patch | ioc |
| **token-provider** | `token-provider setup` | TokenProvider + TokenProviderImpl: token storage/refresh | ioc |
| **utilities** | `utilities setup` | Utilities single-package: HttpClientUtils helpers | ioc |
| **foundation-plus** | `foundation-plus setup` | FoundationPlus single-package: `@_exported import Foundation`, MaybeData, CompletionStatus | ioc |
| **swiftui-plus** | `swiftui-plus setup` | SwiftUIPlus single-package: `@_exported import SwiftUI`, AsyncButton | ioc |
| **test-targets** | `test-targets setup --unit-target <name> --ui-target <name>` | Unit/UI test target scaffold via unit-test-target and UI-test-target subplugins | init |
| **app-extensions** | `app-extensions setup` | SharedKit package + Extensions/ directory for extension targets; owns shared contracts, not concrete extension internals | init |
| **notification-service** | `notification-service setup [--extension-target <name>] [--bundle-id-suffix <suffix>]` | Notification Service Extension wrapper plus `<ExtensionName>Core` handler package | app-extensions |
| **widget-base** | `widget-base setup` | Widget extension target, thin WidgetBundle wrapper, WidgetKit SDK, App Groups, `<WidgetName>Core` package | app-extensions |
| **app-intents** | `app-intents setup` | AppIntent scaffold in WidgetCore, AppIntents SDK on extension | widget-base |
| **static-widget** | `static-widget setup` | StaticConfiguration widget internals in WidgetCore: TimelineProvider, entry, view with interactive Button(intent:) | widget-base, app-intents |
| **live-activity** | `live-activity setup` | ActivityAttributes in SharedKit, ActivityConfiguration + Dynamic Island in WidgetCore, LiveActivityManager in app target | widget-base |
| **module create** | `module create <Name> --type <type>` | Feature/kit/shared/ui/utility module with file layout, Registry re-generation | ioc |
| **http-client** | `http-client setup` | HttpClient IoC registration, swift-httpclient dep, Configuration constants | ioc |
| **app-config** | `app-config setup` | 8 AppConfig files: Env, Configuration, Manager, ApiConfigurator. Registry IoC patch | ioc, secure-store |
| **diagram** | `diagram` | PlantUML dependency graph of all project modules | init |

### Important: ordering constraints

Commands that directly patch Registry.swift (`http-client setup`, `app-config setup`) must run **after** all `module create` calls, because `module create` regenerates Registry from template and wipes direct patches.

`secure-store setup` patches only its imports, registration, and builder when Registry.swift already exists. Do not rerun `ioc setup` merely to adopt SecureStore in a mature custom composition root.

Full recommended pipeline:
```bash
ios-app-manager init
ios-app-manager ioc setup
ios-app-manager relux setup
ios-app-manager secure-store setup --access-group <group>
ios-app-manager token-provider setup
ios-app-manager utilities setup
ios-app-manager foundation-plus setup
ios-app-manager swiftui-plus setup
ios-app-manager test-targets setup --unit-target <UnitTestsName> --ui-target <UITestsName>
ios-app-manager app-extensions setup
ios-app-manager notification-service setup
ios-app-manager widget-base setup
ios-app-manager app-intents setup
ios-app-manager static-widget setup
ios-app-manager live-activity setup
ios-app-manager module create --from <name>.blueprint.json
ios-app-manager app-config setup
ios-app-manager http-client setup
ios-app-manager diagram
```

## Workflow references

- Command/flag reference: [`references/cli-reference.md`](references/cli-reference.md)
- DSL syntax and operations: [`references/dsl-reference.md`](references/dsl-reference.md)
- End-to-end examples: [`references/workflows.md`](references/workflows.md)
- Dependency diagram: [`diagrams/scaffolding-pipeline.puml`](diagrams/scaffolding-pipeline.puml)
- Blueprint examples: [`blueprints/README.md`](blueprints/README.md)
