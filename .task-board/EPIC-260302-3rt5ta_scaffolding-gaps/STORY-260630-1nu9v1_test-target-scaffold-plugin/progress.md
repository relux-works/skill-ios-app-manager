## Status
done

## Assigned To
codex-inline

## Created
2026-06-30T16:47:48Z

## Last Update
2026-06-30T17:19:43Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Architecture principle from user: keep scaffold pluginized. The test-targets plugin is only an orchestrator; unit-test-target and ui-test-target are subplugins with separate responsibilities. Do not merge unrelated test/extension behavior into one generator.
Implemented initial pluginized test-targets setup command with unit/ui subplugin structs, explicit --unit-target/--ui-target parameters, idempotent Project.swift target insertion, and starter Swift Testing/XCUITest source generation. Validation so far: go test ./internal/testtargets ./internal/registry ./internal/cli and go test ./... pass; CLI help lists test-targets.
Implemented pluginized test-targets setup with unit and UI subplugins, explicit target names, idempotent Project.swift edits, Swift Testing and XCUITest starter files. Verification go test ./... passed.

## Precondition Resources
(none)

## Outcome Resources
(none)
