## Status
done

## Assigned To
(none)

## Created
2026-03-22T11:53:34Z

## Last Update
2026-03-22T12:00:24Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Create temp Tuist project with local package
- [x] Inspect generated package target build settings
- [x] Verify strictness behavior before and after Tuist PackageSettings overrides

## Notes
Temp repro completed in /tmp/tuist-package-probe.6ZiUo3/project.
Baseline without root PackageSettings: Tuist generated Packages/UnsafePkg/UnsafePkg.xcodeproj with target UnsafePkg using SWIFT_VERSION = 5.0 and no explicit strict concurrency build settings. xcodebuild showed swiftc invoked with -swift-version 5. The package source used a non-Sendable Box captured inside an @Sendable closure, and the generated package target built successfully.
After adding #if TUIST PackageSettings(baseSettings: ...) to the root Package.swift and regenerating with tuist generate, the same generated target switched to SWIFT_VERSION = 6.0 and got the strict settings block (SWIFT_APPROACHABLE_CONCURRENCY = NO, SWIFT_DEFAULT_ACTOR_ISOLATION = nonisolated, SWIFT_STRICT_CONCURRENCY = complete, pinned SWIFT_UPCOMING_FEATURE_*). Rebuilding the exact same source failed at compile time with: capture of box with non-Sendable type Box in a @Sendable closure.
Conclusion: this strictness gap is real on Tuist-generated package targets and PackageSettings on the root Package.swift is sufficient to close it in generated Xcode targets. Next scaffold work should wire a config-driven source of truth into root Package.swift generation and migration sync for existing projects.

## Precondition Resources
(none)

## Outcome Resources
(none)
