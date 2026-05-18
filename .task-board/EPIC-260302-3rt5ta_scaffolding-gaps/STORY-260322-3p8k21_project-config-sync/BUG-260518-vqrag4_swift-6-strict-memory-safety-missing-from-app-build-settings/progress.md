## Status
closed

## Assigned To
codex

## Created
2026-05-18T19:07:39Z

## Last Update
2026-05-18T19:22:52Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Reproduced/triaged in .temp/repro-user-bugs/current and local Xcode 26.4 Swift.xcspec. App/extension Tuist manifest has SWIFT_STRICT_CONCURRENCY = complete; Xcode spec says Swift 6 strict concurrency is always complete. The provided screenshots show Strict Memory Safety = NO; current strict baseline does not emit SWIFT_STRICT_MEMORY_SAFETY.
Fixed by emitting SWIFT_STRICT_CONCURRENCY_DEFAULT = complete, SWIFT_STRICT_CONCURRENCY = complete, and SWIFT_STRICT_MEMORY_SAFETY = YES for app/extension Project.swift build settings; module Package.swift now emits .enableUpcomingFeature("StrictConcurrency"). Verified with make test, make lint, tuist generate, and xcodebuild -showBuildSettings.
Final review: diff check clean; make test and make lint passed. Scratch Tuist/Xcode verification shows host, extension, and module strictness settings are generated as expected.

## Precondition Resources
(none)

## Outcome Resources
(none)
