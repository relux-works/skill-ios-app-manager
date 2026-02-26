# EPIC-260226-1gj2fg: agents-scaffolding

## Description
When scaffolding a new project (ios-app-manager init), generate an agents/ directory with instructions and skills, then symlink into .claude/ and .codex/ for both Claude Code and Codex CLI support. Follows the canonical agents/skills/ pattern with symlinks to .claude/skills/ and .codex/skills/. Also sets up agents/.instructions/ with project-level instruction files.

Skills to include:
- swiftui — SwiftUI best practices, view composition, state management
- swift-testing-tools — Swift Testing framework patterns and helpers (also add as Swift package dependency to the generated project)
- ios-app-manager — CLI reference, DSL syntax, workflows for managing the Tuist project

More skills may be added later.

Also scaffold 4 standard test targets in the generated project:
- UnitTests
- IntegrationTests
- SnapshotTests
- UITests

These targets must always be present in every scaffolded project.

## Scope
(define epic scope)

## Acceptance Criteria
(define acceptance criteria)
