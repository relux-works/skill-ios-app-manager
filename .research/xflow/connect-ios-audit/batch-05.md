# Batch 05: Commits 41-50

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### 4ba948e Fix avatar loading in Live Activity widget and add LRU cache for 50 avatars

**What**: Major Live Activity reliability improvements. Fixes avatar loading in widget extension (must use `Data(contentsOf:)` + `UIImage(data:)` instead of `UIImage(contentsOfFile:)` which doesn't work in widgets). Replaces single-avatar caching (delete all except current) with LRU cache for 50 avatars with 80% eviction threshold. Adds deep linking from Live Activity Lock Screen (`xflow://contact/{id}`), with `handleDeepLink` in `XFlowApp` and `handleDeepLinkToContact` in `AppRouter` (pending deep link if auth check still running). Adds stale activity cleanup on init, sync on foreground, duplicate detection/cleanup. Adds avatar prefetching for top 10 contacts. Enables APNs push notifications (`usePushNotifications = true`). Adds `aps-environment` entitlement. Bumps version to 0.1.1.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No deep linking infrastructure scaffolding**
   - What's missing: Deep linking (URL scheme handling in App.swift via `.onOpenURL`, route parsing, pending deep link storage during auth check) is a standard iOS pattern with ~50 lines of boilerplate. Every app with widgets, push notifications, or universal links needs this. No CLI support.
   - Severity: MEDIUM (common pattern, moderately complex with auth-state-aware routing)
   - Suggested solution: Consider adding deep linking setup to `init` or as `ios-app-manager deeplink setup` — generates `.onOpenURL` handler in App.swift, URL scheme registration, and router integration.

2. **No version bumping tooling**
   - What's missing: Version bumping across all targets (main app, tests, UI tests, extensions) requires editing `MARKETING_VERSION` in multiple places in pbxproj. In a Tuist project this would be in manifests, but still tedious.
   - Severity: LOW (in Tuist, version is set once in project settings — less of an issue than in raw xcodeproj)
   - Suggested solution: N/A for Tuist projects — single source of truth for version.

---

### 81e3812 Add admin mode, fix voice recording permission handling, improve post-login navigation, bump to 0.1.2

**What**: Multi-concern commit (~396 lines added across 18 files). Three distinct features:
1. **Admin mode**: Hidden admin mode activated by tapping logo 7 times (within 2s window) + password "service7". Gated features: CRM buttons on Profile, info button + motivations panel on ContactDetail, members/CRM buttons on OrganizationDetail, notifications/theme menu items. Stored in `UserDefaultsManager.isAdminMode`. Admin mode makes `isExperimental` always return `true` for non-admin users (controlled toggle only visible in admin mode).
2. **Voice recording permission fix**: Refactors `AudioRecorderService.startRecording()` — now checks `AVAudioApplication.shared.recordPermission` first (denied → immediate error, granted → immediate recording, undetermined → show `.requestingPermission` state). Adds cancel-during-permission handling via `wasCancelledDuringPermission` flag. New `RecordingState.requestingPermission` case. Auto-dismiss errors after 2s.
3. **Post-login navigation**: `navigateToMainApp()` now checks CRM access after login (was just `setRootDestination(.main)`), navigates to next contact if CRM available, otherwise to profile.
4. Version bump to 0.1.2, scheme switched from Release to Debug config.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — all three features are custom app logic
- Manual files/edits:
  - `UserDefaultsManager.swift` — `isAdminMode` property, gated `isExperimental`
  - `ProfileView.swift` — 7-tap logo handler, admin password alert, conditional CRM/menu buttons
  - `ContactDetailView.swift` — conditional info button (admin only), error auto-dismiss
  - `OrganizationDetailView.swift` — conditional members/CRM buttons (admin only)
  - `AudioRecorderService.swift` — permission state machine rewrite
  - `VoiceRecordButton.swift` — `requestingPermission` and `error` state handling
  - `AppRouter.swift` — post-login CRM check + navigation
  - Localization: 7 new keys (admin mode labels, voice permission requesting)

**ALARM**:
- What's missing: **Hardcoded password in source code.** `"service7"` is a plaintext password in `ProfileView.swift` (line 731 in diff). This is a security concern — passwords should never be in source code, especially in a mobile app where the binary can be decompiled. Not a scaffolding issue per se, but worth flagging.
- Severity: MEDIUM (security concern, not scaffolding gap)
- Suggested solution: Use server-side admin check via API, or at minimum obfuscate the check.

---

### d3f4592 Add bulk clear chat context and evaluate global context buttons

**What**: Adds two bulk operations to the CRM contact list (available when filters are active). New API endpoints: `clearChatContextBulk` (POST, takes contactIds + reanalyze flag) and `evaluateGlobalContextBulk` (POST, takes contactIds). New models: `ClearChatContextRequest/Response` (11 lines), `EvaluateGlobalContextRequest/Response` (10 lines). New views: `ClearContextConfirmationSheet` (61 lines — confirmation with reanalyze checkbox, using custom `CheckboxToggleStyle`), `GlobalContextConfirmationSheet` (40 lines — simple confirmation). `ContactListViewModel` gains 6 new state properties and 4 methods for the two operations (prepare → confirm → execute flow). UI: two new buttons in filter action section — red "Clear chat context" and blue "Update global context". 16 new localization keys (en + ru).

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — feature code within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/ClearChatContextResponse.swift` — request/response models
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/EvaluateGlobalContextResponse.swift` — request/response models
  - `Packages/CRM/Sources/CRM/UI/Views/ClearContextConfirmationSheet.swift` — confirmation with checkbox
  - `Packages/CRM/Sources/CRM/UI/Views/GlobalContextConfirmationSheet.swift` — simple confirmation
  - `Packages/CRM/Sources/CRM/Data/Api/CrmService.swift` — 2 new bulk API methods
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+Flow.swift` — bulk operation effects
  - `Packages/CRM/Sources/CRM/Business/CRM.Business+Action.swift` — bulk operation actions
  - Localization: 16 new keys
- Notes: The `CheckboxToggleStyle` is a reusable component that could live in shared UI module. The prepare → confirm → execute pattern is clean and repeats for both operations.

---

### a3700d5 Optimize contact search with parallel requests and fix highlight bug

**What**: Two improvements:
1. **Parallel search optimization**: Replaces single `searchContacts` call with parallel QUICK + MESSAGES strategy. New `SearchType` enum (`.quick`, `.messages`, `.all`). QUICK search returns immediately (name/phone/tgUsername only — no messages JOIN, fast). MESSAGES search runs in parallel, results merged after QUICK results are displayed. Adds `excludeContactIds` parameter to avoid duplicates. UI shows "Searching in messages..." indicator while message search is running.
2. **Highlight bug fix**: Rewrites `HighlightedText` from `AttributedString`-based (with broken range conversion) to `Text` concatenation approach. Splits text into highlighted/non-highlighted parts, concatenates `Text` views with `.foregroundColor()`. Fixes incorrect highlight colors — now uses yellow for matches with configurable base color passed through. Same fix applied to `ContactDetailView.highlightedText()`.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — search optimization and UI bug fix within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/SearchType.swift` — new enum (or in CrmContact.swift)
  - `Packages/CRM/Sources/CRM/UI/Components/CrmContactCard.swift` — `HighlightedText` rewrite, baseColor parameter
  - `Packages/CRM/Sources/CRM/UI/Views/ContactSearchView.swift` — parallel search, loading indicator
  - `Packages/CRM/Sources/CRM/Data/Api/CrmService.swift` — new `searchType`/`excludeContactIds` params
  - Localization: 1 new key (`crm.searchingMessages`)
- Notes: The `HighlightedText` utility (text splitting + concatenation approach) is a generic pattern useful in any search UI. Could live in shared UI components module.

---

### d7607ac Redesign input area: remove gray bubble, add black AI text background

**What**: UI polish for message input area. Removes the gray material bubble background from the entire input section. Adds black rounded rectangle background specifically for the AI operator request text (yellow sparkle icon + white text). Reduces spacing (8→4 in VStack, 10→6 in HStack). Adds `frame(minHeight: 42)` to text input. Minor padding adjustments.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — UI polish within CRM module
- Manual edit to `ContactDetailView.swift` (16 lines changed)

---

### b59f986 Add WhatsApp onboarding UI flow

**What**: Massive feature — ~2140 lines added across 17 files. Complete WhatsApp onboarding flow for users without CRM access. Architecture:
- **Models** (204 lines): `OnboardingSessionStatus` enum (9 states matching backend), `OnboardingStatusResponse` DTO (with pairing code expiry calculation), `StartOnboardingRequest`, `OnboardingProgressStep` (progress state machine), `CountryCode` (20 countries with flags for phone picker).
- **Service** (77 lines): `WhatsAppOnboardingService` — 4 API calls (start, status, refreshCode, cancel).
- **ViewModel** (615 lines): `WhatsAppOnboardingViewModel` — 6-screen state machine (loading → phoneInput → waitingServer → pairingCode → progress → completion/error). Status polling every 3s. Pairing code countdown timer. Fake progress animations for waiting (10min) and sync stages. CRM availability polling after completion.
- **Views** (7 files, ~953 lines total): `WhatsAppOnboardingView` (84 lines — screen router), `PhoneInputView` (252 lines — country picker + phone input + validation), `WaitingServerView` (115 lines — animated progress), `PairingCodeView` (258 lines — code display with countdown + instructions), `OnboardingProgressView` (116 lines — step-by-step progress), `OnboardingCompletionView` (110 lines — success with CRM button), `OnboardingErrorView` (118 lines — error with retry).
- **Router integration**: New `AppDestination.whatsAppOnboarding` case. `AppRouter` checks onboarding status on startup — routes to onboarding if no CRM access, shows completion screen if recently completed.
- **Localization**: 34 new keys (en + ru).

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No onboarding flow scaffolding**
   - What's missing: Multi-step onboarding flows (screen state machine, status polling, timer-based UI, progress steps) are a common mobile pattern, especially for service integration (WhatsApp, Telegram, etc.). The 2140-line implementation follows a predictable pattern: models → service → ViewModel (state machine + polling) → views (one per screen). No CLI support.
   - Severity: LOW (onboarding flows are too app-specific for generic scaffolding — each integration has unique states and APIs)
   - Suggested solution: N/A for scaffolding. The module structure (`module create WhatsAppOnboarding --type feature`) handles the package scaffolding. The business logic is inherently manual.

2. **No feature module with service scaffolding**
   - What's missing: The WhatsApp onboarding creates a complete feature with Service layer (API client), ViewModel, and Views — but `module create --type feature` only generates namespace/module/interface files, not service/viewmodel/view stubs. There's a gap between "empty module skeleton" and "feature with actual architecture".
   - Severity: MEDIUM (every feature follows the same Service + ViewModel + Views pattern, and developers have to create all these files manually every time)
   - Suggested solution: Consider `module create <Name> --type feature --with-service --with-viewmodel` or a separate `ios-app-manager feature scaffold <Name>` that generates Service.swift, ViewModel.swift, and a root View.swift stub within the module.

---

### e242227 Merge pull request #1 from thexflow/feature/CNTFRONT-451

Merge commit, see feature commit `b59f986` (WhatsApp onboarding) above.

---

### 1f372c8 no message

**What**: Backend URL changes — updates test and production environment URLs from `connect-back-test.herokuapp.com` / `connect-back-prod.herokuapp.com` to `xflow-exchanger-service-test.herokuapp.com` / `xflow-exchanger-service-prod.herokuapp.com`. Same change in `SharedCrmService` production URL. Follows the connect → xflow rebrand from batch 04.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — environment URL configuration change
- Manual edits to `Config.swift` and `SharedCrmService.swift`
- Notes: In a scaffolded project, environment URLs would be managed via `app-config setup` / `ApiConfigurator` — the config values would still be manually set, but the infrastructure for environment-based switching would already exist.

---

### 8b3c8e4 Merge remote-tracking branch origin/main

Merge commit, merges WhatsApp onboarding feature branch into working branch.

---

### 11ec5b1 no message

**What**: Another backend URL update — changes Heroku URLs to include auto-generated suffixes (e.g., `xflow-exchanger-service-test-1bc8eedafd28.herokuapp.com`). This is likely a Heroku app rename or new deployment.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — environment URL configuration change
- Manual edits to `Config.swift` and `SharedCrmService.swift`
- Notes: Same as `1f372c8` — URL values are project-specific, no scaffolding needed.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| 4ba948e Avatar LRU cache + deep links | PARTIAL | No deep linking scaffolding |
| 81e3812 Admin mode + voice permissions | YES | Custom app logic, manual (security concern: hardcoded password) |
| d3f4592 Bulk clear/evaluate context | YES | Feature code, manual |
| a3700d5 Parallel search + highlight fix | YES | Optimization + bug fix, manual |
| d7607ac Input area redesign | YES | UI polish, manual |
| b59f986 WhatsApp onboarding | PARTIAL | No feature+service scaffolding, 2140 lines manual |
| e242227 Merge #1 | SKIP | Merge commit |
| 1f372c8 URL changes | YES | Config change, manual |
| 8b3c8e4 Merge origin/main | SKIP | Merge commit |
| 11ec5b1 URL changes | YES | Config change, manual |

### Critical ALARMs (sorted by severity)

1. **MEDIUM: No deep linking infrastructure scaffolding** — URL scheme handling, `.onOpenURL`, route parsing, pending deep link during auth — ~50 lines of boilerplate every app needs. Especially important for apps with widgets/extensions (already flagged as HIGH in batch 03).

2. **MEDIUM: No feature module with service/viewmodel scaffolding** — `module create --type feature` generates only namespace + module files. Every feature also needs Service (API client), ViewModel, and root View — developers create these manually every time. WhatsApp onboarding is 2140 lines but follows a predictable Service → ViewModel → Views pattern.

3. **MEDIUM: Hardcoded password in source code** (81e3812) — Not a scaffolding gap but a security finding. `"service7"` plaintext in `ProfileView.swift`. Mobile app binaries are decompilable.

4. **LOW: No version bumping tooling** — In Tuist projects, version is single source of truth, so less of an issue than in raw xcodeproj.

### Observations

- **Batch 05 shows feature diversification** — no longer purely CRM. WhatsApp onboarding (2140 lines) is the first entirely new feature module. This validates the module architecture: a new feature module would be `module create WhatsAppOnboarding --type feature` + manual service/viewmodel/views.
- **Two different authors** appear in this batch. `1f372c8` and `11ec5b1` are by Mikhail Valiev (backend URL changes), the rest by Aleksandr Chechenev. The "no message" commits from Mikhail are just URL updates — likely coordinating backend deployments.
- **Live Activity maturation continues** — commit `4ba948e` fixes widget-specific gotchas (UIImage(contentsOfFile:) doesn't work in widget extensions, must use Data(contentsOf:)), adds LRU caching (batch 03-04 had single-avatar caching), and adds deep linking from Live Activity. The widget extension saga has spanned batches 03-05 with increasing sophistication.
- **Admin mode pattern** is interesting — hidden feature activation via logo tap count is a common iOS pattern for internal/debug tools. The hardcoded password is concerning for production, though.
- **Parallel search optimization** demonstrates a pattern where the UI framework matters: `async let` for parallel requests, with QUICK results shown immediately while MESSAGES results load. This kind of progressive loading is too app-specific for scaffolding but is a good architectural reference.
- **HighlightedText rewrite** (from AttributedString to Text concatenation) fixes a real bug with `AttributedString.index(_:offsetByCharacters:)` — the custom extension was fragile. The Text concatenation approach is simpler and correct. Good candidate for a shared UI utility.
