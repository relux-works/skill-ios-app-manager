# Batch 04: Commits 31-40

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### 638509a Merge pull request #4 from connect-app/feature/CNTFRONT-437

Merge commit, see feature commit `cd711ae` (Contact card stack in contact page) in batch 03.

---

### db699dc Live Activity with loading/error states. Intermediate commit

**What**: Major Live Activity refactoring (~1335 lines added, ~532 removed). Extracts shared models into a new `ConnectShared` Swift Package (local SPM package with `LiveActivityModels`, `SharedActivityConstants`, `SharedDataModels`). Introduces `MessageDisplay` model for multi-message support in Live Activity (replaces single `lastMessage` with `lastMessages` array). Adds loading/error state handling to `SendMessageIntent` — shows error with 2.5s auto-recovery on send failure, shows loading state after successful send before fetching next contact. Adds `AppDelegate` (86 lines) for push notification registration. Creates `PushNotificationService` (174 lines) for APNs device token management and Live Activity push token registration. Refactors `LiveActivityManager` with improved contact loading flow. Rewrites both `LockScreenView` and `DynamicIslandViews` to support multiple messages and error states. Adds logo image assets to widget extension. Adds `isSending` state flag. Moves `SharedConstants.LiveActivity` constants to `SharedActivityConstants` in the shared package. Refactors `SharedCrmService` to remove duplicated code (now imports from ConnectShared). Adds shared log file writing for intent debugging via App Group container.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No shared Swift Package scaffolding for extensions**
   - What's missing: `ConnectShared` is a local Swift Package (`Package.swift` with sources) used to share code between main app and widget extension. Creating a local SPM package for shared types (models, constants) is a standard pattern when extensions need access to main app types. `ios-app-manager` has no support for creating local Swift packages.
   - Severity: HIGH (every app with extensions needs shared code. This is a local SPM package, not a Tuist module — it's a different pattern from `module create`. In a Tuist-managed project, this would likely be a shared Tuist target, but the pattern of extracting shared code into a separate compilation unit is universal.)
   - Suggested solution: `ios-app-manager shared-package create <Name>` or integrate into the extension scaffolding (`extension create` auto-generates a shared package). For Tuist projects, this could be a shared module with `--shared-with-extension` flag.

2. **No push notification scaffolding**
   - What's missing: `AppDelegate` (push notification registration, `didRegisterForRemoteNotificationsWithDeviceToken`) and `PushNotificationService` (token management, Live Activity push token registration via API) — 260 lines of APNs infrastructure. No CLI support for push notification setup beyond the existing `push send/token` commands which are for testing, not app-level infrastructure.
   - Severity: MEDIUM (push notifications are common in CRM/messaging apps. The boilerplate is substantial: AppDelegate, token lifecycle, server registration.)
   - Suggested solution: `ios-app-manager push setup` that generates AppDelegate with push registration, token service, and Info.plist entries for push capabilities.

---

### 6de4d7f Live Activity: fix avatar display, UI improvements

**What**: Architectural refactoring of `SendMessageIntent` — moves the real implementation from widget extension target to main app target (widget extension keeps only a stub, since `LiveActivityIntent` always runs in the main app process per Apple docs). Fixes avatar display in Dynamic Island by adding validation (`uiImage.size.width > 0`). Moves contact name from `ExpandedCenterView` to `ExpandedLeadingView` (next to avatar). Adds fallback person icon when name is empty. Various UI size/spacing tweaks across Dynamic Island and Lock Screen views.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — this is a Live Activity architectural fix and UI polish
- Manual edits:
  - `xflow-ios/Intents/SendMessageIntent.swift` — full implementation (327 lines, moved from extension)
  - `XFlowLiveActivity/Intents/SendMessageIntent.swift` — stub (10 lines)
  - `XFlowLiveActivity/Views/DynamicIslandViews.swift` — avatar fix, layout changes
  - `XFlowLiveActivity/Views/LockScreenView.swift` — UI improvements
- Notes: The pattern of `LiveActivityIntent` running in main app process (not extension) is an important iOS architectural detail that a scaffolding tool should get right. If extension scaffolding is added, it should generate the intent stub in the extension and the real implementation in the main app target.

---

### 739866c Enable Xcode localization settings

**What**: Enables Xcode's built-in localization analysis — adds `CLANG_ANALYZER_LOCALIZABILITY_NONLOCALIZED = YES` (warns about unlocalized strings) and `STRING_CATALOG_GENERATE_SYMBOLS = YES` (generates typed symbols for string catalogs) to both Debug and Release build settings in `project.pbxproj`. Also bumps `LastUpgradeCheck` to 2620.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: **No localization build settings in `init` scaffolding.** When `ios-app-manager init` generates Tuist manifests, it doesn't set localization-related build settings. In Tuist, these would go in `Settings` configuration on the project or target level. `CLANG_ANALYZER_LOCALIZABILITY_NONLOCALIZED` and `STRING_CATALOG_GENERATE_SYMBOLS` are best-practice settings for any localized app.
- Severity: LOW (one-time manual setting, but it's a common best practice that could be baked into `init`)
- Suggested solution: Add to default build settings in `init` or as an optional `--localization` flag. Also relates to batch 01's MEDIUM ALARM about no localization scaffolding — this is the build settings side of that gap.

---

### c91cbdd Live Activity: add custom Codable for APNs push and polling toggle

**What**: Adds custom `Codable` implementation to `ContactActivityAttributes.ContentState` (58 lines) — encodes `lastUpdated` as Unix timestamp (`TimeInterval`) for APNs payload compatibility (APNs sends JSON, can't handle Swift `Date` encoding). Adds `usePushNotifications` toggle in `SharedConstants.LiveActivity` (defaults to `false`) — when true, uses APNs push for Live Activity updates; when false, falls back to polling. Conditionalizes `startPolling()` call based on this flag.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — Live Activity infrastructure code, APNs integration detail
- Manual edits:
  - `ConnectShared/Sources/LiveActivityModels.swift` — custom Codable for APNs compatibility
  - `connect-ios/Models/LiveActivityManager.swift` — conditional polling based on push toggle
  - `connect-ios/Shared/SharedConstants.swift` — `usePushNotifications` flag
- Notes: The custom Codable for APNs is a gotcha — Live Activity content states updated via APNs must use primitive types (Unix timestamps, not `Date`). If Live Activity scaffolding is ever added, this should be documented or generated with the correct Codable implementation.

---

### 6f8a8c2 Merge pull request #5 from connect-app/feature/CNTFRONT-430

Merge commit, see feature commits `db699dc`, `6de4d7f`, `739866c`, `c91cbdd` above.

---

### 18677ea Add pipe-delimited CSV table parsing and display in CRM context

**What**: Adds rich content parsing — detects pipe-delimited tables in text content and renders them as scrollable tables. New files: `CSVParser` (145 lines — heuristic parser detecting `|` delimiters, header detection, column normalization), `RichContent` model (26 lines — `ContentBlock` enum with `.text` and `.table` cases), `RichContentView` (61 lines — renders mixed text/table blocks), `TableBlockView` (137 lines — horizontally scrollable table with auto-calculated column widths, alternating row colors, header highlighting). Integrates into `MotivationsPanelView` by replacing plain `Text` with `RichContentView` in collapsible sections.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — domain-specific feature code within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/UI/Components/RichContentView.swift` — mixed content renderer
  - `Packages/CRM/Sources/CRM/UI/Components/TableBlockView.swift` — table display component
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/RichContent.swift` — content block model (or `UI/Model/`)
  - `Packages/CRMImpl/Sources/CRMImpl/Utilities/CSVParser.swift` — parser utility
- Notes: `CSVParser` and `TableBlockView` are generic enough to live in a shared utilities or UI components module. Pipe-delimited table parsing is reusable for any feature displaying API-generated content with embedded tables.

---

### 5c1b433 Merge pull request #6 from connect-app/feature/CNTFRONT-442

Merge commit, see feature commit `18677ea` (CSV table parsing) above.

---

### 5088a75 Rename app from Connect to xflow (org.xflow)

**What**: Massive rename — 150 files changed. Renames app from "connect-ios" to "xflow-ios" and organization from "com.connectapp" to "org.xflow". Changes include: directory renames (`connect-ios/` → `xflow-ios/`, `ConnectCRMLiveActivity/` → `XFlowLiveActivity/`, `ConnectShared/` → `XFlowShared/`), bundle identifier (`com.connectapp.connect-ios2` → `org.xflow`), App Group identifier (`group.com.connectapp.connect-ios2` → `group.org.xflow`), Keychain access group, URL scheme (`connectcrm://` → `xflow://`), app entry point rename (`ConnectCRMApp` → `XFlowApp`), widget bundle rename, shared package rename, and file header comments across all 100+ Swift files. Also renames `.idea/` project file. Entitlements updated for both main app and extension.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: **No app rename / rebrand tooling.** Renaming an iOS app touches: bundle identifiers (main + test + UI test + extension targets), App Group identifiers, Keychain access groups, URL schemes, directory names, file headers, import statements for shared packages, xcodeproj/pbxproj target names, scheme files, entitlements files, widget URL schemes. This commit changes 150 files for a rename. In a Tuist-managed project, the rename would be simpler (Tuist manifests are the source of truth), but still substantial.
- Severity: MEDIUM (app renames are infrequent but extremely tedious when they happen. A Tuist-based project centralizes much of this in manifests, but shared packages, entitlements, and Keychain groups still need coordinated updates.)
- Suggested solution: `ios-app-manager rename --from <old> --to <new>` that updates: config file identifiers, regenerates manifests, updates entitlements, updates shared package names. Since ios-app-manager generates the project, it knows all the places that reference the app identity.

---

### 0073f86 Fix Team ID and update version to 0.1

**What**: Fixes Team ID across all targets — changes from `CC258N857A` (old developer account) to `H446YY77RR` (new team). Updates bundle identifiers from `org.xflow` to `org.xflow.app` (more conventional with `.app` suffix) across main app, tests, UI tests, and Live Activity extension targets. Updates App Group to `group.org.xflow.app`, Keychain access group to `H446YY77RR.org.xflow.app`, Keychain keys prefix. Updates entitlements for both main app and extension. Changes local development URL to ngrok. Switches scheme launch config from Debug to Release. Wraps `AvatarCache` debug prints in `#if DEBUG` blocks (~20 prints protected). Updates `SharedCrmService` debug base URL.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No Team ID management**
   - What's missing: Team ID (`DEVELOPMENT_TEAM`) is a critical build setting that must be consistent across all targets (main app, tests, extensions). Changing it requires updating every target in the project. `ios-app-manager init` doesn't manage Team ID.
   - Severity: LOW (in Tuist, Team ID is set in `Tuist.swift` or project `Settings` — single source of truth. But it should be part of the config.)
   - Suggested solution: Add `team_id` field to `ios-app-manager.json` config. `init` uses it for all generated targets.

2. **No bundle identifier suffix convention enforcement**
   - What's missing: The rename from `org.xflow` to `org.xflow.app` (and cascading changes to `org.xflow.app.tests`, `org.xflow.app.uitests`, `org.xflow.app.liveactivity`) shows that bundle identifier structure matters. Extensions and test targets derive their identifiers from the main app's bundle ID. No scaffolding enforces or manages this convention.
   - Severity: LOW (Tuist handles this via `bundleId` in target definitions, but the convention should be set correctly at `init` time.)
   - Suggested solution: Config `bundle_id` should be the base, with automatic suffixes for test (`.tests`), UI test (`.uitests`), and extension targets.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| 638509a Merge #4 | SKIP | Merge commit |
| db699dc Live Activity states | NO | No shared package scaffolding, no push notification setup |
| 6de4d7f Avatar fix | YES | Live Activity architectural fix, manual |
| 739866c Localization settings | PARTIAL | No localization build settings in init |
| c91cbdd APNs Codable | YES | Live Activity infrastructure, manual |
| 6f8a8c2 Merge #5 | SKIP | Merge commit |
| 18677ea CSV table parsing | YES | Domain-specific feature code, manual |
| 5c1b433 Merge #6 | SKIP | Merge commit |
| 5088a75 App rename | NO | No rename/rebrand tooling |
| 0073f86 Team ID fix | PARTIAL | No Team ID or bundle ID convention management |

### Critical ALARMs (sorted by severity)

1. **HIGH: No shared Swift Package scaffolding** — `ConnectShared` (local SPM package for sharing types between app and extension) is a universal pattern for apps with extensions. 430+ lines of shared models/constants with no CLI support. In Tuist, this maps to a shared target, but the need for explicit shared code extraction remains.

2. **MEDIUM: No push notification scaffolding** — AppDelegate + PushNotificationService (260 lines) for APNs device token management. Common infrastructure with no CLI support beyond `push send/token` testing commands.

3. **MEDIUM: No app rename tooling** — 150 files changed for a rename. `ios-app-manager` generates the project, so it knows every identifier location. A `rename` command could automate this entirely.

4. **LOW: No localization build settings** — `CLANG_ANALYZER_LOCALIZABILITY_NONLOCALIZED` and `STRING_CATALOG_GENERATE_SYMBOLS` are best practices not included in `init`.

5. **LOW: No Team ID / bundle ID convention management** — Team ID and bundle identifier suffix conventions should be part of config and enforced across all generated targets.

### Observations

- **Batch 04 is split between Live Activity maturation and project identity changes.** Commits 31-35 finish the Live Activity feature (loading/error states, shared package extraction, APNs integration), while commits 38-40 rename the entire app and fix identity settings.
- **The `ConnectShared` local SPM package pattern is notable** — it's the first time the codebase extracts shared code into a separate compilation unit. This is exactly how Tuist modules work, but done manually via SPM. In the scaffolded Tuist project, this would be a shared Tuist target (e.g., `module create Shared --type shared`), but the extension-sharing aspect adds complexity that `module create` doesn't currently handle.
- **LiveActivityIntent architectural insight** — the discovery that `LiveActivityIntent.perform()` always runs in the main app process (not the extension) is an important pattern. The extension contains only a stub. If Live Activity scaffolding is ever added, this must be handled correctly to avoid the same mistake the codebase went through (full implementation in extension → stub in extension + real impl in main app).
- **App renames are a significant pain point** — 150 files for a rename is the kind of tedious, error-prone task that tooling should eliminate. Since `ios-app-manager` is the source of truth for project generation, it has all the information needed to perform a rename safely.
- **`#if DEBUG` wrapping of log statements** (in 0073f86) is a cleanup that should have been done from the start. If scaffolding generates logging utilities, they should use conditional compilation by default.
