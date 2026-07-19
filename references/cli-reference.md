# ios-app-manager CLI reference

## Binary

```bash
ios-app-manager [command] [flags]
```

## Global flags

| Flag | Description | Default |
|---|---|---|
| `-c, --config <path>` | Project config JSON path | `ios-app-manager.json` |
| `-v, --verbose` | Verbose output toggle | `false` |
| `--version` | Print version and exit | - |
| `-h, --help` | Help output | - |

## Command index

| Command | Description | Example |
|---|---|---|
| `init --config <path> --output <dir> [--force]` | Scaffold a new/updated project tree | `ios-app-manager init --config ios-app-manager.json --output .` |
| `module create <name> --type <type>` | Create a module package layout | `ios-app-manager module create Auth --type feature` |
| `module list` | List modules and metadata | `ios-app-manager module list` |
| `module delete <name> [--force]` | Delete module packages and references | `ios-app-manager module delete Auth --force` |
| `dep add <module> --depends-on <other>` | Add internal module dependency | `ios-app-manager dep add Auth --depends-on CoreKit` |
| `dep add-external --url <git_url> --version <ver> [--module <target>] [--product <product>] [--target-setting KEY=VALUE] [--app-target]` | Add external Swift package dependency | `ios-app-manager dep add-external --url https://github.com/apple/swift-collections.git --version 'branch: "main"' --module Auth` |
| `dep remove <module> --depends-on <other>` | Remove internal module dependency | `ios-app-manager dep remove Auth --depends-on CoreKit` |
| `dep remove-external --package <name>` | Remove external Swift package dependency | `ios-app-manager dep remove-external --package swift-collections` |
| `dep list [<module>]` | List internal and external dependencies | `ios-app-manager dep list Auth` |
| `entitlements list` | Print effective entitlement values | `ios-app-manager entitlements list` |
| `generate makefile` | Generate or update `Makefile` | `ios-app-manager generate makefile` |
| `generate swiftlint` | Generate or update `.swiftlint.yml` | `ios-app-manager generate swiftlint` |
| `generate project-config` | Sync manifest config slices across app + extensions | `ios-app-manager generate project-config` |
| `generate bundle-id` | Generate or update app + extension bundle identifiers | `ios-app-manager generate bundle-id` |
| `generate versions` | Generate or update app + extension versions | `ios-app-manager generate versions` |
| `generate min-target` | Generate or update app + extension deployment target markers | `ios-app-manager generate min-target` |
| `generate team-id` | Generate or update app + extension signing team settings | `ios-app-manager generate team-id` |
| `generate background-modes-config` | Generate or update host app background modes Info.plist key | `ios-app-manager generate background-modes-config` |
| `generate application-configuration` | Generate or update product-level runtime app configuration | `ios-app-manager generate application-configuration` |
| `generate runtime-profiles` | Generate typed distribution/backend runtime policy and Tuist profiles | `ios-app-manager generate runtime-profiles` |
| `generate app-capabilities` | Generate or update host app capabilities from config | `ios-app-manager generate app-capabilities` |
| `generate presentation-config` | Generate or update host app theme/orientation Info.plist keys | `ios-app-manager generate presentation-config` |
| `generate export-compliance-config` | Generate or update host app export compliance Info.plist key | `ios-app-manager generate export-compliance-config` |
| `generate privacy-usage-descriptions-config` | Generate or update host app privacy usage description Info.plist keys | `ios-app-manager generate privacy-usage-descriptions-config` |
| `generate build-flags` | Generate or update strict Swift compiler build flags in app + extension manifests | `ios-app-manager generate build-flags` |
| `clean [--deep] [--kill-xcode]` | Clean local/global build artifacts | `ios-app-manager clean --deep` |
| `profile build` | Profile build timing and target critical path | `ios-app-manager profile build --jobs 8` |
| `profile layout scaffold` | Generate XCTest rendered layout hierarchy XML helper | `ios-app-manager profile layout scaffold` |
| `profile layout analyze --input <xml-or-log>` | Analyze rendered hierarchy XML for agent-readable layout diagnostics | `ios-app-manager profile layout analyze --input layout.xml` |
| `profile runtime scaffold` | Generate debug-only Swift runtime profiling helper | `ios-app-manager profile runtime scaffold` |
| `profile runtime analyze --input <log>` | Analyze `IAM_PROFILE` runtime log lines | `ios-app-manager profile runtime analyze --input runtime.log` |
| `profile runtime errors [--input <log>]` | Analyze unified-log runtime errors/faults and `IAM_ERROR` lines | `ios-app-manager profile runtime errors --input errors.log` |
| `push send --token <token> [--env dev\|prod] [--payload <file>]` | Send APNs push using project credentials | `ios-app-manager push send --token "$TOKEN" --env dev` |
| `push token` | Print latest device token from logs/fallback file | `ios-app-manager push token` |
| `ioc setup` | Set up SwiftIoC container and Registry.swift | `ios-app-manager ioc setup` |
| `relux setup` | Set up Relux state management scaffolding | `ios-app-manager relux setup` |
| `secure-store setup --access-group <group>` | Create SecureStore keychain wrapper module | `ios-app-manager secure-store setup --access-group group.org.xflow.app` |
| `token-provider setup` | Create TokenProvider module | `ios-app-manager token-provider setup` |
| `utilities setup` | Create Utilities module | `ios-app-manager utilities setup` |
| `foundation-plus setup` | Create FoundationPlus utility module | `ios-app-manager foundation-plus setup` |
| `swiftui-plus setup` | Create SwiftUIPlus utility module | `ios-app-manager swiftui-plus setup` |
| `test-targets setup --unit-target <name> --ui-target <name>` | Create unit and/or UI test targets through test-target subplugins | `ios-app-manager test-targets setup --unit-target DemoAppTests --ui-target DemoAppUITests` |
| `app-extensions setup` | Create SharedKit + Extensions/ base | `ios-app-manager app-extensions setup` |
| `notification-service setup` | Create Notification Service Extension wrapper + Core package | `ios-app-manager notification-service setup` |
| `widget-base setup` | Create thin WidgetBundle wrapper + WidgetCore package | `ios-app-manager widget-base setup` |
| `app-intents setup` | Add AppIntent scaffold to WidgetCore | `ios-app-manager app-intents setup` |
| `static-widget setup` | Create static timeline widget internals in WidgetCore | `ios-app-manager static-widget setup` |
| `live-activity setup` | Create Live Activity + Dynamic Island scaffold with widget UI in WidgetCore | `ios-app-manager live-activity setup` |
| `http-client setup` | Add HttpClient IoC registration with swift-httpclient | `ios-app-manager http-client setup` |
| `app-config setup` | Scaffold AppConfig manager and consume typed runtime policy when configured | `ios-app-manager app-config setup` |
| `status` | Project status placeholder command | `ios-app-manager status` |
| `q '<query>'` | Run DSL query expression | `ios-app-manager q 'modules(type=feature) { operation params }'` |
| `m '<mutation>'` | Run DSL mutation expression | `ios-app-manager m 'create_module(name=Auth,type=feature)'` |

## Command reference

### `test-targets setup`

Syntax:
```bash
ios-app-manager test-targets setup [--unit-target <name>] [--ui-target <name>] [--yes]
```

Description:
- Orchestrates test target scaffolding through separate unit-test-target and UI-test-target subplugins.
- Requires at least one explicit target name.
- Adds `.unitTests` and/or `.uiTests` targets to `Project.swift`.
- Creates starter Swift source under `Targets/<TargetName>/Sources/`.
- Is idempotent: reruns do not duplicate targets or overwrite existing starter source files.

Examples:
```bash
ios-app-manager test-targets setup --unit-target DemoAppTests --yes
ios-app-manager test-targets setup --ui-target DemoAppUITests --yes
ios-app-manager test-targets setup --unit-target DemoAppTests --ui-target DemoAppUITests --yes
```

### `notification-service setup`

Syntax:
```bash
ios-app-manager notification-service setup [--extension-target <name>] [--bundle-id-suffix <suffix>] [--yes]
```

Description:
- Creates a Notification Service Extension target under `Extensions/<ExtensionName>/`.
- Keeps the `.appex` target thin: the generated wrapper subclasses `UNNotificationServiceExtension` and delegates to Core.
- Creates `<ExtensionName>Core` as a SwiftPM package under the extension directory.
- Adds a package-level Swift Testing target for the Core package.
- Links the extension target to `<ExtensionName>Core` and `UserNotifications`.
- Adds `NSExtensionPrincipalClass` and the notification service extension point to the extension Info.plist.
- Is idempotent: reruns do not duplicate manifest dependencies or Core test targets.

Examples:
```bash
ios-app-manager notification-service setup --yes
ios-app-manager notification-service setup \
  --extension-target VideoCallNotificationService \
  --bundle-id-suffix notification-service \
  --yes
```

### `init`

Syntax:
```bash
ios-app-manager init --config <path> --output <dir> [--force]
```

Description:
- Loads config and scaffolds project files to the output directory.
- Without `--force`, existing scaffold files are protected from overwrite.

Flags:
- `--config <path>`: Config file path.
- `--output <dir>`: Target directory (default `.`).
- `--force`: Overwrite scaffold files if they exist.

Examples:
```bash
ios-app-manager init --config ios-app-manager.json --output .
ios-app-manager init --config ios-app-manager.json --output ./DemoApp --force
```

### `module create`

Syntax:
```bash
ios-app-manager module create <name> --type <feature|kit|shared|ui|utility>
```

Description:
- Creates module packages/files according to Relux conventions.
- Module name is validated (PascalCase).

Flags:
- `--type <type>`: Required module kind.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager module create Auth --type feature
```

### `module list`

Syntax:
```bash
ios-app-manager module list
```

Description:
- Prints module table with name, type, package names, and dependency count.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager module list
```

### `module delete`

Syntax:
```bash
ios-app-manager module delete <name> [--force]
```

Description:
- Deletes module packages and cleans manifest references.
- Interactive confirmation is required unless `--force` is set.

Flags:
- `--force`: Skip confirmation prompt.
- `--config <path>`: Optional command-level config override.

Examples:
```bash
ios-app-manager module delete Auth
ios-app-manager module delete Auth --force
```

### `dep add`

Syntax:
```bash
ios-app-manager dep add <module> --depends-on <other>
```

Description:
- Adds an internal dependency from `<module>` to `<other>`.
- Detects invalid dependency changes (for example, circular dependencies).

Flags:
- `--depends-on <other>`: Required dependency module.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep add Auth --depends-on CoreKit
```

### `dep add-external`

Syntax:
```bash
ios-app-manager dep add-external --url <git_url> --version <ver> [--module <target>] [--product <product>] [--target-setting KEY=VALUE] [--app-target]
```

Description:
- Adds an external Swift package dependency.
- Optional `--module` links the package to a specific module package manifest.

Flags:
- `--url <git_url>`: Required package repository URL.
- `--version <ver>`: Required version requirement.
- `--module <target>`: Optional target module scope.
- `--product <product>`: Optional Swift product name when it differs from the package name. Repeat for multiple products.
- `--target-setting KEY=VALUE`: Optional Tuist `PackageSettings.targetSettings` build setting applied to the product(s). Repeat for multiple settings.
- `--app-target`: Optional link of the product(s) into the host app target in `Project.swift`.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep add-external \
  --url https://github.com/apple/swift-collections.git \
  --version 'branch: "main"' \
  --module Auth
```

### `dep remove`

Syntax:
```bash
ios-app-manager dep remove <module> --depends-on <other>
```

Description:
- Removes an internal dependency relation.

Flags:
- `--depends-on <other>`: Required dependency module.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep remove Auth --depends-on CoreKit
```

### `dep remove-external`

Syntax:
```bash
ios-app-manager dep remove-external --package <name>
```

Description:
- Removes an external package from project/module manifests.

Flags:
- `--package <name>`: Required package name.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep remove-external --package swift-collections
```

### `dep list`

Syntax:
```bash
ios-app-manager dep list [<module>]
```

Description:
- Prints internal dependency table and external package table.
- With `<module>`, output is scoped to one module and root/module-level external deps.

Flags:
- `--config <path>`: Optional command-level config override.

Examples:
```bash
ios-app-manager dep list
ios-app-manager dep list Auth
```

### `entitlements list`

Syntax:
```bash
ios-app-manager entitlements list
```

Description:
- Lists entitlement keys and resolved values.

Flags:
- `-p, --path <plist>`: Optional explicit entitlements file path.
- `--config <path>`: Used when `--path` is not provided.

Examples:
```bash
ios-app-manager entitlements list
ios-app-manager entitlements --path App/App.entitlements list
```

### `generate makefile`

Syntax:
```bash
ios-app-manager generate makefile
```

Description:
- Creates `Makefile` if missing, or regenerates the managed section while preserving custom section.
- Generated `build` and `test` targets run `tuist generate` before `xcodebuild` and call the `clean-package-artifacts` hook on exit.
- Project-specific cleanup should be injected by overriding `PACKAGE_ARTIFACT_CLEANUP_CMD` in the preserved custom section; the generated default is a no-op.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate makefile
```

### `generate swiftlint`

Syntax:
```bash
ios-app-manager generate swiftlint
```

Description:
- Creates `.swiftlint.yml` if missing, or regenerates it from project config.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate swiftlint
```

### `generate bundle-id`

Syntax:
```bash
ios-app-manager generate bundle-id
```

Description:
- Syncs `bundle_id` from config into scaffold-managed `Project.swift` manifests.
- Updates the host app manifest and every `Extensions/*/Project.swift` generated from scaffold templates.
- Host app targets converge to the configured app bundle id.
- Extension targets preserve their suffix and derive the final bundle id from the configured host bundle root, for example `<bundle_id>.widget`.
- Intended dependency: `init` scaffold must already exist.

Flags:
- `--config <path>`: Config file path.

Example:
```bash
ios-app-manager generate bundle-id
```

### `generate versions`

Syntax:
```bash
ios-app-manager generate versions
```

Description:
- Syncs `marketing_version` and `project_version` from config into scaffold-managed `Project.swift` manifests.
- Updates the host app manifest and every `Extensions/*/Project.swift` generated from scaffold templates.
- Intended dependency: `init` scaffold must already exist.

Flags:
- `--config <path>`: Config file path.

Example:
```bash
ios-app-manager generate versions
```

### `generate min-target`

Syntax:
```bash
ios-app-manager generate min-target
```

Description:
- Syncs `min_target` from config into scaffold-managed `Project.swift` manifests.
- Updates the host app manifest and every `Extensions/*/Project.swift` generated from scaffold templates.
- Canonicalizes both `deploymentTargets: .iOS(minTarget)` and `"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`.
- Older manifests without explicit `minTarget` markers are migrated forward during sync.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate min-target
```

### `generate team-id`

Syntax:
```bash
ios-app-manager generate team-id
```

Description:
- Syncs `team_id` from config into scaffold-managed `Project.swift` manifests.
- Updates the host app manifest and every `Extensions/*/Project.swift` generated from scaffold templates.
- Canonicalizes `let developmentTeam = "<team_id>"` and `"DEVELOPMENT_TEAM": .string(developmentTeam)` across app, test, app-like, and extension build settings.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate team-id
```

### `generate background-modes-config`

Syntax:
```bash
ios-app-manager generate background-modes-config
```

Description:
- Syncs host app background mode values from `ios-app-manager.json`.
- `background_modes` is optional and values are validated against Apple's documented `UIBackgroundModes` strings.
- `audio` writes the iOS background mode for Audio, AirPlay, and Picture in Picture.
- `remote-notification` writes the APNs background notification mode for apps that implement the remote notification fetch callback.
- `voip` writes the VoIP background mode only when explicitly listed.
- Unknown values are rejected by config validation.
- Omitting the field or using an empty list removes the scaffold-owned `UIBackgroundModes` key.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate background-modes-config
```

### `generate presentation-config`

Syntax:
```bash
ios-app-manager generate presentation-config
```

Description:
- Syncs host app presentation values from `ios-app-manager.json`.
- `theme` supports `automatic`, `light`, and `dark`.
- `orientation` supports `automatic`, `portrait`, and `landscape`.
- `automatic` removes the owned Info.plist key and lets iOS use its default behavior.
- `light`/`dark` write `UIUserInterfaceStyle` for the host app target only.
- `portrait`/`landscape` write `UISupportedInterfaceOrientations` for the host app target only.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate presentation-config
```

### `generate export-compliance-config`

Syntax:
```bash
ios-app-manager generate export-compliance-config
```

Description:
- Syncs host app export compliance values from `ios-app-manager.json`.
- `uses_non_exempt_encryption` is optional and must be explicit when the key should exist.
- `false` writes `ITSAppUsesNonExemptEncryption` as `.boolean(false)` for the host app target only.
- Omitting the field removes the scaffold-owned Info.plist key.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate export-compliance-config
```

### `generate privacy-usage-descriptions-config`

Syntax:
```bash
ios-app-manager generate privacy-usage-descriptions-config
```

Description:
- Syncs host app privacy usage description values from `ios-app-manager.json`.
- `privacy_usage_descriptions.bluetooth_always` writes `NSBluetoothAlwaysUsageDescription` for the host app target only.
- `privacy_usage_descriptions.bluetooth_peripheral` writes `NSBluetoothPeripheralUsageDescription` for the host app target only.
- `privacy_usage_descriptions.camera` writes `NSCameraUsageDescription` for the host app target only.
- `privacy_usage_descriptions.microphone` writes `NSMicrophoneUsageDescription` for the host app target only.
- `privacy_usage_descriptions.local_network` writes `NSLocalNetworkUsageDescription` for the host app target only.
- Omitting or emptying a field removes the scaffold-owned Info.plist key.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate privacy-usage-descriptions-config
```

### `generate application-configuration`

Syntax:
```bash
ios-app-manager generate application-configuration
```

Description:
- Syncs product-level runtime app identity from `ios-app-manager.json`.
- Writes an `ApplicationConfiguration` Info.plist dictionary into scaffold-managed app-like targets.
- Generates the configured `SharedConfig` package sources:
  - `InfoPlistReading.swift`
  - `ApplicationConfiguration.swift`
- Generates app-target `Configuration+ApplicationConfiguration.swift`.
- Wires the configured `SharedConfig` dependency into root `Package.swift` and target manifests.
- Keeps target identity separate: extensions keep their own `CFBundleIdentifier`, while `ApplicationConfiguration.applicationBundleIdentifier` points to the containing application bundle id.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate application-configuration
```

### `generate runtime-profiles`

Syntax:
```bash
ios-app-manager generate runtime-profiles
```

Description:
- Validates the versioned `runtime_profiles` schema and approved distribution-policy boundaries.
- Validates environment-keyed Firebase public metadata against operator-supplied plist paths named by environment variables; local paths and API keys are not retained or printed.
- Generates typed Swift backend/profile descriptors with exact API origins and no synthesized URL path.
- Generates Tuist configurations and shared schemes for `pilotTestFlight`, `appStore`, `internal`, and `tests`, including matching package-project configurations.
- Adds the selected profile to generated `ApplicationConfiguration` and converges managed output on update or removal.
- Orchestrates schema, Firebase-input, runtime-descriptor, and Tuist-project subplugins in dependency order.

Flags:
- `--config <path>`: Optional command-level config override.

Example and schema:
- [`runtime-profiles.md`](runtime-profiles.md)
- [`runtime-profiles.schema.json`](runtime-profiles.schema.json)

### `generate app-capabilities`

Syntax:
```bash
ios-app-manager generate app-capabilities
```

Description:
- Orchestrates host app capability subplugins.
- `app-groups` is the current capability subplugin. It syncs configured `app_groups` from `ios-app-manager.json` into `Tuist/ProjectDescriptionHelpers/AppCapabilities.swift`, host `Project.swift` Info.plist keys, generated `SharedConfig`, and `Configuration+AppGroups.swift`.
- Generated app-group code reads product-level service identity through `Configuration.ApplicationConfiguration`; use `generate project-config` when syncing app groups so the application configuration facade is present.
- Runs as a scaffold generator plugin with explicit `init` dependency. Concrete capability concerns should be added as subplugins rather than collected into one large capability function.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate app-capabilities
```

### `generate build-flags`

Syntax:
```bash
ios-app-manager generate build-flags
```

Description:
- Syncs a strict Swift compiler baseline into scaffold-managed `Project.swift` manifests.
- Updates the host app manifest and every `Extensions/*/Project.swift` generated from scaffold templates.
- Canonicalizes Swift strictness build settings such as strict memory safety, strict concurrency checking, approachable concurrency, default actor isolation, and selected upcoming feature toggles.
- Values are loaded from `project_settings.swift` in `ios-app-manager.json`; omitted values use the scaffold strict baseline.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate build-flags
```

### `generate project-config`

Syntax:
```bash
ios-app-manager generate project-config
```

Description:
- Orchestrates project-manifest config sync across scaffolded app and extension manifests.
- Runs the current config-sync leaf plugins in order:
  - `generate bundle-id`
  - `generate versions`
  - `generate min-target`
  - `generate team-id`
  - `generate platform-destinations`
  - `generate background-modes-config`
  - `generate presentation-config`
  - `generate export-compliance-config`
  - `generate privacy-usage-descriptions-config`
  - `generate application-configuration`
  - `generate runtime-profiles`
  - `generate app-capabilities`
  - `generate build-flags`
  - `generate package-strictness`
- Prints one summary that shows the outcome of each leaf sync.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate project-config
```

### `clean`

Syntax:
```bash
ios-app-manager clean [--deep] [--kill-xcode]
```

Description:
- Default mode performs quick project clean.
- `--deep` includes global cache cleanup.
- `--kill-xcode` kills Xcode first and implies deep clean.

Flags:
- `--deep`: Deep clean mode.
- `--kill-xcode`: Kill Xcode before clean; implies deep clean.

Examples:
```bash
ios-app-manager clean
ios-app-manager clean --deep
ios-app-manager clean --kill-xcode
```

### `profile build`

Syntax:
```bash
ios-app-manager profile build [flags]
```

Description:
- Runs or analyzes an Xcode build with `-showBuildTimingSummary`.
- Uses Tuist graph JSON to estimate target dependency critical path when graph analysis is enabled.
- Writes run artifacts under `.temp/build-profile/<timestamp>/` by default.
- Supports text and JSON reports.

Flags:
- `--workspace <name>`: Workspace to build. Defaults to `<app_name>.xcworkspace`.
- `--scheme <name>`: Scheme to build. Defaults to `product_name` or `app_name`.
- `--configuration <name>`: Build configuration. Default `Debug`.
- `--destination <specifier>`: xcodebuild destination. Default `generic/platform=iOS Simulator`.
- `--derived-data-path <path>`: DerivedData path.
- `--result-bundle-path <path>`: `.xcresult` path.
- `--log <path>`: Analyze an existing xcodebuild log instead of running a build.
- `--graph-json <path>`: Analyze an existing `tuist graph --format legacyJSON` file.
- `--output-root <path>`: Artifact directory.
- `--skip-generate`: Skip `tuist generate --no-open` before build.
- `--skip-graph`: Skip target graph analysis.
- `--parallelize-targets`: Pass `-parallelizeTargets` to xcodebuild. Default true.
- `--jobs <n>`: Pass `-jobs`.
- `--xcodebuild-arg <arg>`: Extra xcodebuild argument; repeat for multiple arguments.
- `--format <text|json>`: Report format.

Examples:
```bash
ios-app-manager profile build
ios-app-manager profile build --jobs 8 --configuration Debug
ios-app-manager profile build --log .temp/build-profile/xcodebuild.log --skip-graph
ios-app-manager profile build --format json > .temp/build-profile/report.json
```

### `profile layout scaffold`

Syntax:
```bash
ios-app-manager profile layout scaffold [--output <path>] [--force]
```

Description:
- Writes `LayoutHierarchyProbe.swift` under `Targets/<AppName>UITests/Sources/Diagnostics/` by default.
- The helper is intended for UI test targets and imports XCTest.
- Provides `LayoutHierarchyProbe.xml(for:)` and `XCTestCase.attachLayoutHierarchyXML(...)`.
- Serializes the rendered XCTest/accessibility hierarchy into XML with element type, path, identifier, label, value, state, and screen-coordinate frame attributes.
- Attaches the XML to the `.xcresult` and prints it between `IAM_LAYOUT_XML_START` / `IAM_LAYOUT_XML_END` markers.

Example:
```bash
ios-app-manager profile layout scaffold
```

### `profile layout analyze`

Syntax:
```bash
ios-app-manager profile layout analyze --input <xml-or-log> [flags]
```

Description:
- Parses `LayoutHierarchyProbe` XML, Appium/WebDriverAgent page source XML, or UI-test logs containing `IAM_LAYOUT_XML_*` markers.
- Prints an agent-readable rendered hierarchy tree.
- Reports element type counts, max depth, duplicate accessibility identities, missing interactive identity, tiny tap targets, and offscreen frames.
- Supports text and JSON reports.

Flags:
- `--input <path>`: Required XML/log path.
- `--min-tap-size <points>`: Minimum interactive frame width/height. Default `44`.
- `--max-elements <n>`: Maximum hierarchy elements included in the report. Default `200`.
- `--include-hidden`: Include explicitly hidden elements in issue detection.
- `--format <text|json>`: Report format.

Examples:
```bash
ios-app-manager profile layout analyze --input .temp/layout/feed.xml
ios-app-manager profile layout analyze --input .temp/layout/ui-test.log --format json
```

### `profile runtime scaffold`

Syntax:
```bash
ios-app-manager profile runtime scaffold [--output <path>] [--force]
```

Description:
- Writes `PerformanceProbe.swift` under `Targets/<AppName>/Sources/Diagnostics/` by default.
- The helper is wrapped in `#if DEBUG`.
- Provides SwiftUI `.profiled("ViewName")`, `PerformanceProbe.measure`, `PerformanceProbe.measureAsync`, and `PerformanceProbe.event`.
- Provides `PerformanceProbe.markAppStart()` and `.firstRenderProfiled("RootView")` to measure app launch to first meaningful render.
- Provides `PerformanceProbe.error(...)` for deterministic app-level error events.
- Emits both signposts and structured `IAM_PROFILE {json}` lines.

Example:
```bash
ios-app-manager profile runtime scaffold
```

### `profile runtime analyze`

Syntax:
```bash
ios-app-manager profile runtime analyze --input <log> [flags]
```

Description:
- Parses `IAM_PROFILE` JSON lines from a captured app log.
- Groups events by kind/name and reports count, total duration, average, max, main-thread count, slow count, repeated-call warnings, and main-thread slow-call warnings.
- Reports app startup-to-first-render timing when `app_start` and `first_render` markers are present.

Flags:
- `--input <path>`: Required log path.
- `--slow-ms <n>`: Main-thread duration threshold. Default `16`.
- `--repeat-threshold <n>`: Repeated-call warning threshold. Default `50`.
- `--format <text|json>`: Report format.

Example:
```bash
ios-app-manager profile runtime analyze --input .temp/runtime-profile.log --slow-ms 16 --repeat-threshold 50
```

### `profile runtime errors`

Syntax:
```bash
ios-app-manager profile runtime errors [--input <log>] [flags]
```

Description:
- Parses runtime error signals from unified-log NDJSON/plain output and `IAM_ERROR` structured lines.
- Without `--input`, collects recent host logs with `log show --style ndjson --last <duration> --predicate '(logType == "error" OR logType == "fault")'`.
- With `--simulator`, collects logs through `xcrun simctl spawn <device> log show`.
- Groups by severity, process, subsystem, category, and normalized message signature.
- Adds hints for crash, exception, hang, thread-checker, and SwiftUI background-publish messages.

Flags:
- `--input <path>`: Optional log file path.
- `--last <duration>`: Time window for collection when `--input` is omitted. Default `10m`.
- `--predicate <predicate>`: Custom unified-log predicate.
- `--process <name>`: Filter collected logs by process.
- `--subsystem <id>`: Filter collected logs by subsystem.
- `--category <name>`: Filter collected logs by category.
- `--simulator`: Collect from simulator.
- `--device <udid|booted>`: Simulator device for `--simulator`.
- `--include-default`: Include non-error unified-log entries from NDJSON input.
- `--max-examples <n>`: Example messages per group. Default `3`.
- `--format <text|json>`: Report format.

Examples:
```bash
ios-app-manager profile runtime errors --process DemoApp --last 15m
ios-app-manager profile runtime errors --simulator --device booted --subsystem com.example.demo
ios-app-manager profile runtime errors --input .temp/runtime-errors.log --format json
```

### `push send`

Syntax:
```bash
ios-app-manager push send --token <token> [--env dev|prod] [--payload <file>]
```

Description:
- Sends APNs push request using credentials from config.
- Fails when APNs returns a non-2xx status.

Flags:
- `--token <token>`: Required device token.
- `--env <dev|prod>`: APNs environment (default `dev`).
- `--payload <file>`: Optional JSON payload file.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager push send --token "$DEVICE_TOKEN" --env dev --payload ./push.json
```

### `push token`

Syntax:
```bash
ios-app-manager push token
```

Description:
- Prints latest APNs token detected from simulator logs.
- Can use explicit fallback token file.

Flags:
- `--token-file <path>`: Optional fallback token file.
- `--config <path>`: Optional config; used to read fallback token path if set.

Examples:
```bash
ios-app-manager push token
ios-app-manager push token --token-file ./tmp/device-token.txt
```

### `status`

Syntax:
```bash
ios-app-manager status
```

Description:
- Placeholder command in current scaffold.
- Current output: `not implemented`.

Example:
```bash
ios-app-manager status
```

### `q`

Syntax:
```bash
ios-app-manager q '<query>'
```

Description:
- Parses and executes DSL query expression (`operation(params) { fields }`).

Flags:
- `--format pretty|compact`: Output format (default `pretty`).

Examples:
```bash
ios-app-manager q 'summary() { message kind operation }'
ios-app-manager q --format compact 'deps(module=Auth) { operation params }'
```

### `m`

Syntax:
```bash
ios-app-manager m '<mutation>'
```

Description:
- Parses and executes DSL mutation expression (`operation(params)`).

Flags:
- `--format pretty|compact`: Output format (default `pretty`).

Examples:
```bash
ios-app-manager m 'create_module(name=Auth,type=feature)'
ios-app-manager m --format compact 'add_dep(module=Auth,depends_on=CoreKit)'
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Command succeeded |
| `1` | Command failed (argument/flag validation, parse/runtime errors, APNs failure, unsupported DSL operation) |
