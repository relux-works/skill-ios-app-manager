# TASK-260302-3qnpd4: wire-registry-into-root

## Description
Replace 7 old per-module CLI commands with registry-driven commands in root.go. Delete old CLI files. Update all integration tests.

## root.go changes

1. Add blank imports for all module packages (triggers init() registration):
   _ "github.com/relux-works/ios-app-manager/internal/appconfig"
   _ "github.com/relux-works/ios-app-manager/internal/httpclient"
   _ "github.com/relux-works/ios-app-manager/internal/ioc"
   _ "github.com/relux-works/ios-app-manager/internal/relux"
   _ "github.com/relux-works/ios-app-manager/internal/securestore"
   _ "github.com/relux-works/ios-app-manager/internal/tokenprovider"
   _ "github.com/relux-works/ios-app-manager/internal/utilities"
   And import "github.com/relux-works/ios-app-manager/internal/registry"

2. Replace these 7 lines:
   newIocCommand(opts),
   newReluxCommand(opts),
   newSecureStoreCommand(opts),
   newUtilitiesCommand(opts),
   newTokenProviderCommand(opts),
   newHttpClientCommand(opts),
   newAppConfigCommand(opts),
   
   With a loop:
   for _, mod := range registry.AllSorted() {
       cmd.AddCommand(NewSetupCommand(mod, opts))
   }

## Delete old CLI files

Delete these 7 files:
- internal/cli/ioc.go
- internal/cli/relux.go
- internal/cli/secure_store.go
- internal/cli/token_provider.go
- internal/cli/utilities.go
- internal/cli/http_client.go
- internal/cli/app_config.go

## Test updates

Critical: NewSetupCommand requires confirmation (Proceed? [y/N]). Old tests dont pass --yes.

For ALL integration test files (ioc_test.go, secure_store_test.go, http_client_test.go, app_config_test.go):
- Every executeRootCommand call that runs "setup" must add "--yes" flag:
  executeRootCommand("--config", configPath, "ioc", "setup", "--yes")
- OR for secure-store: executeRootCommand("--config", configPath, "secure-store", "setup", "--access-group", "group.com.example.demo", "--yes")

Completion message changes:
- "SwiftIoC setup complete" → "IoC setup complete" (mod.Name = "IoC")
- Check all register.go Name fields and update test assertions accordingly

SecureStore access-group validation tests:
- Tests TestSecureStoreSetupMissingAccessGroupNoConfigGroups, TestSecureStoreSetupMissingAccessGroupWithConfigGroups, TestSecureStoreSetupWrongAccessGroup still need to pass
- The errors now come from Plan() instead of CLI validateAccessGroup
- Error messages are the same so assertions should still match
- BUT: cobra marks --access-group as Required, so missing flag → cobra error before Plan() runs
- Need to check: cobra Required flag error vs our custom error. If cobra catches first, test expectations may differ
- Solution: SecureStore ExtraFlags has Required: true, so cobra will error on missing flag. Tests for missing --access-group will get cobra error not our custom message. Update: change ExtraFlags Required to false and let Plan() handle validation instead (this gives better error messages with config context)

IMPORTANT: After changes, run: go test ./internal/cli/... and verify ALL tests pass.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
