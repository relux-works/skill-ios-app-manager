Implemented component integration for ios-app-manager root.

Changes:
- Updated component interfaces to required contracts with GenerateOpts, ModuleOpts, ManifestEdit, and ProjectStatus.
- Added AppManager implementation that orchestrates TuistProjectManager and ReluxManager for CreateModule, provides project Status by loading config and scanning Packages, and delegates DeleteModule via manifest edit.
- Wired CLI status command through AppManager and added constructor injection via NewRootCommandWithAppManager for future commands.
- Added mock-based AppManager unit tests and CLI status tests for real + injected manager flows.

Verification:
- go test ./internal/components/... (pass)
- go test ./internal/cli/... (pass)
- go vet ./... (pass)
- go build ./cmd/ios-app-manager/ (pass)