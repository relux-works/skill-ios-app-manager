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
- Clean artifacts: `ios-app-manager clean [--deep] [--kill-xcode]`
- Status placeholder: `ios-app-manager status`

### Infrastructure setup (run in order)

- IoC container: `ios-app-manager ioc setup`
- Relux state management: `ios-app-manager relux setup`
- Keychain wrapper: `ios-app-manager secure-store setup --access-group <group>`
- Token provider: `ios-app-manager token-provider setup`
- Utilities: `ios-app-manager utilities setup`
- HTTP client: `ios-app-manager http-client setup`
- AppConfig (env switching + ApiConfigurator): `ios-app-manager app-config setup`

Pipeline order matters — each command has prerequisites:
```
init → ioc setup → relux setup → secure-store setup → token-provider setup
     → utilities setup → module create → http-client setup → app-config setup
```

### Module management

- Create module: `ios-app-manager module create <name> --type <feature|kit|shared|ui|utility|relux-feature>`
- List modules: `ios-app-manager module list`
- Delete module: `ios-app-manager module delete <name> [--force]`

Module type guidance:
- `feature`: UI module, interface/implementation split (no Relux business logic)
- `relux-feature`: Feature with full Relux business logic (actions, effects, state, flow)
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

- Add key: `ios-app-manager entitlements add <key> [--value <val>]`
- Remove key: `ios-app-manager entitlements remove <key>`
- List keys: `ios-app-manager entitlements list`
- Optional explicit plist path: `--path <entitlements_file>`

### Push tooling

- Get token: `ios-app-manager push token`
- Send push: `ios-app-manager push send --token <token> [--env dev|prod] [--payload <file>]`

### DSL entrypoints

- Query: `ios-app-manager q '<query>'`
- Mutation: `ios-app-manager m '<mutation>'`

## Scaffolding pipeline — dependency graph

See [`diagrams/scaffolding-pipeline.puml`](../../../diagrams/scaffolding-pipeline.puml) for the visual dependency graph. Always consult this diagram to understand what depends on what before running setup commands.

### Pipeline elements

| Element | CLI command | What it creates | Prerequisites |
|---------|-------------|-----------------|---------------|
| **init** | `ios-app-manager init` | Project scaffold: Tuist manifests (Project.swift, Package.swift, Workspace.swift, Tuist.swift), App.swift, Info.plist, entitlements, Configuration namespace, Assets | Config file |
| **ioc setup** | `ios-app-manager ioc setup` | SwiftIoC integration: Registry.swift with IoC container, App.swift init injection, SwiftIoC package dependency | init |
| **relux setup** | `ios-app-manager relux setup` | Relux state management: ReluxLogger, Relux infrastructure registrations in Registry, swift-relux + swiftui-relux dependencies | ioc setup |
| **secure-store setup** | `ios-app-manager secure-store setup --access-group <group>` | SecureStore module (Packages/SecureStore + SecureStoreImpl): Keychain wrapper with interface/impl split, `SecureStoring` protocol | ioc setup |
| **token-provider setup** | `ios-app-manager token-provider setup` | TokenProvider module (Packages/TokenProvider + TokenProviderImpl): token storage/refresh with interface/impl split | ioc setup |
| **utilities setup** | `ios-app-manager utilities setup` | Utilities module (Packages/Utilities): shared helpers (HttpClientUtils, etc.) | ioc setup |
| **module create** | `ios-app-manager module create <Name> --type <type>` | Feature/kit/shared/ui/utility module with appropriate file layout, Registry re-generation | ioc setup |
| **http-client setup** | `ios-app-manager http-client setup` | HttpClient IoC registration, swift-httpclient dependency, Configuration+HttpClient constants | ioc setup, swift-httpclient |
| **app-config setup** | `ios-app-manager app-config setup` | 8 AppConfig Swift files in app target: namespace, Env, Configuration, Presets, Protocols (IApiConfigManager), Manager (NSLock + SecureStore keychain persistence), ApiConfigurator struct, UrlComponents. Patches Registry with IoC registration | ioc setup, secure-store setup |

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
ios-app-manager module create <Name> --type <type>   # all modules first
ios-app-manager http-client setup                     # patches Registry
ios-app-manager app-config setup                      # patches Registry
```

## Workflow references

- Command/flag reference: [`references/cli-reference.md`](references/cli-reference.md)
- DSL syntax and operations: [`references/dsl-reference.md`](references/dsl-reference.md)
- End-to-end examples: [`references/workflows.md`](references/workflows.md)
- Dependency diagram: [`diagrams/scaffolding-pipeline.puml`](../../../diagrams/scaffolding-pipeline.puml)
