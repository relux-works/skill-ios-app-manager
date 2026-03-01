# Module Audit: httpclient

## Package
`internal/httpclient/` — CLI command: `http-client setup`

## Category
**network**

---

## Setup() Flow (step by step)

1. **Validate input** — requires non-empty `ProjectRoot` and `AppName`
2. **Resolve modules path** — defaults to `"Packages"` if `ModulesPath` is empty
3. **Add swift-httpclient external dependency** to root `Package.swift`
   - URL: `https://github.com/relux-works/swift-httpclient.git`
   - Version: `from: 6.0.0`
   - Package name: `HttpClient`
   - Uses `deps.AddExternalDep()` — idempotent (skips if already present)
4. **Add HttpClient to Project.swift** dependencies
   - Content: `.external(name: "HttpClient")`
   - Uses `tuistproj.ApplyManifestEditsToFile()` — idempotent
5. **Create `Configuration+HttpClient.swift`** file
   - From embedded template (no Go template rendering — raw file copy)
   - Defines `Configuration.HttpClient` enum with timeout constants
   - Idempotent: skips if file already exists
6. **Patch Registry.swift**
   - Adds `import HttpClient` after `import SwiftIoC`
   - Inserts IoC registration line at `// MARK: - Network` anchor
   - Inserts `buildHttpClient()` builder method at `// MARK: - Network Builders` anchor
   - Idempotent: skips if `IRpcAsyncClient.self` already found in file

---

## Files Created

| File | Path (relative to project root) |
|------|------|
| Configuration+HttpClient.swift | `Targets/{AppName}/Sources/Configuration/Configuration+HttpClient.swift` |

Template content (static, no Go template variables):
```swift
extension Configuration {
    enum HttpClient {
        static let timeoutForResponse: TimeInterval = 10
        static let timeoutResourceInterval: TimeInterval = 120
    }
}
```

---

## Files Patched

| File | Patch |
|------|-------|
| `{ModulesPath}/Package.swift` | Adds `swift-httpclient` external dependency via `deps.AddExternalDep()` |
| `Project.swift` (project root) | Adds `.external(name: "HttpClient")` to dependencies via `tuistproj.ApplyManifestEditsToFile()` |
| `Targets/{AppName}/Sources/App/{AppName}.Registry.swift` | Adds `import HttpClient`, IoC registration in network section, builder method in network-builders section |

### Registry.swift patch details

**Import added:**
```swift
import HttpClient
```

**Registration line** (inserted after network anchor):
```swift
            ioc.register(IRpcAsyncClient.self, lifecycle: .container, resolver: Self.buildHttpClient)
```

**Builder method** (inserted into Network Builders extension):
```swift
    private static func buildHttpClient() -> IRpcAsyncClient {
        RpcClient(
            sessionConfig: ApiSessionConfigBuilder.buildConfig(
                timeoutForResponse: Configuration.HttpClient.timeoutForResponse,
                timeoutResourceInterval: Configuration.HttpClient.timeoutResourceInterval
            )
        )
    }
```

---

## SetupInput Fields

```go
type SetupInput struct {
    ProjectRoot string   // required — project directory
    AppName     string   // required — from config.AppName
    ModulesPath string   // optional — from config.ModulesPath, defaults to "Packages"
}
```

All three fields come from the config file (`ios-app-manager.json`). No CLI flags beyond `--config`.

---

## Prerequisites

1. **Config file** must exist with `app_name` set
2. **IoC setup** must have been run first — Registry.swift must exist with:
   - `import SwiftIoC`
   - `// MARK: - Network (scaffolding anchor: network)` anchor
   - `// MARK: - Network Builders (scaffolding anchor: network-builders)` anchor
   - Extension block with braces after network-builders anchor
3. **Project.swift** must exist at project root (for dependency patching)
4. **Package.swift** must exist at modules root (for external dep)

Without IoC setup, the command fails with "read Registry.swift" error (tested in `TestHttpClientSetupWithoutIoC`).

---

## Special Flags/Params

**None.** No extra CLI flags. Everything comes from config.

---

## Registry Fit Assessment

### Fits cleanly: YES

**Module struct mapping:**
```go
Module{
    ID:           "http-client",
    Name:         "HttpClient",
    Description:  "HTTP client IoC registration with swift-httpclient",
    Dependencies: []ModuleID{"ioc"},
    Category:     "network",
    Setup:        httpclient.Setup, // signature already matches func(SetupInput) error
    Plan:         /* straightforward */,
    UsageGuide:   "HttpClient setup complete",
}
```

**Why it fits well:**
- `SetupInput` already matches the common `SetupInput` shape (ProjectRoot, AppName, ModulesPath) — trivially unifiable
- No special flags needed — no access-group, no extra params
- Single prerequisite: `ioc` — maps directly to `Dependencies: []ModuleID{"ioc"}`
- All operations are idempotent — safe for re-runs
- Registry anchors used: `network` and `network-builders` — consistent with the anchor system
- Category is clearly `network` (uses MARK: Network anchors)

**Minor notes:**
- The module uses `deps.AddExternalDep` and `tuistproj.ApplyManifestEditsToFile` which are shared infrastructure — no custom patching hacks
- Builder method uses `findMatchingBrace()` for brace-matching insertion — slightly more sophisticated than simple string replace, but contained within the package
- Template is a single static file (no Go template variables) — could be simplified to inline content if desired

### No adaptation needed. Cleanest fit of foundation modules for the registry pattern.

---

## Key Takeaways

1. **Simplest module audited** — minimal SetupInput, no special flags, single prerequisite
2. **Network category** — uses network/network-builders anchors in Registry.swift
3. **Pure transport layer** — registers `IRpcAsyncClient` with `RpcClient` implementation, configured via `Configuration.HttpClient` constants
4. **Excellent idempotency** — every step checks before acting (file exists, string contains, dep already present)
5. **Clean registry fit** — maps 1:1 to the Module struct with zero adaptation needed
6. **External dep management** — adds `swift-httpclient` package (from: 6.0.0) to both Package.swift and Project.swift
