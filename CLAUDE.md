# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go CLI tool (`ios-app-manager`) that scaffolds and manages Tuist-based iOS projects with Relux state management architecture. It is **not** an iOS project itself — it generates and manipulates iOS project files.

## Build & Test Commands

```bash
make setup          # Create local Go cache dirs (.cache/)
make build          # Build binary → ./ios-app-manager
make test           # Run all tests: go test ./...
make test-update    # Rebuild golden files, then run all tests
make lint           # go vet ./...
```

Run a single package's tests:
```bash
go test ./internal/config/...
go test ./internal/tuistproj/... -run TestSpecificFunc
```

Update golden files for snapshot tests:
```bash
go test ./internal/testutil -update
```

Go caches are kept in `.cache/` (gitignored) via `GOCACHE`, `GOMODCACHE`, `GOPATH` env vars set in the Makefile.

## Architecture

**Entry point**: `cmd/ios-app-manager/main.go` → `internal/cli.Execute()`

**Layered design**:
```
CLI (cobra commands)  →  AppManager (orchestrator)  →  Domain Managers  →  File I/O / Templates
     internal/cli/       internal/components/           internal/*/
```

**CLI commands**: `init`, `status`, `module`, `dep`, `entitlements`, `push`, `generate`, `clean`, `query` (`q`), `mutate` (`m`), `ioc`, `relux`

### Key packages (`internal/`)

| Package | Role |
|---------|------|
| `cli` | Cobra command tree (root, subcommands, flags) |
| `components` | `AppManager` interface — orchestrates tuist + relux managers |
| `config` | `ProjectConfig` schema, JSON loading, validation. Default config: `ios-app-manager.json` |
| `tuistproj` | Tuist manifest generation & editing (Project.swift, Package.swift) |
| `relux` | Relux state management scaffolding (actions, effects, state, flow) |
| `modules` | Module creation by type (feature, kit, shared, ui, utility, relux-feature) |
| `scaffold` | Full project scaffolding from config → template rendering → file writes |
| `template` | Tuist manifest templates (embedded via `embed.FS`) |
| `dsl` | Custom query/mutation DSL parser and executor (**operations are stubs**) |
| `deps` | Dependency graph management (internal module deps + external Swift packages) |
| `entitlements` | iOS entitlements plist manipulation |
| `ioc` | SwiftIoC container integration setup (`ioc setup` command) |
| `push` | APNs push token + notification sending |
| `clean` | Build artifact cleanup |
| `testutil` | Golden file testing + stdout capture helpers |
| `e2e` | End-to-end pipeline tests (full CLI workflow against temp dirs) |

### Template system

Templates are Go `text/template` files embedded via `embed.FS`:
- `internal/template/tuist/*.tmpl` — Tuist manifests (Project.swift, Workspace.swift, Package.swift, Tuist.swift)
- `internal/tuistproj/templates/*.tmpl` — Package.swift generation for modules
- `internal/relux/templates/*.tmpl` — Relux scaffolding (namespace, module, interface, impl, actions, effects, state, flow)
- `internal/relux/setup_templates/*.tmpl` — Initial Relux app setup files (composition root, IoC registration/resolver)

### Module types

When creating modules (`module create <name> --type <type>`):
- **feature**: UI module with interface/implementation split. Generates namespace + module + interface + impl files. Does NOT scaffold Relux business logic.
- **kit**: Business logic library, interface/impl split (same template set as feature, no UI flag)
- **shared**: Shared state/services, interface/impl split
- **ui**: Pure UI components, interface/impl split
- **utility**: Single-package utility (no interface/impl split)
- **relux-feature**: Feature with full Relux business logic: actions, effects, state, flow. Auto-adds `swift-relux` dependency.

### Generated file layout

Modules generate Swift files into `Module/` and `Business/` subdirectories:

```
Packages/<Name>/Sources/<Name>/
    <Name>.swift                              ← namespace enum
    Module/
        <Name>.Module.swift                   ← module definition
        <Name>.Module+Interface.swift         ← public interface
Packages/<Name>Impl/Sources/<Name>Impl/
    Module/
        <Name>.Module+Impl.swift              ← implementation
```

For `relux-feature` type, additional `Business/` subdirectory:
```
Packages/<Name>/Sources/<Name>/
    Business/
        <Name>.Business+Action.swift          ← actions
        <Name>.Business+Effect.swift          ← effects
Packages/<Name>Impl/Sources/<Name>Impl/
    Business/
        <Name>.Business+State.swift           ← state
        <Name>.Business+Flow.swift            ← flow (reducer)
```

### Dependency injection

`AppManager` aggregates `TuistProjectManager` and `ReluxManager` interfaces. File I/O functions (`loadConfig`, `readDir`, `removeAll`) are injected as function fields for testability.

## Testing patterns

- Standard Go `testing` package, no external test framework
- Golden file snapshots via `internal/testutil/golden.go` — use `-update` flag to regenerate
- `testutil.CaptureOutput()` for stdout assertions
- Tests are colocated with source (`*_test.go` in same package)
- Golden/test data lives in `testdata/` directories (both at repo root and within packages) — this is a standard Go convention, `go build` ignores `testdata/`
- `internal/e2e/` runs full pipeline tests: init → module create → validation against a temp directory

## Skill documentation

Detailed CLI reference, DSL syntax, and workflow examples are in `.claude/skills/ios-app-manager/`:
- `SKILL.md` — quick start and command overview
- `references/cli-reference.md` — full flag/command reference
- `references/dsl-reference.md` — DSL query/mutation syntax
- `references/workflows.md` — end-to-end usage examples

## Demo App Workflow

The demo/test app lives in `.temp/demo-project/` and is **generated output** of the CLI tool. **Never edit the demo app files directly.** When the demo app has bugs or missing files:

1. Fix the **source of truth**: Go CLI code, templates (`*.tmpl`), scaffolding logic, or snapshot golden files
2. Rebuild the CLI: `make build`
3. Delete the demo app: `rm -rf .temp/demo-project/`
4. Regenerate from scratch using the full pipeline:
   ```bash
   cd .temp/demo-project
   ../../ios-app-manager init
   ../../ios-app-manager ioc setup
   ../../ios-app-manager relux setup
   ../../ios-app-manager secure-store setup
   ../../ios-app-manager token-provider setup
   ../../ios-app-manager utilities setup
   ../../ios-app-manager module create <Name> --type relux-feature
   ```
5. Verify the regenerated app is correct (build with tuist/xcodebuild if needed)

The config file is saved separately at `.temp/demo-config.json` — copy it into the demo project directory before `init`.

## Dependencies

Minimal: Cobra (CLI framework) + gopkg.in/yaml.v3. No other external Go deps.

The CLI manages these **Swift** packages in generated projects:
- **SwiftIoC** (from: 1.0.1) — dependency injection, added by `ioc setup`
- **swiftui-relux** (from: 8.0.1) — SwiftUI Relux bindings, added by `relux setup`
- **swift-relux** (from: 9.0.1) — core Relux framework, added automatically when creating `relux-feature` modules
