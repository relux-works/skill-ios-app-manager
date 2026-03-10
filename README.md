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

## Build & Run

```bash
cd tuist-starter
make setup    # create local Go cache dirs
make build    # build binary -> ./ios-app-manager
make test     # run all tests
make lint     # go vet
```

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
