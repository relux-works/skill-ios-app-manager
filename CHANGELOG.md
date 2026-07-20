# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- Added versioned runtime-profile config, typed Swift backend/profile descriptors, Tuist configurations and schemes, Firebase public-client input validation, and policy-aware AppConfig templates.
- Added generic runtime-profile schema, example configuration, migration/removal guidance, and creation/update/removal/invalid-config golden tests.
- Added explicit typed Firebase identity-sharing groups that preserve fail-closed duplicate rejection and environment-specific API/auth/storage/grant/quota realms.
- Added a non-destructive `fireauth-relux setup` capability with exact FireAuthRelux/FireAuthKit pins, Tuist product overrides, validated bundle configuration loading, in-process test injection, and app-process deterministic UI-test selection without live Firebase traffic.

### Fixed

- Kept generated package-project configurations aligned with app runtime profiles, canonicalized Tuist `PackageSettings` initializer order, and removed duplicate SharedConfig package/product dependencies during forced scaffold convergence.
- Updated SecureStore builder configuration to use the canonical generated app-group property instead of the removed Info.plist-shaped accessor.
- Replaced typed legacy `Configuration` arrays, obsolete app schemes, and existing `PackageSettings` configuration arguments cleanly when adopting runtime profiles in mature projects while retaining unrelated custom schemes.
- Made SharedConfig dependency insertion comma-safe and changed SecureStore/AppConfig Registry integration to preserve unrelated custom composition.
- Changed TokenProvider setup to converge generator-owned Registry blocks instead of regenerating mature composition roots, and made managed dependency/build-setting updates byte-idempotent.

## [v0.10.1] - 2026-07-14

### Changed

- Updated the scaffolded `SwiftUIRelux` dependency to version 9.
- Refreshed the README with an explicit Relux stack summary and ecosystem context.

### Fixed

- Fixed Tuist target list patching so inserted targets keep valid comma placement in generated manifests.

### Notes

- Earlier releases were tagged before changelog tracking started in-repo.
