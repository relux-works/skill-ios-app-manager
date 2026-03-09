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
# Clone and run setup (builds CLI, registers skill globally, symlinks binary)
git clone git@github.com:relux-works/skill-ios-app-manager.git
cd skill-ios-app-manager
./scripts/setup.sh
```

Setup does:
- Builds Go CLI binary in `tuist-starter/`
- Symlinks skill to `~/.agents/skills/ios-app-manager` -> `~/.claude/skills/` + `~/.codex/skills/`
- Symlinks binary to `~/.local/bin/ios-app-manager`

## Repository structure

```
skill-ios-app-manager/
├── SKILL.md              # This file (skill definition)
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
3. Run local cycle:
   - `make setup && make build && make test`

Use `--force` when you intentionally want to overwrite scaffold files:
- `ios-app-manager init --config ios-app-manager.json --output . --force`

## Command groups

### Project lifecycle

- Init scaffold: `ios-app-manager init --config <path> --output <dir> [--force]`
- Generate Makefile: `ios-app-manager generate makefile`
- Generate SwiftLint config: `ios-app-manager generate swiftlint`
- Clean artifacts: `ios-app-manager clean [--deep] [--kill-xcode]`
- Status: `ios-app-manager status`
- Diagram: `ios-app-manager diagram` — generates PlantUML module dependency diagram

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
