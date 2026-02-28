# Connect-iOS Commit Audit — Agent Instructions

## Your Job

Analyze a batch of git commits from `.temp/connect-ios/` repo against ios-app-manager scaffolding capabilities. For each commit, produce a report.

## How to Analyze a Commit

```bash
cd .temp/connect-ios
git show <sha> --stat     # see what files changed
git show <sha>            # see the full diff
```

For merge commits — skip them, just note "Merge commit, see feature commits above."

For "no message" commits — still analyze the diff.

## Report Format (per commit)

```markdown
### <sha_short> <commit message>

**What**: <1-2 sentence description of what this commit does>

**Scaffolding assessment**:
- [ ] INSTRUCTION | ALARM

**INSTRUCTION** (if scaffoldable):
- `ios-app-manager <command>` — what CLI command handles this
- Manual files: list of files that need to be created manually and WHERE they go in Packages/<Name>/ structure
- Notes: any special wiring or configuration

**ALARM** (if NOT scaffoldable):
- What's missing: <specific capability that ios-app-manager lacks>
- Severity: HIGH (blocks feature) | MEDIUM (workaround exists) | LOW (nice to have)
- Suggested solution: <brief idea of what to build>
```

## ios-app-manager Capabilities Reference

Current CLI commands:
- `init` — project scaffold (Tuist manifests, App.swift, Info.plist, entitlements, Configuration)
- `ioc setup` — SwiftIoC integration (Registry.swift)
- `relux setup` — Relux state management infrastructure
- `secure-store setup --access-group <group>` — Keychain wrapper module
- `token-provider setup` — Token storage/refresh module
- `utilities setup` — Shared helpers module
- `module create <Name> --type <type>` — Create module (feature, relux-feature, kit, shared, ui, utility)
- `dep add <module> --depends-on <other>` — Internal dependency
- `dep add-external --url <url> --version <ver> --module <target>` — External package
- `http-client setup` — HttpClient IoC registration
- `app-config setup` — AppConfig manager + ApiConfigurator
- `entitlements add <key>` — Entitlements management
- `push send/token` — APNs tooling
- `generate makefile` — Build automation

Module types:
- `feature` — UI module, interface/impl split (no Relux)
- `relux-feature` — Feature with Relux business logic (actions, effects, state, flow)
- `kit` — Business logic library
- `shared` — Shared state/services
- `ui` — Pure UI components
- `utility` — Single-package utility

Generated structure for relux-feature:
```
Packages/<Name>/Sources/<Name>/
    <Name>.swift                    ← namespace (Data/Api/DTO, Business/Model, UI/Model)
    Module/<Name>.Module.swift
    Module/<Name>.Module+Interface.swift
    Business/<Name>.Business+Action.swift
    Business/<Name>.Business+Effect.swift
Packages/<Name>Impl/Sources/<Name>Impl/
    Module/<Name>.Module+Impl.swift
    Business/<Name>.Business+State.swift
    Business/<Name>.Business+Flow.swift
```

## Important

- Read previous batch reports before starting yours (for context continuity)
- Be specific about file paths and CLI commands
- ALARM entries are the most valuable output — be detailed about what's missing
- Don't over-explain obvious things
- Merge commits = skip with a note
