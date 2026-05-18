# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-05-18

### 2220 — Project Config Strictness And Team ID Regression
- ROOT CAUSE: `generate project-config` had no `team-id` leaf; `team_id` changes only reached `ApplicationConfiguration.developmentTeamID`, leaving `developmentTeam` constants and `DEVELOPMENT_TEAM` stale or absent in extensions.
- ROOT CAUSE: Xcode can show strict concurrency via `SWIFT_STRICT_CONCURRENCY_DEFAULT`; setting only `SWIFT_STRICT_CONCURRENCY` is not enough for explicit Build Settings display.
- FIX: Added `generate team-id`, `SWIFT_STRICT_CONCURRENCY_DEFAULT`, `SWIFT_STRICT_MEMORY_SAFETY`, and explicit module `StrictConcurrency` SwiftPM settings.
- SCOPE: `tuist-starter/internal/scaffold/team_id.go`, `tuist-starter/internal/config/swift_settings.go`, extension project template, generator docs/tests.
