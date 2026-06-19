# TASK-260619-35ob2g: pre-generate-script-config

## Description
Add ios-app-manager project config support for typed pre-generate lifecycle scripts that run after Tuist dependency installation and before Tuist project generation.

## Scope
- Parse scripts.pre_generate from ios-app-manager.json.
- Validate script path and execution language.
- Generate a Makefile target that runs configured scripts after tuist install and before tuist generate.
- Document the config contract and cover it with tests.

## Acceptance Criteria
- Config loader parses pre_generate scripts.
- Validator rejects invalid script paths and languages.
- Generated Makefiles expose run-pre-generate-scripts.
- setup/generate/build/test run the hook at the correct lifecycle point.
- go test ./... passes in tuist-starter.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
