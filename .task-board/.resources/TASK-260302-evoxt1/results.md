# TASK-260302-evoxt1 results

## Code changes
- Added generic ExternalDeps application loop to internal/cli/setup_command.go before dependency checks and Plan().
- Removed hardcoded external dependency constants/functions/calls from internal/ioc/setup.go.
- Removed hardcoded external dependency constants/functions/calls from internal/relux/setup.go (kept patchPackageSwiftForRelux).
- Removed hardcoded external dependency constants/functions/calls from internal/httpclient/setup.go.
- Added tests in internal/cli/setup_command_test.go:
  - TestSetupCommandAppliesExternalDepsBeforePlan
  - TestSetupCommandExternalDepsIdempotent

## Verification
- make test: PASS
- make build: PASS
- make lint: PASS
- Demo setup pipeline command chain: PASS in .temp/demo-project-codex-1772404442
- tuist install/generate + xcodebuild: BLOCKED by sandbox (tuist fails at startup with mkdir operation not permitted).