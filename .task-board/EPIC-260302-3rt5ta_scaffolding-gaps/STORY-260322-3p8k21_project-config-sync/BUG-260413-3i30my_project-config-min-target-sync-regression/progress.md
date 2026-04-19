## Status
backlog

## Assigned To
(none)

## Created
2026-04-13T08:07:44Z

## Last Update
2026-04-13T08:07:59Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Repro:
1. In generated project x-platform-airdrop/ios, change ios-app-manager.json min_target from 26.0 to 18.0.
2. Run ios-app-manager generate project-config from the project root.
3. Observe output: min-target reports already up to date.
4. Inspect Project.swift: minTarget stays 26.0.
5. Inspect root Package.swift: existing #if TUIST PackageSettings strictness block was removed during package-strictness substep.
6. Package manifests under Packages/* also keep stale platform values until patched manually.

Expected:
- generate project-config or generate min-target rewrites root Project.swift minTarget and IPHONEOS_DEPLOYMENT_TARGET from config source of truth.
- package-strictness must not delete an existing root PackageSettings block when config did not ask for that change.
- If module package deployment target sync is intended, it should update Packages/*/Package.swift too; otherwise docs should explicitly say it is out of scope.

## Precondition Resources
(none)

## Outcome Resources
(none)
