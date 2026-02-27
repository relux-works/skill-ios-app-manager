# STORY-260227-219uhe Progress

## Status: done

## Changes

### 1. Scaffold entitlements (`internal/scaffold/entitlements.go`)
- Added `keychain-access-groups` entry to `GenerateEntitlements()`
- Format: `$(AppIdentifierPrefix)<bundleId>.shared`
- Always included when `bundleId` is non-empty (same pattern as `aps-environment`)
- Added `keychainAccessGroup()` helper function

### 2. Configuration+Keychain.swift (`internal/scaffold/configuration_keychain.go`)
- New `GenerateConfigurationKeychain(cfg)` function
- Generates `Configuration+Keychain.swift` with constants:
  - `serviceName` = `<bundleId>` (for keychain service identification)
  - `accessGroup` = `<teamId>.<bundleId>.shared` (runtime-resolved group)
- Generated at init time as part of scaffold

### 3. Scaffold integration (`internal/scaffold/scaffold.go`)
- Added `Configuration+Keychain.swift` to `planFiles()` output

### 4. Tests
- `entitlements_test.go`: assertions for keychain-access-groups in plist
- `configuration_keychain_test.go`: unit test for constants generation
- `scaffold_test.go`: file existence + content assertions for both entitlements and config

## All tests passing
