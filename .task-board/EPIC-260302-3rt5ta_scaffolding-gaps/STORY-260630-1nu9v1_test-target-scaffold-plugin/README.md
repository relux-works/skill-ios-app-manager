# Test target scaffold plugin

## Description
Implement test target scaffolding as a separate scaffold plugin with subplugins. The top-level command owns orchestration and idempotency. Unit test and UI test target generation are separate subplugins so projects can create either one independently or both together. Target names are explicit command parameters; the scaffold must not assume fixed names such as UnitTests or UITests unless a caller chooses them.

## Scope
Add CLI/API shape, config model if needed, Tuist Project.swift target generation, source directory scaffolding, package/dependency wiring, generated Makefile/test behavior if affected, docs, and tests. Unit test targets should default to Swift Testing-compatible Swift files. UI test targets should generate Swift XCUITest-compatible scaffolding and leave room for Page Object/accessibility-id patterns from ios-testing-tools.

## Acceptance Criteria
A caller can add a unit test target with an explicit target name; a caller can add a UI test target with an explicit target name; running the same command twice does not duplicate targets, files, dependencies, schemes, or package settings; changing supported config converges existing generated output; generated unit tests compile as Swift test sources; generated UI tests compile as Swift XCUITest sources and can launch the host app; project-config/build/test commands continue to work; README and SKILL.md document the test target plugin and subplugin contract.
