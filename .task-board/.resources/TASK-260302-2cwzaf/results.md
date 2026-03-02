# TASK-260302-2cwzaf results

## Code changes
- internal/cli/diagram.go
  - Added --format flag with supported values: puml, png, svg (default: puml).
  - Added format validation with clear unsupported-format error.
  - Made default output path format-aware: diagrams/scaffolding-pipeline.<format> when --output is not explicitly set.
  - Kept direct write behavior for puml.
  - For png/svg: writes sibling .puml source, then executes plantuml -t<format> <puml-path> via exec.Command.
  - Added clear missing-binary error: plantuml not found in PATH (install: brew install plantuml).
- internal/cli/diagram_test.go
  - Updated help test to include --format.
  - Added invalid format validation test.
  - Added default output path behavior test for puml/png/svg.
  - Added png/svg rendering tests using a fake plantuml binary in PATH.
  - Added missing-plantuml error test.

## Verification
- env GOCACHE=/tmp/go-build go test ./internal/cli -run Diagram
- env GOCACHE=/tmp/go-build go test ./...
- env GOCACHE=/tmp/go-build go vet ./...
- env GOCACHE=/tmp/go-build go build ./...
