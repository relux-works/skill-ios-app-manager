# ios-app-manager workflows

This guide contains practical agent workflows using the current CLI.

## 1) Create a new project from scratch

```bash
# 1) Initialize scaffold from config
ios-app-manager init --config ios-app-manager.json --output .

# 2) Generate helper targets
ios-app-manager generate makefile

# 3) Run bootstrap/build/test
make setup
make build
make test
```

If scaffold files already exist and overwrite is intentional:

```bash
ios-app-manager init --config ios-app-manager.json --output . --force
```

## 2) Add a new feature module

```bash
# Create feature module
ios-app-manager module create Feed --type feature

# Verify module appears in catalog
ios-app-manager module list
```

Useful variants:

```bash
ios-app-manager module create CoreKit --type kit
ios-app-manager module create SharedUI --type ui
ios-app-manager module create Logger --type utility
```

## 3) Wire module dependencies

```bash
# Add internal module dependency
ios-app-manager dep add Feed --depends-on CoreKit

# Add external package to the module
ios-app-manager dep add-external \
  --url https://github.com/apple/swift-collections.git \
  --version 'branch: "main"' \
  --module Feed

# Inspect dependency graph
ios-app-manager dep list Feed
ios-app-manager dep list
```

Rollback/remove operations:

```bash
ios-app-manager dep remove Feed --depends-on CoreKit
ios-app-manager dep remove-external --package swift-collections
```

## 4) Build and test cycle

```bash
# Keep generated artifacts in sync
ios-app-manager generate makefile

# Project cycle
make build
make test

# Fast clean after local runs
ios-app-manager clean
```

For heavier cleanup:

```bash
ios-app-manager clean --deep
ios-app-manager clean --kill-xcode
```

## 5) Common troubleshooting

### Invalid or missing config

Symptoms:
- `load config:` errors from most commands.

Actions:
```bash
ios-app-manager init --config ./ios-app-manager.json --output .
```
- Ensure config path is correct and readable.
- Ensure required config fields are present (`app_name`, `bundle_id`, signing and push fields where needed).

### Existing files block `init`

Symptoms:
- Error mentions existing files and `--force`.

Actions:
```bash
ios-app-manager init --config ios-app-manager.json --output . --force
```

### Dependency changes fail

Symptoms:
- `dep add` rejects operation (for example circular dependency).

Actions:
```bash
ios-app-manager dep list
ios-app-manager dep list <module>
```
- Remove conflicting edges, then re-add the desired dependency.

### APNs send failures

Symptoms:
- `push send` fails with APNs status/reason.

Actions:
```bash
ios-app-manager push token
ios-app-manager push send --token <token> --env dev --payload ./push.json
```
- Verify config includes `push_key_path`, `push_key_id`, `team_id`, `bundle_id`.
- Verify token/environment match simulator/device and certificate scope.

### Entitlements lookup/edit issues

Symptoms:
- `entitlements` commands fail to resolve plist path.

Actions:
```bash
ios-app-manager entitlements --path App/App.entitlements list
ios-app-manager entitlements --path App/App.entitlements add aps-environment --value development
```

## Optional DSL-assisted workflow (scaffold-level)

DSL commands are available for agent scripting and parser validation:

```bash
ios-app-manager q 'summary() { message kind operation }'
ios-app-manager m 'create_module(name=Feed,type=feature)'
```

See [`dsl-reference.md`](dsl-reference.md) for the full grammar and operation list.
