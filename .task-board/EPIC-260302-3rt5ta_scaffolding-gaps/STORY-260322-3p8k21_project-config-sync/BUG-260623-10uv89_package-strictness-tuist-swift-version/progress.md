## Status
to-review

## Assigned To
codex

## Created
2026-06-23T12:06:27Z

## Last Update
2026-06-23T12:30:50Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Reproduced from Tap2Cash: ios-app-manager config already carries Swift 6/concurrency strictness, but generate package-strictness does not sync root Tuist PackageSettings.targetSettings for project-owned package targets. Tuist-generated Xcode package targets then compile with -swift-version 5 despite SwiftPM manifests using .swiftLanguageMode(.v6).
Implemented generator fix: package-strictness now discovers local Swift 6 package manifests recursively from root Package.swift and ModulesPath, then syncs root Tuist PackageSettings.targetSettings with EffectiveSwiftSettings Xcode build settings. Verified go test ./... and installed via ./scripts/setup.sh.
Completed verification from Tap2Cash: installed ios-app-manager now generates root Tuist PackageSettings.targetSettings for recursively discovered local Swift 6 package targets using EffectiveSwiftSettings. Re-ran ios-app-manager generate package-strictness in Tap2Cash; app/Package.swift now carries generated swiftPackageTargetSettings and no manual temp override. Full Go suite passed and Tap2Cash iPhone1 build/install passed with local targets on -swift-version 6.

## Precondition Resources
(none)

## Outcome Resources
(none)
