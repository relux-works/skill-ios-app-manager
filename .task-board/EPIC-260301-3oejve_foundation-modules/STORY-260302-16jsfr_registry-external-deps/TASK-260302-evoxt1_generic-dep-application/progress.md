## Status
done

## Assigned To
[implementer] developer (codex)

## Created
2026-03-01T22:08:19Z

## Last Update
2026-03-02T08:15:32Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [ ] Code written per task description and AC
- [ ] Tests written and passing
- [ ] Lint clean
- [ ] Build not broken
- [ ] Result noted on the board
- [ ] setup_command.go has generic ExternalDeps application loop before Plan()
- [ ] ioc/setup.go has no external dep constants or functions (swiftIoCURL etc removed)
- [ ] relux/setup.go has no external dep constants or functions (swiftUIReluxURL etc removed, BUT patchPackageSwiftForRelux kept)
- [ ] httpclient/setup.go has no external dep constants or functions
- [ ] make test passes (all packages green)
- [ ] make lint passes
- [ ] Demo app full pipeline works
- [ ] xcodebuild BUILD SUCCEEDED

## Notes
agent spawned: codex (pid=76182, exit=0)
Implemented generic ExternalDeps application in internal/cli/setup_command.go before dependency checks and Plan(). Removed module-specific external dep constants/functions from internal/ioc/setup.go, internal/relux/setup.go, internal/httpclient/setup.go (kept patchPackageSwiftForRelux). Added tests in internal/cli/setup_command_test.go for external dep application + idempotency. Verification: make test PASS, make build PASS, make lint PASS. Demo command pipeline PASS in .temp/demo-project-codex-1772404442. tuist/xcodebuild step is blocked in this sandbox: tuist crashes with mkdir operation not permitted.

## Precondition Resources
(none)

## Outcome Resources
- [results.md](file://TASK-260302-evoxt1/results.md) — Implementation and verification summary
