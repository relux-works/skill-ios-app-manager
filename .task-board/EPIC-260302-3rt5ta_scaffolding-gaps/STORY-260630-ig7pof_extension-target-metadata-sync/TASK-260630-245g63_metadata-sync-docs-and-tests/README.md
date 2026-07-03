# TASK-260630-245g63: metadata-sync-docs-and-tests

## Description
Add regression tests and docs covering dependency direction: extension plugins register targets, project-config plugins sync cross-cutting metadata.

## Scope
Add regression coverage and documentation for extension metadata sync, including concurrency restrictions under build flags and project-config orchestration.

## Acceptance Criteria
Go tests cover versions, bundle id, min target, team id, build flags/concurrency on extension manifests; README/SKILL/CLI reference explain extension plugins create thin targets while project-config sync propagates cross-cutting metadata; go test ./... passes.
