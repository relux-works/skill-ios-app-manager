# STORY-260224-24gvhw: e2e-validation-xflow

## Description
End-to-end validation: take xflow iOS app (/Users/alexis/src/xflow/connect-ios/) as reference, compose a project config based on its existing setup (bundle ID, team ID, entitlements, etc), feed it to our CLI tool, and get a fully structured Tuist project.

Expected output:
- Working Tuist project that generates and builds via make setup + make build
- Project structure matches our conventions (interface/impl module split, Relux architecture)
- One simple Relux feature module: TodoList (no persistence, no API — pure in-memory state)
  - TodoList.Interface module: protocols, DTOs (TodoItem, TodoAction)
  - TodoList.Implementation module: Store, Reducer, State, Actions, SwiftUI view
- Config derived from xflow app: bundle ID, team, deployment target, entitlements
- NO actual code porting from xflow — just the project skeleton + one demo module

This is the smoke test / proof of concept that the whole pipeline works end to end.

Reference: /Users/alexis/src/xflow/connect-ios/

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
