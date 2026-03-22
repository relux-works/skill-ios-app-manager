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
```

Setup does:
- Builds Go CLI binary in `tuist-starter/`
- Copies the skill repo into `~/.agents/skills/ios-app-manager`
- Degitizes the installed copy after sync (`.git`, `.gitignore`, `.gitattributes`, `.gitmodules` are removed)
- Symlinks `~/.claude/skills/ios-app-manager` and `~/.codex/skills/ios-app-manager` to the installed copy in `~/.agents/skills/ios-app-manager`
- Symlinks binary to `~/.local/bin/ios-app-manager`

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

## Command groups

### Project lifecycle

- Init scaffold: `ios-app-manager init --config <path> --output <dir> [--force]`
- Generate Makefile: `ios-app-manager generate makefile`
- Generate SwiftLint config: `ios-app-manager generate swiftlint`
- Sync project manifest config: `ios-app-manager generate project-config`
- Generate app/extension versions: `ios-app-manager generate versions`
- Generate app/extension min target: `ios-app-manager generate min-target`
- Generate app/extension Swift strictness: `ios-app-manager generate build-flags`
- Generate root/module package strictness: `ios-app-manager generate package-strictness`
- Clean artifacts: `ios-app-manager clean [--deep] [--kill-xcode]`
- Status: `ios-app-manager status`
- Diagram: `ios-app-manager diagram` — generates PlantUML module dependency diagram

Generate commands are scaffold generator plugins:
- Each `generate <artifact>` entrypoint is a separate scaffold plugin with its own responsibility and dependency contract.
- Use this pattern for scaffold-only sync tasks instead of overloading `init`.
- `generate project-config` is the orchestration entrypoint for project manifest sync and currently runs `generate versions`, `generate min-target`, `generate build-flags`, and `generate package-strictness`.
- `generate versions` depends on the `init` scaffold shape and syncs both `marketing_version` and `project_version` from `ios-app-manager.json` into the host app `Project.swift` and every `Extensions/*/Project.swift`.
- `generate min-target` depends on the same scaffold shape and syncs `min_target` into both `deploymentTargets` and `IPHONEOS_DEPLOYMENT_TARGET` for the host app and extensions.
- `generate build-flags` depends on the same scaffold shape and syncs Swift language/concurrency build settings from `project_settings.swift` into the host app and extensions.
- `generate package-strictness` syncs the same `project_settings.swift` Swift language/concurrency settings into root `Package.swift` and every module `Packages/*/Package.swift`.
- When `project_settings.swift` is omitted, Swift strictness defaults are derived from `swift_version` and the scaffold's current strict baseline.
- Generated Makefiles use `tuist generate --no-open` by default. To auto-open Xcode explicitly, run `tuist generate --open` yourself or override the generated Makefile call with `make generate TUIST_GENERATE_FLAGS=--open`.

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
ios-app-manager generate build-flags
ios-app-manager generate package-strictness
```

### Infrastructure setup (run in order)

- IoC container: `ios-app-manager ioc setup`
- Relux state management: `ios-app-manager relux setup`
- Keychain wrapper: `ios-app-manager secure-store setup --access-group <group>`
- Token provider: `ios-app-manager token-provider setup`
- Utilities: `ios-app-manager utilities setup`
- FoundationPlus (re-export + helpers): `ios-app-manager foundation-plus setup`
- SwiftUIPlus (re-export + components): `ios-app-manager swiftui-plus setup`
- App extensions base: `ios-app-manager app-extensions setup`
- Widget base (WidgetBundle + WidgetKit): `ios-app-manager widget-base setup`
- App Intents (interactive widgets): `ios-app-manager app-intents setup`
- Static widget (timeline widget): `ios-app-manager static-widget setup`
- Live Activity (ActivityKit + Dynamic Island): `ios-app-manager live-activity setup`
- HTTP client: `ios-app-manager http-client setup`
- AppConfig (env switching + ApiConfigurator): `ios-app-manager app-config setup`

Pipeline order matters — each command has prerequisites:
```
init → ioc → relux → secure-store → token-provider → utilities
     → foundation-plus → swiftui-plus
     → app-extensions → widget-base → app-intents → static-widget
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
- External add: `ios-app-manager dep add-external --url <git_url> --version <ver> [--module <target>]`
- External remove: `ios-app-manager dep remove-external --package <name>`
- List dependencies: `ios-app-manager dep list [<module>]`

Framework policy:
- Scaffold-generated local package products are emitted as dynamic libraries by default.
- External Swift package products added through `setup` plugins, `module create --from <blueprint>`, or `dep add-external` are automatically forced to `.framework` in root `Package.swift` under Tuist `PackageSettings.productTypes`.
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
| **secure-store** | `secure-store setup --access-group <group>` | SecureStore + SecureStoreImpl: Keychain wrapper with interface/impl split | ioc |
| **token-provider** | `token-provider setup` | TokenProvider + TokenProviderImpl: token storage/refresh | ioc |
| **utilities** | `utilities setup` | Utilities single-package: HttpClientUtils helpers | ioc |
| **foundation-plus** | `foundation-plus setup` | FoundationPlus single-package: `@_exported import Foundation`, MaybeData, CompletionStatus | ioc |
| **swiftui-plus** | `swiftui-plus setup` | SwiftUIPlus single-package: `@_exported import SwiftUI`, AsyncButton | ioc |
| **app-extensions** | `app-extensions setup` | SharedKit package + Extensions/ directory for extension targets | init |
| **widget-base** | `widget-base setup` | Widget extension target, WidgetBundle, WidgetKit SDK, App Groups | app-extensions |
| **app-intents** | `app-intents setup` | AppIntent scaffold (WidgetToggleIntent), AppIntents SDK | widget-base |
| **static-widget** | `static-widget setup` | StaticConfiguration widget: TimelineProvider, entry, view with interactive Button(intent:) | widget-base, app-intents |
| **live-activity** | `live-activity setup` | ActivityAttributes in SharedKit, ActivityConfiguration + Dynamic Island, LiveActivityManager | widget-base |
| **module create** | `module create <Name> --type <type>` | Feature/kit/shared/ui/utility module with file layout, Registry re-generation | ioc |
| **http-client** | `http-client setup` | HttpClient IoC registration, swift-httpclient dep, Configuration constants | ioc |
| **app-config** | `app-config setup` | 8 AppConfig files: Env, Configuration, Manager, ApiConfigurator. Registry IoC patch | ioc, secure-store |
| **diagram** | `diagram` | PlantUML dependency graph of all project modules | init |

### Important: ordering constraints

Commands that directly patch Registry.swift (`http-client setup`, `app-config setup`) must run **after** all `module create` calls, because `module create` regenerates Registry from template and wipes direct patches.

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
ios-app-manager app-extensions setup
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
