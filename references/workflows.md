# ios-app-manager workflows

This guide contains practical agent workflows using the current CLI.

## 1) Create a new project from scratch

```bash
# 1) Initialize scaffold from config
ios-app-manager init

# 2) Set up infrastructure (order matters — see diagrams/scaffolding-pipeline.puml)
ios-app-manager ioc setup
ios-app-manager relux setup
ios-app-manager secure-store setup --access-group group.org.xflow.app
ios-app-manager token-provider setup
ios-app-manager utilities setup

# 3) Create feature modules
ios-app-manager module blueprint Auth > auth.blueprint.json
# Edit auth.blueprint.json to configure data/UI layers, then:
ios-app-manager module create --from auth.blueprint.json
ios-app-manager module create Profile --type feature

# 4) Set up components that patch Registry (must be AFTER module create)
ios-app-manager http-client setup
ios-app-manager app-config setup

# 5) Generate Xcode project
tuist install && tuist generate
```

If scaffold files already exist and overwrite is intentional:

```bash
ios-app-manager init --force
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
ios-app-manager generate project-config
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

## 5) Bump app + extension manifest config

```bash
# 1) Edit the source-of-truth config
$EDITOR ios-app-manager.json

# 2) Sync scaffolded manifests
ios-app-manager generate project-config

# 3) Regenerate Tuist artifacts
tuist install && tuist generate
```

Leaf sync commands remain available when you only want one slice:

```bash
ios-app-manager generate versions
ios-app-manager generate min-target
```

## 6) Common troubleshooting

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
```

## Optional DSL-assisted workflow (scaffold-level)

DSL commands are available for agent scripting and parser validation:

```bash
ios-app-manager q 'summary() { message kind operation }'
ios-app-manager m 'create_module(name=Feed,type=feature)'
```

See [`dsl-reference.md`](dsl-reference.md) for the full grammar and operation list.
