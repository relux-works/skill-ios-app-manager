# Native macOS apps

## Scaffold gap

The iOS init pipeline exposes iPhone/iPad destinations, iOS deployment settings and iOS-specific capability helpers. A native macOS menu-bar app needs macOS destinations, LSUIElement, explicit sandbox policy and hostless unit tests. Do not manually patch generated Project.swift to achieve this.

## Owning plugin

`ios-app-manager generate macos-app --config ios-app-manager.json` is a standalone scaffold plugin (no iOS init dependency). It orchestrates manifest, package, app-entry and unit-test subgenerators. It generates a native Tuist app, a `<AppName>Core` framework and a hostless `<AppName>CoreTests` target; no UI-test target. `min_target` means macOS version and must be 13.0 or newer. Set `macos: {menu_bar: true, sandbox: false, packages: [{url: "https://example.com/package.git", version: "1.0.0", product: "Example"}]}` with the normal identity/version fields. Disable sandbox only for apps that need host process/service control. Explicit package versions are pinned exactly and exposed to the Core framework. The app imports its Core facade. Store business logic in Core, platform UI in the app.

Re-run this command to converge config-owned Project.swift and Workspace.swift. Source files are seeded only when absent, so handwritten code/tests survive regeneration. The separate Core Package.swift enables `swift test --package-path Packages/<AppName>Core` on macOS. Native Xcode packages and SwiftPM use the same configured exact versions. The package manifest is config-owned too. Do not mix the iOS `generate project-config` orchestration into this native scaffold; use `generate macos-app` for synchronization.

Generate with `tuist generate --no-open`, then build with `xcodebuild -workspace <AppName>.xcworkspace -scheme <AppName> -destination 'platform=macOS' build`. Sign using the configured DEVELOPMENT_TEAM; do not silently substitute another team. The generated app source is a starting point that can be implemented normally. MenuBarExtra lifecycle/composition stays in the app; services, state and reducers stay in Core.
