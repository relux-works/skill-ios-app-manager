# Batch 03: Commits 21-30

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### fb51fe1 Add optimistic UI for comments and fix delete suggested task empty response

**What**: Extends the optimistic UI pattern (from batch 02's message sending) to comments. New `PendingComment` model (50 lines), `PendingCommentBubble` component (126 lines) with yellow comment styling and status indicator. Rewrites `addComment()` to create pending comment immediately, update status on success/failure, match with server comments via polling. Adds retry/cancel for failed comments. Also fixes `deleteSuggestedTask` — server returns empty body, switches from `delete<T>` to `deleteVoid` (new `NetworkManager` method). Adds optimistic hide for suggested task deletion.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — extends existing optimistic UI pattern within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/PendingComment.swift` — pending comment model (reuses `MessageStatus` enum from batch 02)
  - `Packages/CRM/Sources/CRM/UI/Components/PendingCommentBubble.swift` — pending comment bubble (126 lines)
  - Edits to `CRM.Business+Flow.swift` — comment send/retry/cancel effects, server matching
  - Edits to networking layer — `deleteVoid` method for empty-body DELETE responses
- Notes: Same optimistic UI pattern as `PendingMessage` — the two share `MessageStatus` enum and `MessageStatusView` component. Copy-paste pattern continues.

---

### 15f8063 Add return to previous contact notification after auto-navigation

**What**: Major navigation flow rework. Replaces the "timer countdown → navigate to next contact" pattern with immediate navigation + "return to previous" notification. Changes: `AppDestination.contactDetail` gains `previousContactId` parameter, `ContactDetailViewModel` gets return notification timer (3s auto-dismiss), `shouldReturnToPrevious` flag. Sends current contact ID as `previousContactId` when navigating to next. After message/comment send, uses `navigateToNextContactImmediately()` instead of timer-based flow. Removes `startNextContactFlow()`, `showNextContactNotification`, and countdown notification UI. New localization keys: `button.returnBack`, `contact.returnToPrevious`.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — navigation architecture and ViewModel logic
- Manual edits to:
  - `AppRouter.swift` — add `previousContactId` to `contactDetail` case
  - `ContentView.swift` — pass `previousContactId` through navigation
  - `ContactDetailViewModel` — return notification timer, `shouldReturnToPrevious` flag
  - `ContactDetailView` — `.onChange` handler for return navigation
  - Localization files — new keys

**ALARM**:
- What's missing: **No navigation route scaffolding.** The `AppDestination` enum (route definitions) and `ContentView` (route → view mapping) are fundamental app infrastructure with no CLI support. Every time a feature adds parameters to navigation routes, both files must be manually updated. In a Tuist modular project, navigation definitions would need to be coordinated across module boundaries.
- Severity: LOW (this is inherently app-specific, and the current monolithic `AppDestination` enum wouldn't survive modularization anyway — each feature module would define its own routes)
- Suggested solution: N/A — navigation architecture is too app-specific for scaffolding. Document as manual setup.

---

### 6dfe4d0 Fix search results list to open from bottom and use dynamic spacing

**What**: Fixes search results dropdown positioning. Moves `searchResultsList` from `.overlay(alignment: .bottom)` on `secondInputBlock` to a VStack above it with 10pt spacing. Adds `scaleEffect(x:1, y:-1)` flip trick on ScrollView and individual cards so the list opens from bottom.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — UI layout fix in ContactDetailView
- Manual edit to `ContactDetailView.swift` (18 lines changed)

---

### a7f519f Fix: show empty contact list instead of No CRM access when tabs are unavailable

**What**: Removes "No CRM access" error state from contact list. When no tabs are available and `getNextContact` returns nil, now shows empty contact list instead of lock icon with "No access" message. Renames `showNoAccessState` → `showEmptyContactsList`, `setNoAccessState()` → `setShowEmptyContactsList()`, removes `showNoAccess` computed property and `noAccessView` entirely. Also removes `.reversed()` on search results (handled by flip trick from `6dfe4d0`).

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — UX logic fix within CRM module
- Manual edits to:
  - `ContactListViewModel.swift` — rename state/methods
  - `ContactListView.swift` — remove `noAccessView`, update references
  - `ContactDetailViewModel.swift` — remove `.reversed()` on search results

---

### e72d799 Redesign return to previous contact notification bubble

**What**: Simplifies the "return to previous contact" notification. Moves it from above pending messages to below the message input (first in flipped ScrollView). Increases timer from 3s to 6s. Replaces HStack layout (text + button) with VStack. Removes glass material background — now just a plain text button with red foreground color.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — UI polish within ContactDetailView
- Manual edits to:
  - `ContactDetailView.swift` — move notification position, simplify layout
  - `ContactDetailViewModel.swift` — change timer from 3s to 6s

---

### 64e1b08 Seamless contact transition with preloading after message/comment send

**What**: Implements seamless contact transitions. New `ContactDataCache` singleton (45 lines) — in-memory cache that stores preloaded contact data, consumed (deleted) after retrieval. New `PreloadedContactData` model (19 lines). `initiateSeamlessTransition()` method: shows loader, blocks input, fetches next contact ID, preloads contact data + messages in parallel, waits minimum 1.5s delay, stores in cache, triggers navigation. On destination view's `onAppear`, synchronously checks cache and applies preloaded data (skipping API calls). Reduces polling interval from 20s to 5s. Adds `matchedGeometryEffect` animation between `MessageInputBubble` and `PendingMessageBubble`. Moves search results back to `.overlay` positioning with dynamic `inputBlockHeight` tracking via `GeometryReader`.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — advanced navigation + data preloading pattern
- Manual files:
  - `Packages/CRMImpl/Sources/CRMImpl/Services/ContactDataCache.swift` — preloaded data cache (or shared infrastructure module)
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/PreloadedContactData.swift` — preloaded data model
  - Extensive edits to `ContactDetailViewModel` — seamless transition logic (~108 lines added)
  - Extensive edits to `ContactDetailView` — preloaded data application, animation, loader UI
  - Localization: `contact.loadingNextContact` key

**ALARM**:
- What's missing: **No data preloading/prefetch infrastructure.** The seamless transition pattern (fetch next screen's data before navigating, store in cache, apply on appear) is a general UX optimization pattern. The implementation is ~150+ lines across 3 new files + extensive ViewModel changes. This pattern could benefit any feature that navigates between data-heavy screens.
- Severity: LOW (specific to CRM workflow where rapid contact cycling is the core UX — unlikely to be generalized into scaffolding)
- Suggested solution: N/A for scaffolding. Document as architectural pattern.

---

### 85fcff0 Add contact filters and statistics for CRM list

**What**: Large feature (~1300 lines added). Implements contact filtering and statistics for CRM list. New models: `ContactFilterParams` (filter state with readiness, urgency, topic type, funnel status, follow-up, analyzing, date range), `ContactStatistics` (urgency counts), `Readiness`/`ResponseUrgency`/`TopicType`/`FunnelStatus` enums (all `CaseIterable` with display names). New components: `FilterPopupView` (422 lines — sheet with multi-select chips per category, date pickers, reset/apply), `ContactStatisticsView` (158 lines — urgency counts with emoji icons). New API endpoints: `getContactStatistics`, `getFilteredContactIds`, `analyzeChats` (bulk). `ContactListViewModel` gains filter state, statistics loading, filter application, bulk re-analysis support. `ContactListView` adds statistics header and filter sheet trigger. 31 new localization keys (en + ru) for filter labels.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — feature code across CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/ContactFilter.swift` — filter models + enums (234 lines)
  - `Packages/CRM/Sources/CRM/UI/Components/FilterPopupView.swift` — filter sheet (422 lines)
  - `Packages/CRM/Sources/CRM/UI/Components/ContactStatisticsView.swift` — statistics display (158 lines)
  - `Packages/CRM/Sources/CRM/Data/Api/CrmService.swift` — 3 new API methods (statistics, filtered IDs, bulk analyze)
  - `Packages/CRM/Sources/CRM/Business/CRM.Business+Action.swift` — filter/statistics actions
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+State.swift` — filter state, statistics
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+Flow.swift` — filter/statistics effects
  - Localization: 31 new keys for filter categories and labels

**ALARM**:
- What's missing: **No filter/list infrastructure scaffolding.** Filtering (multi-select chips, date range, apply/reset) and statistics (aggregated counts with visual indicators) are standard CRM/list patterns. The implementation is 800+ lines of new UI code plus model/service code. Every list-based feature (contacts, organizations, notifications) would benefit from similar filter infrastructure.
- Severity: MEDIUM (filtering is a universal list pattern, and 400+ lines for a filter sheet is substantial boilerplate that repeats across features)
- Suggested solution: Consider a generic `FilterableList<T, Filter>` pattern or at minimum a reusable `MultiSelectChipView` component. Not a CLI command, but a shared UI component module.

---

### 121898f Widget

**What**: First part of Live Activity / Widget implementation. Adds foundational models and shared infrastructure: `ContactActivityAttributes` with `ContentState` (Activity Attributes for Live Activity — contact ID, name, phone, messages, urgency, follow-up), `SharedConstants` (App Group identifier, Keychain access group, UserDefaults keys, Live Activity config, urgency emoji mapping), `SharedUserDefaults` (App Group-backed UserDefaults wrapper for sharing data between app and widget extension). Adds entitlements files for both main app and widget extension.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No widget/extension scaffolding**
   - What's missing: `ios-app-manager` has no support for creating app extensions (Widget, Live Activity, etc.). This involves: new target in Tuist manifests, extension entitlements, shared code between main app and extension, App Group configuration, Info.plist for extension.
   - Severity: HIGH (app extensions are a common iOS pattern. Widget/Live Activity requires coordinated setup across Tuist Project.swift, entitlements, App Groups, and shared code modules. Manual setup is error-prone.)
   - Suggested solution: `ios-app-manager extension create <Name> --type widget-extension` that generates: Tuist target definition, entitlements with App Group, shared module for App Group data, extension entry point.

2. **No App Group / shared data scaffolding**
   - What's missing: `SharedConstants` (App Group identifier, shared UserDefaults keys) and `SharedUserDefaults` (App Group-backed wrapper) are infrastructure for app ↔ extension communication. No CLI support for this.
   - Severity: MEDIUM (every app with extensions needs this, and it must match entitlements configuration)
   - Suggested solution: Could be part of `extension create` — auto-generate shared constants module with App Group identifier matching entitlements config.

---

### 37dd5d3 Widget

**What**: Second part — the actual Live Activity implementation (~2480 lines). Creates `ConnectCRMLiveActivity/` extension target with: `ContactLiveActivity` widget configuration, `LockScreenView` (249 lines — lock screen presentation with contact info, messages, send button), `DynamicIslandViews` (296 lines — compact/expanded Dynamic Island views with avatar, urgency emoji, contact name, message preview), `SendMessageIntent` (218 lines — AppIntent for sending messages directly from Live Activity with shared Keychain token access and next contact auto-loading). Main app additions: `LiveActivityManager` (433 lines — manages Live Activity lifecycle: start/stop/update, background polling with 30s interval, work hours scheduling, avatar caching to App Group container), `LiveActivityModels+MainApp.swift` (62 lines — main app extensions for creating content states from `CrmContactExtended`). Updates `SharedConstants.swift` and `SharedCrmService.swift` (372 lines — lightweight CRM service for widget extension with direct API calls). Also modifies `KeychainManager` for shared Keychain access group. Commits `.idea/` files (IntelliJ IDEA project files — should be gitignored).

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No Live Activity scaffolding** (extends 121898f alarm)
   - What's missing: Live Activity requires: ActivityAttributes definition, Widget configuration, Lock Screen views, Dynamic Island views (compact leading/trailing, expanded regions, minimal), AppIntents for interactive buttons. All of this is ~900 lines of boilerplate per Live Activity.
   - Severity: HIGH (Live Activities are increasingly common for CRM/messaging apps. The boilerplate is substantial and follows a strict pattern.)
   - Suggested solution: `ios-app-manager extension create <Name> --type live-activity` with templates for all required views and intents.

2. **No shared service scaffolding for extensions**
   - What's missing: `SharedCrmService` (372 lines) duplicates networking code because the widget extension can't use the main app's `NetworkManager` directly. Shared Keychain access for tokens also required manual `KeychainManager` modification.
   - Severity: MEDIUM (every extension that makes API calls needs its own lightweight networking, duplicating main app code)
   - Suggested solution: When creating extensions, generate a shared networking module that both main app and extension can import. The `secure-store setup` could support `--shared-access-group` for Keychain sharing.

3. **`.idea/` files committed** — not a scaffolding issue, but `.gitignore` gap (see batch 01 ALARM about missing .gitignore in `init`).

---

### cd711ae Contact card stack in contact page

**What**: Implements a stacked card UI for contact display in the contact detail page. New components: `ContactCardStack` (249 lines — orchestrates stack of 5 cards with animation states idle/flyingOut/complete), `MainContactCard` (229 lines — primary card with avatar + glass info bubble), `BackgroundCardView` (97 lines — placeholder cards behind main with shimmer loading), `StackCard` (246 lines — unified card component for all positions). New View extensions: `View+Glass` (81 lines — Liquid Glass effect for iOS 26+ with fallback to material for older iOS), `View+Shimmer` (72 lines — shimmer loading animation modifier). Updates `ContactDetailView` to use `ContactCardStack` instead of inline contact info. Updates `VoiceRecordButton` layout.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — custom UI components within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/UI/Components/ContactCardStack/ContactCardStack.swift` — stack orchestrator
  - `Packages/CRM/Sources/CRM/UI/Components/ContactCardStack/MainContactCard.swift` — main contact card
  - `Packages/CRM/Sources/CRM/UI/Components/ContactCardStack/BackgroundCardView.swift` — placeholder cards
  - `Packages/CRM/Sources/CRM/UI/Components/ContactCardStack/StackCard.swift` — unified card
  - `Packages/Components/Sources/Components/Extensions/View+Glass.swift` — Liquid Glass extension (shared, reusable)
  - `Packages/Components/Sources/Components/Extensions/View+Shimmer.swift` — shimmer effect (shared, reusable)
- Notes: `View+Glass` and `View+Shimmer` are generic extensions that belong in a shared UI components module. They're reusable across features. `View+Glass` proactively adopts iOS 26 Liquid Glass with fallback — good pattern for shared code.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| fb51fe1 Optimistic comments | YES | Extends existing pattern, manual |
| 15f8063 Return to previous | YES | Navigation route changes, manual |
| 6dfe4d0 Search results fix | YES | One-file UI fix |
| a7f519f Empty contact list | YES | UX logic fix |
| e72d799 Notification redesign | YES | UI polish |
| 64e1b08 Seamless transition | YES | Data preloading pattern, manual |
| 85fcff0 Contact filters | PARTIAL | No filter/list infrastructure |
| 121898f Widget (models) | NO | No widget/extension scaffolding |
| 37dd5d3 Widget (full impl) | NO | No Live Activity scaffolding, no shared service |
| cd711ae Contact card stack | YES | Custom UI components, manual |

### Critical ALARMs (sorted by severity)

1. **HIGH: No widget/extension scaffolding** — App extensions (Widget, Live Activity) require coordinated setup across Tuist targets, entitlements, App Groups, shared code modules. Two commits (121898f + 37dd5d3) add ~2770 lines of extension infrastructure with no CLI support. This is a common iOS pattern that's increasingly expected in CRM/messaging apps.

2. **MEDIUM: No shared service scaffolding for extensions** — Widget extensions can't use the main app's networking directly, leading to `SharedCrmService` (372 lines) duplicating API calls. Shared Keychain access also requires manual setup.

3. **MEDIUM: No filter/list infrastructure** — Filtering (multi-select chips, date range, apply/reset) at 400+ lines per filter sheet is substantial boilerplate that repeats across list features.

4. **LOW: No data preloading pattern** — Seamless transitions with prefetch are nice UX but too app-specific for scaffolding.

5. **LOW: No navigation route scaffolding** — Route definition and mapping are fundamental but too app-specific.

### Observations

- **Batch 03 shows two distinct phases**: commits 21-26 are iterative CRM feature polishing (optimistic comments, navigation flow rework, UI fixes), while commits 27-30 introduce significant new infrastructure (filters, Live Activities, card stacks).
- **Live Activity is the biggest scaffolding gap found so far** — at ~2770 lines across 2 commits, it's the largest single feature with zero CLI support. Extensions require Tuist target coordination, entitlements, App Groups, and shared code — all areas where scaffolding would save significant manual effort.
- **iOS 26 Liquid Glass adoption** (`View+Glass`) is forward-looking — the codebase is already preparing for the next iOS version with graceful fallbacks. This is a good pattern for the shared UI module.
- **Pattern duplication continues** — `PendingCommentBubble` copies the optimistic UI pattern from `PendingMessageBubble` (batch 02). Both use `MessageStatus` and `MessageStatusView`. A generic `PendingItem<T>` would reduce duplication.
