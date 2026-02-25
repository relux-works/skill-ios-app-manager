# SPM intra-package linking (Interface → Impl) + Tuist override research

- Task: `TASK-260224-1t59sd`
- Date: 2026-02-24
- Artifacts: `.temp/linking-test/`

## Question

We want **Interface** and **Impl** to be **separate dynamic libraries** (our “all-dynamic” rule). With SwiftPM, we currently model this as:

- One `Package.swift`
- `Interface` target + `Impl` target
- `Impl` target depends on `Interface` target
- Two products (`Interface` and `Impl`)

The concern: **within a single Swift package**, a target-to-target dependency is effectively **statically linked into the consuming product**, meaning the `Impl` dynamic library *contains* the `Interface` code/symbols instead of dynamically linking the `Interface` dynamic library.

## Key takeaways (highlights)

1. **Confirmed:** In a single package with two `.dynamic` library products, `Impl` **does not** dynamically link `Interface`. `Interface` symbols end up inside `Impl` as well (duplication).
2. **Confirmed:** Splitting into **two packages** results in `Impl` dynamically linking `Interface` (visible in `otool -L`).
3. **Tuist cannot “fix” this.** Tuist can choose how a *package product* is integrated (e.g., framework/static framework) via `PackageSettings`, but it can’t change how SwiftPM composes a product from intra-package targets. The only reliable way to force a dynamic boundary between Interface and Impl is **separate packages**.

## Why this happens (SPM model)

SwiftPM target dependencies are expressed as **target → target** within the same package (e.g. `.target(name:)`, `.byName(name:)`). There’s no supported way (tools v6+) to express “this target depends on that *product* from the same package” (product deps require a `package:`). See `PackageDescription.Target.Dependency` API in Xcode toolchain:

- `Target.Dependency` cases include `targetItem` and `productItem` (the latter requires a package).  
  Source: `/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/lib/swift/pm/ManifestAPI/PackageDescription.swiftmodule/arm64-apple-macos.swiftinterface`:
  - `Target.Dependency` cases around line ~`827`
  - `.product(name:package:)` overloads around lines ~`951–967` (tools v6+ requires a `package:` string)

Empirically, when you build `MyFeatureImpl` as a dynamic library product and it depends on `MyFeature` **target** in the same package, SwiftPM links `MyFeature`’s object code into `MyFeatureImpl` (so `MyFeatureImpl` does not carry a dynamic dependency on `MyFeature`).

## Experiments (actual code + Mach-O verification)

All test code lives under `.temp/linking-test/`:

- **Single-package:** `.temp/linking-test/single-package/Packages/MyFeaturePackage`
- **Two-packages:** `.temp/linking-test/two-packages/MyFeature`, `.temp/linking-test/two-packages/MyFeatureImpl`, `.temp/linking-test/two-packages/TwoPackageApp`

### Important note for this environment (Codex sandbox)

SwiftPM and Tuist try to write user caches under `/Users/alexis/...` by default, which is **not writable** in this sandbox. To make builds reproducible here:

- Set a writable `HOME` (e.g. `HOME=/tmp/spm-home`)
- Use `swift build --disable-sandbox` (SwiftPM otherwise uses `sandbox-exec`, which is blocked here)

This does **not** affect the linking semantics we’re investigating; it only makes the tooling runnable in this session.

---

## Option 1 — Single Swift package (Interface target + Impl target, both `.dynamic` products)

Package: `.temp/linking-test/single-package/Packages/MyFeaturePackage/Package.swift`

Build:

```bash
cd .temp/linking-test/single-package/Packages/MyFeaturePackage
HOME=/tmp/spm-home swift build --disable-sandbox -c debug --product MyFeature
HOME=/tmp/spm-home swift build --disable-sandbox -c debug --product MyFeatureImpl
```

### Mach-O type (dynamic)

```bash
file .build/debug/libMyFeature.dylib .build/debug/libMyFeatureImpl.dylib
```

Observed:

```
.build/debug/libMyFeature.dylib:     Mach-O 64-bit dynamically linked shared library arm64
.build/debug/libMyFeatureImpl.dylib: Mach-O 64-bit dynamically linked shared library arm64
```

### Linkage: Impl does NOT link Interface dynamically

```bash
otool -L .build/debug/libMyFeatureImpl.dylib
```

Observed (no `libMyFeature.dylib` dependency):

```
.build/debug/libMyFeatureImpl.dylib:
	@rpath/libMyFeatureImpl.dylib ...
	/usr/lib/libSystem.B.dylib ...
	/usr/lib/swift/libswiftCore.dylib ...
```

### Proof of duplication: Interface symbol exists in BOTH dylibs

```bash
nm -gU .build/debug/libMyFeatureImpl.dylib | rg uniqueToken
nm -gU .build/debug/libMyFeature.dylib | rg uniqueToken
```

Observed (same `MyFeatureInterface.uniqueToken()` symbol in both):

```
0000000000000740 T _$s9MyFeature0aB9InterfaceO11uniqueTokenSSyFZ
```

Also, the interface string literal is present in `Impl`:

```bash
strings .build/debug/libMyFeatureImpl.dylib | rg MYFEATURE_INTERFACE_UNIQUE_260224
```

Observed:

```
MYFEATURE_INTERFACE_UNIQUE_260224
```

**Conclusion (Option 1):** Even when both products are `.dynamic`, `Impl` still **contains** `Interface` code/symbols (static inclusion). This violates the “Interface and Impl are separate dynamic modules” rule.

---

## Option 2 — Split into TWO Swift packages (Interface package + Impl package)

Packages:

- Interface: `.temp/linking-test/two-packages/MyFeature`
- Impl: `.temp/linking-test/two-packages/MyFeatureImpl` (depends on `../MyFeature`)
- App harness: `.temp/linking-test/two-packages/TwoPackageApp`

Build:

```bash
cd .temp/linking-test/two-packages/TwoPackageApp
HOME=/tmp/spm-home2 swift build --disable-sandbox -c debug
```

### Mach-O type (dynamic)

```bash
file .build/debug/libMyFeature.dylib .build/debug/libMyFeatureImpl.dylib
```

Observed:

```
.build/debug/libMyFeature.dylib:     Mach-O 64-bit dynamically linked shared library arm64
.build/debug/libMyFeatureImpl.dylib: Mach-O 64-bit dynamically linked shared library arm64
```

### Linkage: Impl DOES link Interface dynamically

```bash
otool -L .build/debug/libMyFeatureImpl.dylib
```

Observed (explicit dependency on `libMyFeature.dylib`):

```
.build/debug/libMyFeatureImpl.dylib:
	@rpath/libMyFeatureImpl.dylib ...
	@rpath/libMyFeature.dylib ...
	/usr/lib/libSystem.B.dylib ...
	/usr/lib/swift/libswiftCore.dylib ...
```

### Proof: Interface string literal is NOT inside Impl

```bash
strings .build/debug/libMyFeatureImpl.dylib | rg MYFEATURE_INTERFACE_UNIQUE_260224
```

Observed: no matches.

**Conclusion (Option 2):** Splitting into two packages produces a true dynamic boundary: `Impl` dynamically links `Interface` and does not duplicate its symbols/strings.

---

## Tuist: can it override this?

### What Tuist can do

Tuist exposes `PackageSettings` to control **how package products are represented in the generated Xcode project** (e.g., `productTypes`, destinations, and build settings). See:

- `PackageSettings` in `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/lib/ProjectDescription.framework/.../ProjectDescription.swiftinterface`
  - `productTypes: [String: Product]`
  - `targetSettings: [String: Settings]`
  - Defined around line ~`785` in `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/lib/ProjectDescription.framework/Versions/A/Modules/ProjectDescription.swiftmodule/arm64-apple-macos.swiftinterface`

### What Tuist can’t do (relevant to this problem)

Tuist does **not** (and cannot, without rewriting SwiftPM internals) change how SwiftPM composes a product from **intra-package targets**. If `MyFeatureImpl` target depends on `MyFeature` target inside the *same* package, SwiftPM will still include `MyFeature` in `MyFeatureImpl`’s binary.

In other words: Tuist can choose **framework vs static framework** for a *product*, but it cannot make `Impl` link `Interface` dynamically *when both are targets in the same package*.

Also, Tuist target dependencies on packages are expressed at the **product** level (`TargetDependency.package(product: ...)`), not at a “package target” level:

- `TargetDependency.package(product:type:condition:)` is defined around line ~`1546` in `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/lib/ProjectDescription.framework/Versions/A/Modules/ProjectDescription.swiftmodule/arm64-apple-macos.swiftinterface`.

### Reality check in this Codex sandbox

`tuist generate` / `tuist dump` fail here because Tuist compiles manifests via `xcrun swift` in a way that tries to write module cache files under `/Users/alexis/.cache/...` (not writable in this session). This is an environment limitation of the sandbox; it doesn’t change the SwiftPM behavior demonstrated above.

## Recommendation (single package vs two packages)

Given the requirement “Interface and Impl must be separate dynamic modules”:

- **Recommend: TWO packages per module.**
  - `ModuleName` package: `.dynamic` library product, Interface target only
  - `ModuleNameImpl` package: `.dynamic` library product, depends on `ModuleName` *package product*

This is the only approach tested here that results in `Impl` dynamically linking `Interface` (verified by `otool -L`) and avoids symbol duplication.

## Relevance to current bsim manifests

The reference bsim packages in `.research/ref_bsim-*-package.swift` use the single-package pattern:

- `BSimSDKImpl` depends on `.target(name: "BSimSDK")` (same package)
- Similar for `BSimIDImpl` → `BSimID`, `BSimTOTPImpl` → `BSimTOTP`

Based on the experiments above, even if those products are made `.dynamic`, the Impl dylib will still **embed** Interface code from the same package.
