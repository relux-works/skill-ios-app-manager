## Status
done

## Assigned To
[implementer] developer (codex)

## Created
2026-03-02T09:36:37Z

## Last Update
2026-03-02T09:43:34Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Code written per task description and AC
- [x] Tests written and passing
- [x] Lint clean
- [x] Build not broken
- [x] Result noted on the board
- [x] --format flag accepts puml, png, svg
- [x] Default format is puml (backward compatible)
- [x] Default output path adapts to format: diagrams/scaffolding-pipeline.<format>
- [x] png/svg render via plantuml CLI (exec.Command)
- [x] Clear error if plantuml not in PATH
- [x] Invalid format rejected with error message
- [x] Existing tests still pass
- [x] New tests for format validation

## Notes
agent spawned: codex (pid=74439, exit=0)
Implemented in internal/cli/diagram.go and internal/cli/diagram_test.go. Added --format (puml|png|svg), format-aware default output paths, plantuml rendering for png/svg via exec.Command, missing-plantuml PATH error, and format validation. Verification: env GOCACHE=/tmp/go-build go test ./..., go vet ./..., go build ./...

## Precondition Resources
(none)

## Outcome Resources
- [results.md](file://TASK-260302-2cwzaf/results.md) — Implementation and verification summary
