## Status
done

## Assigned To
codex

## Created
2026-05-07T10:05:42Z

## Last Update
2026-05-07T10:18:51Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Define platform/min-target tuple model
- [x] Wire module create defaults from ios-app-manager config
- [x] Expose CLI override for module platforms
- [x] Update package generation tests and build CLI

## Notes
Implemented module platform/min-target tuples via repeatable --platform <platform>:<min_target>. Default module platforms now derive from ios-app-manager.json min_target as iOS:<min_target>. Verification: go test ./... and make build passed.
Strengthened platform guard rails: platform is now typed components.Platform enum; module creation requires resolved platform list length >= 1; empty tuple, empty platform/min target, unknown platform, and non-major.minor versions fail with explicit messages before generated package files are kept. Verification: go test ./... and make build passed.

## Precondition Resources
(none)

## Outcome Resources
(none)
