TASK-260224-24aa6b results

Implemented
- Added clean manager with quick/deep modes and kill-xcode helper: internal/clean/manager.go
- Added clean manager tests for quick/deep path coverage and kill delegation: internal/clean/manager_test.go
- Wired CLI clean command with --deep and --kill-xcode flags: internal/cli/clean.go
- Added CLI tests for quick/deep/kill behavior and warning path: internal/cli/clean_test.go
- Updated root stub expectations now that clean is implemented: internal/cli/root_test.go

Verification
- GOCACHE=$(pwd)/.tmp/go-build go test ./internal/clean/...
- GOCACHE=$(pwd)/.tmp/go-build go test ./internal/clean -run TestQuickClean|TestDeepClean|TestKillXcode -count=1
- GOCACHE=$(pwd)/.tmp/go-build go test ./internal/cli -run TestCleanCommand|TestStubCommandsPrintNotImplemented -count=1
- GOCACHE=$(pwd)/.tmp/go-build go vet ./internal/clean/...
- GOCACHE=$(pwd)/.tmp/go-build go build ./...

Notes
- Full go test ./... currently has pre-existing unrelated failures in internal/cli and internal/scaffold around periphery.yml assertions.