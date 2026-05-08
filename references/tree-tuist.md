# Tuist Operations Decision Tree

This tree documents the current `ios-app-manager` CLI workflows for Tuist-oriented project management.

## Start Here

Use this decision tree:

1. Need a brand-new project scaffold?
   Run [A. Project Init Flow](#a-project-init-flow-config---templates---scaffold---generate).
2. Need to create/list/delete modules?
   Run [B. Module CRUD](#b-module-crud-create-list-delete-with-types).
3. Need to change dependencies or manifests?
   Run [C. Manifest Management](#c-manifest-management-package-swift--projectswift).
4. Need to manage entitlements?
   Run [D. Entitlements](#d-entitlements-management).
5. Need Makefile/SwiftLint/project-config regeneration?
   Run [E. Generation](#e-generated-artifacts--project-config-regeneration).
6. Need compile/test/cleanup loop?
   Run [F. Build Cycle](#f-build-cycle-generate---build---test---clean).
7. Stuck on errors?
   Run [G. Troubleshooting](#g-troubleshooting--recovery-patterns).

## Working Variables

```bash
CONFIG=ios-app-manager.json
APP=DemoApp
MODULES=Packages
```

## A. Project Init Flow (Config -> Templates -> Scaffold -> Generate)

When to use:
- New Tuist project from config.
- Rebuild baseline project files from a known config.

Steps:

1. Prepare config (minimum required keys shown):

```json
{
  "app_name": "DemoApp",
  "bundle_id": "com.example.demo",
  "team_id": "ABCDE12345",
  "organization_name": "Example Org",
  "marketing_version": "1.0.0",
  "project_version": "1",
  "swift_version": "6.2",
  "min_target": "17.0",
  "modules_path": "Packages"
}
```

2. Initialize scaffold:

```bash
ios-app-manager init --config "$CONFIG" --output .
```

3. If files already exist and you want to overwrite scaffold-managed files:

```bash
ios-app-manager init --config "$CONFIG" --output . --force
```

4. Generate/update secondary artifacts from config:

```bash
ios-app-manager generate --config "$CONFIG" project-config
ios-app-manager generate --config "$CONFIG" makefile
ios-app-manager generate --config "$CONFIG" swiftlint
```

5. Run Tuist generation cycle:

```bash
make setup
# or:
tuist install
tuist generate
```

Resulting core files:
- `Tuist/Config.swift`
- `Project.swift`
- `Workspace.swift`
- `Package.swift`
- `Makefile`
- `.swiftlint.yml`
- `.periphery.yml`
- `${APP}.entitlements`

## B. Module CRUD (Create, List, Delete with Types)

When to use:
- Add new module package(s).
- Inspect module inventory.
- Remove a module and clean references.

Supported module types:
- `feature`
- `kit`
- `shared`
- `ui`
- `utility`

Create:

```bash
ios-app-manager module create Auth --type feature --config "$CONFIG"
ios-app-manager module create CoreKit --type kit --config "$CONFIG"
ios-app-manager module create SharedState --type shared --config "$CONFIG"
ios-app-manager module create DesignUI --type ui --config "$CONFIG"
ios-app-manager module create Logging --type utility --config "$CONFIG"
```

List:

```bash
ios-app-manager module list --config "$CONFIG"
```

Delete with prompt:

```bash
ios-app-manager module delete Auth --config "$CONFIG"
```

Delete without prompt:

```bash
ios-app-manager module delete Auth --config "$CONFIG" --force
```

## C. Manifest Management (`Package.swift` / `Project.swift`)

When to use:
- Add/remove internal module deps.
- Add/remove external Swift package deps.
- Keep project/workspace refs consistent after module CRUD.

Internal dependency (interface -> interface):

```bash
ios-app-manager dep add Auth --depends-on CoreKit --config "$CONFIG"
ios-app-manager dep remove Auth --depends-on CoreKit --config "$CONFIG"
```

External dependency (root + optional module linkage):

```bash
ios-app-manager dep add-external \
  --url https://github.com/apple/swift-collections.git \
  --version 'from: "1.0.0"' \
  --module Auth \
  --config "$CONFIG"

ios-app-manager dep remove-external --package swift-collections --config "$CONFIG"
```

Inspect dependencies:

```bash
ios-app-manager dep list --config "$CONFIG"
ios-app-manager dep list Auth --config "$CONFIG"
```

What gets updated automatically:
- `module create/delete` updates project/workspace package references.
- `dep add/remove` updates module `Package.swift` dependency blocks.
- `dep add-external/remove-external` updates root `Package.swift` and module manifest(s).

## D. Entitlements Management

When to use:
- Add/remove/list app entitlement keys in `${APP}.entitlements`.

List:

```bash
ios-app-manager entitlements list --config "$CONFIG"
```

Explicit plist path override:

```bash
ios-app-manager entitlements list --path ./DemoApp.entitlements
```

## E. Generated Artifacts + Project Config Regeneration

When to use:
- Config changed (app name, modules path, target versions, push settings).
- You need generated target updates while preserving custom Makefile section.

Commands:

```bash
ios-app-manager generate --config "$CONFIG" project-config
ios-app-manager generate --config "$CONFIG" makefile
ios-app-manager generate --config "$CONFIG" swiftlint
```

Project-config behavior:
- Runs scaffold config sync across root app + extension manifests.
- Currently syncs version markers, min deployment target markers, host app capabilities from `app_groups`, strict Swift compiler build flags, and package strictness.
- Config sync slices are scaffold generator plugins. Broad domains should orchestrate subplugins: `app-capabilities` is the host capability plugin, and `app-groups` is its concrete subplugin with `init` dependency.
- If a new scaffold concern does not belong to an existing plugin/subplugin, add a new pluggable generator/setup module instead of patching generated files by hand.

Makefile behavior:
- Generated section is rewritten from config.
- Custom section below marker is preserved.
- `build` and `test` regenerate Tuist project files before invoking `xcodebuild`.
- `build` and `test` call `clean-package-artifacts` on exit. The hook is a no-op by default and can be wired by overriding `PACKAGE_ARTIFACT_CLEANUP_CMD` in the custom section.
- `build` uses `BUILD_DESTINATION` (`generic/platform=iOS Simulator` by default); `test` uses `TEST_DESTINATION` (`$(DESTINATION)` by default).

## F. Build Cycle (Generate -> Build -> Test -> Clean)

When to use:
- Regular local development loop.
- CI-like validation.

Preferred loop:

```bash
make generate
make build
make test
make clean
```

Direct commands (equivalent intent):

```bash
tuist generate
make clean-package-artifacts # no-op unless PACKAGE_ARTIFACT_CLEANUP_CMD is overridden
xcodebuild -workspace "$APP.xcworkspace" -scheme "$APP" -destination 'generic/platform=iOS Simulator' build
xcodebuild -workspace "$APP.xcworkspace" -scheme "$APP" -destination 'platform=iOS Simulator,name=iPhone 16,OS=17.0' test
ios-app-manager clean
ios-app-manager clean --deep
```

## G. Troubleshooting + Recovery Patterns

1. `config file "...json" does not exist`
   Fix:
   - Pass `--config` explicitly.
   - Create the file, then rerun `ios-app-manager init`.

2. `validate config file ...`
   Fix:
   - Fill required fields: `app_name`, `bundle_id`, `team_id`, `swift_version`, `min_target`, `marketing_version`.
   - Use `major.minor` format for `swift_version` and `min_target`.

3. `module name "... " must be PascalCase`
   Fix:
   - Rename module to PascalCase (`AuthKit`, not `auth-kit`).

4. `module type is required` or `unknown module type`
   Fix:
   - Use one of: `feature|kit|shared|ui|utility`.

5. `module package already exists`
   Fix:
   - Choose a different module name, or delete old module first.

6. `module delete canceled`
   Fix:
   - Re-run with `--force` if deletion is intended.

7. `circular dependency: A -> B -> A`
   Fix:
   - Remove one dependency edge:
   ```bash
   ios-app-manager dep remove A --depends-on B --config "$CONFIG"
   ```

8. `target dependencies section not found` in dependency edits
    Fix:
    - Normalize manifest structure to include `dependencies: [ ... ]` in target/package blocks.
    - Re-run dependency command.

9. Tuist/Xcode cache corruption or stale artifacts
    Fix:
    ```bash
    ios-app-manager clean --deep --kill-xcode
    make setup
    ```
