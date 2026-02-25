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

### Module management

- Create module: `ios-app-manager module create <name> --type <feature|kit|shared|ui|utility>`
- List modules: `ios-app-manager module list`
- Delete module: `ios-app-manager module delete <name> [--force]`

Module type guidance:
- `feature`: Relux + UI, interface/implementation split
- `kit`: Logic module with interface/implementation split
- `shared`: Shared state/services with interface/implementation split
- `ui`: UI-focused module with interface/implementation split
- `utility`: Single-package utility module

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

## Workflow references

- Command/flag reference: [`references/cli-reference.md`](references/cli-reference.md)
- DSL syntax and operations: [`references/dsl-reference.md`](references/dsl-reference.md)
- End-to-end examples: [`references/workflows.md`](references/workflows.md)
