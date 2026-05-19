# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

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
