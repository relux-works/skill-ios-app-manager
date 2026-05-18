## Status
reviewing

## Assigned To
codex

## Created
2026-03-22T10:16:10Z

## Last Update
2026-05-18T19:22:52Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Add min_target and deployment target sync for app plus extensions
- [x] Choose command shape and backward-compatibility strategy for generate versions
- [x] Design reusable scalar manifest sync primitives for Project.swift and extension manifests
- [x] Cover idempotency, migration, and docs for the new config sync workflow

## Notes
Completed the first project-config sync module family. The generate tree now has leaf plugins for versions and min-target plus an orchestration entrypoint generate project-config. Root and extension manifests share explicit minTarget markers, older manifests migrate forward during sync, and the public docs were updated to recommend generate project-config as the main workflow. Verified with go test ./... from tuist-starter.

## Precondition Resources
(none)

## Outcome Resources
(none)
