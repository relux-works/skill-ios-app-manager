# SwiftPM executables, plugins, macros (swift-tools-version 6.2) + Tuist interop

Date: 2026-02-24  
Task: `TASK-260224-1p9xuj` (forensics-spm-executables)

## Scope / questions answered

- What can Swift Package Manager (SPM) model in `Package.swift` (package “types”, products, targets)?
- How to structure packages that ship **executables** (CLI tools) alongside libraries (and our Interface/Impl convention)?
- How **plugins** work (build tool plugins vs command plugins), and how they connect to executables.
- How **macros** are packaged (targets/products) and how they behave.
- What’s notable / different when using `// swift-tools-version: 6.2`.
- How **Tuist** and SPM interop in practice (two main approaches).

---

## Highlights / key takeaways

- **Executable delivery = product + executable target.** Use `.executableTarget(...)` for the module, and usually a `.executable(...)` product for a stable runnable name (e.g. `snapshotsdiff`). See example in `.temp/repos/membrana-app/scripts/makeSnapshotsDiffs/Package.swift:10`.  
- **Plugins “consume tools”.** Build tool plugins typically depend on an **executable target** (a tool you build from source) or a **binary target** (a prebuilt tool). This is an intentional SwiftPM design. (SE-0303; see “Proposed solution” and API in the proposal.)
- **Command plugins are permission-gated.** They can request e.g. write access to the package directory and require explicit user opt-in when invoked. This is why `swift package` often needs flags like `--allow-writing-to-package-directory` for some plugin workflows. See `swift-docc-plugin` usage docs.  
- **Macros are “host executables” under the hood.** The macro implementation target is built for the host and run during compilation, in a sandbox similar to package plugins. (Swift Forums pitch: “Custom macro support for SwiftPM”.)
- **Swift 6.2 adds manifest-level warning controls.** New `swiftSettings` helpers like `.treatAllWarnings(as:)` and `.treatWarning(_:as:)` exist for package targets. Also, macro build performance work in SwiftPM includes “pre-built swift-syntax dependencies.” (Swift 6.2 release notes.)
- **Tuist interop has 2 modes:** (1) **Xcode-native SPM integration** via `Project(packages:)` + `.package(product:)` deps (matches `.temp/repos/membrana-app/Project.swift`), or (2) **Tuist-managed external deps** via `Tuist/Package.swift` + `.external(...)` deps (Tuist “Dependencies” docs).

---

## 1) `Package.swift` essentials (syntax that matters)

### `swift-tools-version`

- The first line `// swift-tools-version: X.Y` declares the minimum Swift tools version required to build the package, and selects which `PackageDescription` API is available in the manifest. (SwiftPM docs: “Package.swift tools version”.)
- Because SwiftPM changed manifest behavior in Swift 5.8, **Foundation is not implicitly imported** into manifests anymore. In tools versions ≥ 5.8 (including 6.2), add `import Foundation` yourself when you use it. (Swift 5.8 release notes.)

Reference examples in this repo:
- `// swift-tools-version: 6.2` in `.research/ref_bsim-sdk-package.swift:1`
- `// swift-tools-version: 6.2` in `.temp/repos/membrana-app/scripts/cleanup-allure-report/Package.swift:1`

### Core manifest shape

At the top level:
- `name` (package name)
- `platforms` (optional, but recommended for iOS/macOS packages)
- `products` (what other packages/Xcode can depend on; also what `swift run` can run by a stable name)
- `dependencies` (external packages)
- `targets` (modules + tests + executables + plugins + macros)

---

## 2) “Package types” (what SPM can represent)

SwiftPM doesn’t have a single `type: ...` for packages; the package “kind” emerges from the **products/targets** you declare.

Common patterns:

### Library package

- Products: `.library(name:type:targets:)` (type can be `.static`, `.dynamic`, or omitted/automatic depending on SwiftPM/Xcode settings).
- Targets: `.target(...)` + `.testTarget(...)`.

### Executable (CLI/tool) package

- Products: `.executable(name:targets:)` (recommended when you want a stable runnable name).
- Targets: `.executableTarget(...)` (must contain `main.swift` or an `@main` entry point; SwiftPM docs).

Local example:
- `.temp/repos/membrana-app/scripts/makeSnapshotsDiffs/Package.swift:10` defines an executable product `snapshotsdiff` backed by `.executableTarget(name: "SnapshotsDiff", ...)`.

### Plugin package

- Product: `.plugin(name:targets:)`.
- Target: `.plugin(name:capability:dependencies:)`.
- Used by other packages/targets via `plugins: [...]` on a target. (SE-0303.)

### Macro package

- Product: `.macro(name:targets:)`.
- Target: `.macro(name:dependencies:swiftSettings:...)` (exact signature varies by tools version; see Swift Forums macro pitch for SwiftPM support).
- Typically also has a normal library target that exposes public API using the macro. (Swift Forums macro pitch + Swift.org “Packages with Macros”.)

### System library package

- Targets: `.systemLibrary(name:pkgConfig:providers:)` (wraps an OS/library dependency installed outside SwiftPM).

### Binary package

- Target: `.binaryTarget(name:url:checksum:)` or `.binaryTarget(name:path:)`.
- Often used as a tool dependency for build tool plugins (SE-0303 references this pattern).

---

## 3) Executables: recommended structures + gotchas

### Minimal executable package

Template:

```swift
// swift-tools-version: 6.2
import PackageDescription

let package = Package(
  name: "mytool",
  products: [
    .executable(name: "mytool", targets: ["mytool"]),
  ],
  targets: [
    .executableTarget(name: "mytool"),
  ]
)
```

### Executable + shared libraries (recommended split)

Keep reusable code in library targets; keep the entrypoint thin:

```
products: [
  .library(name: "MyLib", targets: ["MyLib"]),
  .executable(name: "mytool", targets: ["mytool"]),
]
targets: [
  .target(name: "MyLib"),
  .executableTarget(name: "mytool", dependencies: ["MyLib"]),
]
```

### Our required convention: Interface/Impl split (BSim pattern)

The “generated module” convention is explicitly documented in the task and matches the reference packages in `.research/`:

- Interface target == package name (path `Interface/`)
- Implementation target = `${Name}Impl` (path `Impl/`)
- Tests target = `${Name}Tests` (path `Tests/`)
- Products:
  - library `${Name}` → Interface target
  - library `${Name}Impl` → Impl target
- Impl depends on Interface
- Consumers depend only on Interface; cross-package “composition” packages can also depend on Impl for wiring.

See:
- `.research/ref_bsim-sdk-package.swift:12` (two library products) + `.research/ref_bsim-sdk-package.swift:25` (Interface/Impl targets).
- `.research/ref_bsim-id-package.swift:12` (two library products) + `.research/ref_bsim-id-package.swift:37` (Impl depends on both interface + other packages’ interface+impl).

### Adding a CLI executable to the convention

Add a third product/target:

- Product: `.executable(name: "<toolname>", targets: ["<toolname>"])`
- Target: `.executableTarget(name: "<toolname>", dependencies: ["<Name>Impl"])` (or Interface, if you want to forbid using internals).

This yields a package that can be both:
- embedded as libraries into an app, and
- run as a CLI (useful for codegen, validation, local utilities, CI scripts).

### “Script tool” pattern observed in a Tuist project (membrana-app)

In `.temp/repos/membrana-app/` there are several `scripts/<ToolName>/Package.swift` packages that define standalone executables (examples below):

- `.temp/repos/membrana-app/scripts/checkLocalization/Package.swift:12` → `.executableTarget(name: "checkLocalization")`
- `.temp/repos/membrana-app/scripts/prepareForBuild/Package.swift:8` → `.executableTarget(name: "prepareForBuild", path: "Sources")`
- `.temp/repos/membrana-app/scripts/makeSnapshotsDiffs/Package.swift:10` → `.executable(name: "snapshotsdiff", ...)`

That repo’s Xcode prebuild script runs a **prebuilt binary** (`checkLocalizationBinary`) rather than `swift run`, likely for speed/stability inside Xcode build phases:
- `.temp/repos/membrana-app/scripts/prebuild-script_check-localization.sh:6`

---

## 4) Plugins: build tool vs command plugin (and executable dependencies)

### Build tool plugins (generate files during build)

Design intent (SwiftPM):
- A build tool plugin runs as part of building a target and can create build commands.
- Plugins typically depend on a tool (executable target or binary target) that does the actual work. (SE-0303.)

Manifest-level building blocks (from SE-0303 proposal API):
- Define a plugin target:
  - `.plugin(name: "MyPlugin", capability: .buildTool(), dependencies: ["MyTool"])`
- Define the tool:
  - `.executableTarget(name: "MyTool", ...)` (or `.binaryTarget(...)`)
- Apply plugin to a target:
  - `.target(name: "MyLib", plugins: [.plugin(name: "MyPlugin")])`

### Command plugins (developer-invoked commands)

Command plugins are invoked manually, e.g. from the CLI or Xcode menus (depending on toolchain).

Common manifest pattern (from Swift forums “Plugin Explorer” post):

```swift
.plugin(
  name: "DocGen",
  capability: .command(
    intent: .custom(verb: "docgen", description: "Generate docs"),
    permissions: [
      .writeToPackageDirectory(reason: "Writes generated docs to Docs/")
    ]
  )
)
```

Operational reality (example: `swift-docc-plugin`):
- When you run some plugin commands, SwiftPM may require flags such as:
  - `--allow-writing-to-package-directory`
  - `--allow-writing-to-directory <path>`
- See `swift-docc-plugin` README for the exact invocation patterns and flags.

---

## 5) Macros in packages (SwiftPM support)

From SwiftPM’s perspective, macros are packaged and built similarly to plugins:

- You ship a `.macro` product that clients depend on.
- The macro implementation is built for the **host** and run during compilation.
- Macro implementations execute in a sandbox “similar to the sandbox used by Package Plugins”. (Swift Forums macro pitch.)

Typical package structure (conceptual):

- `MyMacros` (macro *product* clients depend on)
- `MyMacrosImpl` (macro implementation target; depends on `SwiftSyntax`/compiler plugin support)
- `MyLibrary` (regular library that exposes a public API using `@MyMacro`)
- `MyLibraryTests`

Swift 6.2 also calls out macro build performance work and a SwiftPM improvement:
- “SwiftPM now supports pre-built swift-syntax dependencies.” (Swift 6.2 release notes.)

---

## 6) Swift tools version 6.2: concrete, SPM-relevant specifics

### Manifest-level warning controls

Swift 6.2 introduces precise warning controls that are expressible in `Package.swift` using `swiftSettings`, e.g.:

```swift
.target(
  name: "MyLibrary",
  swiftSettings: [
    .treatAllWarnings(as: .error),
    .treatWarning("DeprecatedDeclaration", as: .warning),
  ]
)
```

Source: Swift 6.2 release notes (section “Precise warning control”).

### Macro build performance & swift-syntax packaging

Swift 6.2 release notes explicitly mention:
- macro build performance improvements, and
- that SwiftPM supports pre-built `swift-syntax` dependencies.

For macro-heavy packages, this is directly relevant (faster builds, less local toolchain friction).

---

## 7) Tuist + SwiftPM interop (what to copy into tuist-starter)

There are two primary ways Tuist interacts with Swift packages:

### A) “Xcode-native SPM integration” (Tuist just passes packages through)

- Define packages in `Project(packages: [...])`.
- Use `.package(product: "...")` dependencies in targets.

This pattern exists in the local reference Tuist project:
- Package list is passed into `Project(...)` at `.temp/repos/membrana-app/Project.swift:25`.
- Packages are chosen via an env var override (`TUIST_USE_LOCAL_PACKAGES`) at `.temp/repos/membrana-app/Project.swift:15`.
- Targets depend on SwiftPM products via `.package(product: ...)` at `.temp/repos/membrana-app/Project.swift:118`.

Tuist docs note this mode explicitly: “If you want to use Xcode’s default integration mechanism, you can pass the list `packages` when instantiating a project.”

### B) “Tuist-managed dependencies” (`Tuist/Package.swift` + `.external(...)`)

Tuist also supports managing dependencies through `Tuist/Package.swift`:
- Put a `Package.swift` either under `Tuist/` or at the root of the project.
- Optionally add a `PackageSettings` section to override product types (e.g., force a package product to be a `.framework`).
- Depend on these from your targets using `.external(name: ...)`. (Tuist “Dependencies” docs.)

This approach is useful when you need deterministic, repo-wide dependency settings or want Tuist to control how dependencies integrate.

---

## Open gaps / assumptions

- `tuist-akme` is not present in this workspace, so “how tuist-akme uses `Package.swift`” is inferred from:
  - the Tuist docs (general mechanisms), and
  - a local Tuist project that clearly uses Swift packages (`membrana-app`).

If `tuist-akme` is available elsewhere, the quickest validation is:
- locate `Tuist/Package.swift` (or root `Package.swift`) and check for `#if TUIST` `PackageSettings` usage,
- scan for `Project(packages:)` and `.package(product:)` usage patterns.

---

## Sources (fact-check links)

SwiftPM / Swift:
- SwiftPM manifest tools version docs: https://docs.swift.org/package-manager/PackageDescription/PackageDescription.html (see “Package.swift Tools Version”).  
- Swift 5.8 released (SwiftPM manifest change: no implicit `Foundation` import): https://www.swift.org/blog/swift-5.8-released/  
- Swift 6.2 released (warning controls in manifest; macro build perf; prebuilt swift-syntax): https://www.swift.org/blog/swift-6.2-released/  
- Swift.org “Packages with Macros”: https://www.swift.org/documentation/packageswithmacros/  

Plugins:
- SE-0303 “Package Manager Extensible Build Tools”: https://github.com/swiftlang/swift-evolution/blob/main/proposals/0303-swiftpm-extensible-build-tools.md  
- Swift Forums “Plugin Explorer” (command plugin manifest + permissions example): https://forums.swift.org/t/plugin-explorer/57519  
- `swift-docc-plugin` usage & permission flags: https://github.com/swiftlang/swift-docc-plugin  

Macros:
- Swift Forums pitch “Custom macro support for SwiftPM” (macro targets/products; sandbox model): https://forums.swift.org/t/pitch-2-custom-macro-support-for-swiftpm/69672  

Tuist:
- Tuist “Dependencies” guide (Xcode integration vs `Tuist/Package.swift` + `.external`): https://docs.tuist.dev/en/guides/develop/projects/dependencies  
- Tuist “Directory structure”: https://docs.tuist.dev/en/guides/develop/projects/directory-structure  
- Tuist `Project` model docs: https://docs.tuist.dev/en/references/projectdescription/project  

Local reference files (this repo checkout):
- BSim naming/structure convention examples:
  - `.research/ref_bsim-sdk-package.swift`
  - `.research/ref_bsim-id-package.swift`
  - `.research/ref_bsim-totp-package.swift`
- Tuist + SPM pattern examples (membrana-app):
  - `.temp/repos/membrana-app/Project.swift`
  - `.temp/repos/membrana-app/scripts/*/Package.swift`
  - `.temp/repos/membrana-app/scripts/prebuild-script_check-localization.sh`

