# TASK-260228-2mqb8g: update-flow-snapshot-tests

## Description
Update golden file snapshots and tests for flow template changes. Same pattern as namespace story — regenerate goldens, verify all tests pass.

Steps:
1. Run make test to identify failures
2. Run make test-update to regenerate golden files
3. Run make test to confirm all pass
4. Review golden files for correctness

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
