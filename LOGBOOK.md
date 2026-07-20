# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-07-20

### 0620 — Executable Tests Profile And Generated-Source Hygiene
- ROOT CAUSE: Runtime-profile generation hardcoded the Tests action as `.targets([])` and discarded the mature app scheme that carried its testables and hosted-test launch argument. Supported reruns therefore produced a shared Tests scheme that `xcodebuild test` refused to execute.
- ROOT CAUSE: Capability insertion wrote indentation before a newly inserted newline, leaving a whitespace-only line, while SharedConfig emitted every Info.plist read on one line and exceeded the generated SwiftLint error threshold for mature type names.
- ROOT CAUSE: The app-groups convergence path could report an existing capability as up to date without normalizing legacy whitespace or the final collection comma, so supported mature-project reruns preserved generator-owned lint failures.
- DECISION: `runtime_profiles.test_action` is the durable, organization-agnostic ownership boundary. It requires at least one explicit test target and optionally carries static non-secret launch flags used only by the Tests action; missing or empty target metadata fails closed.
- FIX: Generate typed testables/arguments into the Tests scheme, normalize both insertion and converged capability paths at the closing line boundary, and wrap SharedConfig read arguments deterministically.
- VALIDATION: Mature Converter unit and fixture UI bundles execute through the generated Tests scheme on their canonical iOS 26.5 simulators; an iOS 26.3 snapshot mismatch was runtime drift, not accepted as generator evidence.
- SCOPE: Runtime schema/config/golden output, Converter-shaped mature-project fixtures, app capability and SharedConfig regressions, public docs, and supported generation convergence.

### 0457 — Non-Destructive Mature Runtime Adoption
- ROOT CAUSE: Runtime-profile migration treated the `[Configuration]` type annotation as the legacy initializer, appended a second `PackageSettings.configurations` argument, inserted SharedConfig after an unterminated dependency item, and left Converter's Debug/Release app scheme active after replacing those configurations.
- ROOT CAUSE: SecureStore setup delegated to full IoC Registry regeneration, which would erase mature custom registrations and builders before AppConfig adoption.
- FIX: Added syntax-aware configuration/argument/scheme migration, comma-safe dependency insertion, and focused SecureStore/AppConfig Registry patches that preserve unrelated composition.
- SCOPE: Converter-shaped fixtures cover custom Registry preservation, typed Tuist manifests, legacy app-scheme replacement, second-run convergence, and validation-output secrecy.
- REVIEW: Independent review found that a preserved final custom scheme also needed comma termination before managed schemes; the shared comma-safe insertion helper and regression now cover it.
- STATUS: Full Go tests/vet/build, second-run byte convergence, real Converter-shaped Tuist generation, and the PilotTestFlight simulator build pass; final independent source re-review follows.

### 0335 — API Origin Default-Port Isolation
- ROOT CAUSE: Backend origin ownership lowercased the full host string but did not normalize an omitted port, an explicit scheme-default port, or zero-padded numeric port spelling. Equivalent HTTPS origins such as `https://api.example.com` and `https://api.example.com:443` could therefore evade the environment-isolation collision check.
- FIX: Canonical ownership keys now compare lowercase scheme/hostname plus normalized numeric ports, omit the effective HTTPS `443` and HTTP `80` defaults, and preserve IPv6 host brackets. Generated Swift continues to serialize each configured exact origin without rewriting it.
- STATUS: Regression coverage includes omitted, explicit-default, zero-padded default/non-default, HTTP, HTTPS, and IPv6 forms; full generator validation follows before review handoff.

### 0310 — Explicit Shared Firebase Public Identity
- DECISION: Duplicate Firebase project IDs, Google App IDs, resource names, and validation hooks remain rejected unless every non-fixture participant declares the same `identity_sharing_group` and the complete five-field public registration tuple matches exactly.
- DECISION: The generated Swift group is a Firebase public-client trust boundary only. Firebase tokens carry no backend-environment claim; API origins plus auth, storage, grant, and quota namespaces remain unique and environment-owned.
- DECISION: The runtime-profile schema remains version 1 because `identity_sharing_group` is optional and existing configs retain their prior fail-closed behavior.
- SCOPE: Runtime-profile config/schema validation, Swift descriptors, generic fixture, documentation, and Go/golden/integration coverage.

### 0105 — Typed Runtime Profiles And Package Configuration Convergence
- DECISION: Distribution profiles and backend environments are independent fixed enums; generated policy validates the approved profile matrix and never synthesizes an API path.
- DECISION: Firebase config persists public registration metadata plus an environment-variable hook name only; plist paths and API keys remain process-local validation inputs.
- ROOT CAUSE: Custom app configurations must also be applied to Tuist-generated package projects, and the Swift profile case `internal` must be escaped. Forced scaffold reruns also exposed duplicate local/external SharedConfig dependencies; real Tuist validation caught generated `PackageSettings` arguments outside initializer order.
- ROOT CAUSE: The SecureStore builder still referenced the removed Info.plist-shaped app-group accessor (`GROUP_*`), while current shared configuration generates canonical lower-camel properties such as `main`; the real Swift build exposed the mismatch.
- FIX: Added typed runtime schema/subplugins, package configuration sync, escaped Swift enum output, canonical SharedConfig dependency convergence, one-run migration to `productTypes` → `baseSettings` → `targetSettings` ordering, and canonical SecureStore app-group property derivation.
- SCOPE: `tuist-starter/internal/config`, `tuist-starter/internal/scaffold`, `tuist-starter/internal/appconfig`, runtime-profile docs/examples/tests.
- STATUS: Unit/golden tests, JSON Schema validation, Tuist generation, Swift 6 typecheck, and simulator builds for all four generated profile schemes passed in a generic fixture.

## 2026-07-02

### 1252 — Remote Notification Background Mode
- ROOT CAUSE: Apps implementing `application:didReceiveRemoteNotification:fetchCompletionHandler:` need `remote-notification` in `UIBackgroundModes`; the scaffold contract only accepted `audio` and `voip`.
- FIX: Extend `background_modes` validation/generation/docs to include `remote-notification`.
- SCOPE: `tuist-starter/internal/config`, `tuist-starter/internal/scaffold/background_modes_config_test.go`, `README.md`, `SKILL.md`, `references/cli-reference.md`.
- STATUS: `go test ./...`, `./scripts/setup.sh`, downstream `generate background-modes-config`, and VideoCallDemo focused push harness test passed.

## 2026-07-01

### 1836 — Background Modes Project Config
- DECISION: `UIBackgroundModes` is owned by a dedicated `generate background-modes-config` leaf and wired into `generate project-config`, not merged into presentation/privacy sync.
- FIX: Added `background_modes` config enum validation for `audio` and explicit `voip`; unknown values are rejected.
- FIX: Host app `Project.swift` converges `UIBackgroundModes` idempotently; omitted or empty config removes the scaffold-owned key.
- SCOPE: `tuist-starter/internal/config`, `tuist-starter/internal/scaffold/background_modes_config.go`, `tuist-starter/internal/cli/generate_test.go`, docs.
- STATUS: `go test ./...`, `go vet ./...`, and `./scripts/setup.sh` passed.

## 2026-06-30

### 2106 — Extension Metadata Sync Generators
- DECISION: Cross-cutting extension metadata belongs to project-config leaf generators, not concrete extension plugins.
- FIX: Added `generate bundle-id` and wired it into `generate project-config`; existing `versions`, `min-target`, `team-id`, and `build-flags` discover `Extensions/**/Project.swift`.
- FIX: Added regression coverage for configured Swift concurrency restrictions propagating to extension manifests.
- SCOPE: `tuist-starter/internal/scaffold/bundle_id.go`, `generator_bundle_id.go`, `build_flags_test.go`, `generate_test.go`, docs.
- STATUS: `go test ./...` passed from `tuist-starter`.

## 2026-05-19

### 1606 — Setup Xcode Sudo Bootstrap
- DECISION: Default setup remains non-privileged; full Xcode selection is opt-in via `./scripts/setup.sh --select-xcode`.
- DECISION: Sudo path uses a hardcoded printed command list only: `sudo -v` and `sudo xcode-select -s /Applications/Xcode.app/Contents/Developer`; no eval or user-provided shell string execution.
- SCOPE: `scripts/setup.sh`, `README.md`, `SKILL.md`.
- STATUS: Setup can install on CommandLineTools-only machines while reporting full-Xcode readiness warnings.

### 1558 — Runtime Error And Startup Diagnostics
- DECISION: Runtime error MVP parses unified-log NDJSON/plain lines plus app-emitted `IAM_ERROR`; collection uses `log show --style ndjson` with `logType == error/fault`.
- DECISION: Startup latency uses explicit `app_start` and `first_render` markers from `PerformanceProbe`, not inference from arbitrary samples.
- SCOPE: `tuist-starter/internal/profile/runtime_errors.go`, `runtime_probe_template.go`, `profile runtime errors`, runtime docs.
- STATUS: Analyzer groups errors by severity/scope/signature and reports app-start-to-first-render duration.

### 1543 — Rendered Layout Hierarchy Diagnostics
- DECISION: Layout diagnostics use public XCTest/accessibility hierarchy XML, not `debugDescription` text or private UIKit/SwiftUI tree scraping.
- DECISION: Analyzer accepts generated `LayoutHierarchyProbe` XML, Appium/WebDriverAgent XML, and logs with `IAM_LAYOUT_XML_*` markers.
- SCOPE: `tuist-starter/internal/profile/layout.go`, `layout_probe_template.go`, `profile layout scaffold/analyze`, layout docs.
- STATUS: Reports agent-readable tree, duplicate identities, missing interactive identity, tiny tap targets, and offscreen frames.

### 1535 — Local Profile Diagnostics Architecture
- DECISION: Build profiling uses `xcodebuild -showBuildTimingSummary` plus `tuist graph --format legacyJSON`; `.xcactivitylog` parsing deferred.
- DECISION: Runtime profiling uses generated debug-only `PerformanceProbe.swift` with signposts plus `IAM_PROFILE` JSON lines; no private SwiftUI API dependency.
- SCOPE: `tuist-starter/internal/profile`, `tuist-starter/internal/cli/profile.go`, `references/*profiling*`, `diagrams/profile-diagnostics-architecture.puml`.
- STATUS: MVP covers build critical-path estimates and runtime repeated/slow call reports.

## 2026-05-18

### 2220 — Project Config Strictness And Team ID Regression
- ROOT CAUSE: `generate project-config` had no `team-id` leaf; `team_id` changes only reached `ApplicationConfiguration.developmentTeamID`, leaving `developmentTeam` constants and `DEVELOPMENT_TEAM` stale or absent in extensions.
- ROOT CAUSE: Xcode can show strict concurrency via `SWIFT_STRICT_CONCURRENCY_DEFAULT`; setting only `SWIFT_STRICT_CONCURRENCY` is not enough for explicit Build Settings display.
- FIX: Added `generate team-id`, `SWIFT_STRICT_CONCURRENCY_DEFAULT`, `SWIFT_STRICT_MEMORY_SAFETY`, and explicit module `StrictConcurrency` SwiftPM settings.
- SCOPE: `tuist-starter/internal/scaffold/team_id.go`, `tuist-starter/internal/config/swift_settings.go`, extension project template, generator docs/tests.
