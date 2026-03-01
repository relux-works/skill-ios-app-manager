# TASK-260302-2e5o6x: dep-checking-in-setup-command

## Description
Add dependency checking to NewSetupCommand in setup_command.go.

Before the Plan phase, if module has Dependencies:
1. Resolve Registry.swift path: {projectRoot}/Targets/{appName}/Sources/App/{appName}.Registry.swift
2. Read file content (if file missing and module has deps → error: run ioc setup first)
3. Call registry.CheckDependencies(mod.ID, content)
4. If unmet deps → return error listing what is missing

Skip check entirely if module.Dependencies is empty.

Files to modify:
- internal/cli/setup_command.go — add dep check between input assembly and Plan call
- internal/cli/setup_command_test.go — add tests:
  - Module with deps + no Registry.swift → error mentioning ioc setup
  - Module with deps + Registry has dep → success
  - Module with deps + Registry missing dep → error listing missing
  - Module with no deps + no Registry.swift → success (no check)

Use existing test helpers: fakeModule(), executeSetupCommand(), testProjectConfig(), writeModuleConfig().
For dep tests, create a fakeModuleWithDeps() that depends on a registered FakeDep module.
Write Registry.swift file in temp dir at the expected path to simulate presence.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
