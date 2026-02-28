# Batch 01: Commits 1-10

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### 4db7b36 Initial Commit

**What**: Xcode-generated template project — vanilla `.xcodeproj` with ContentView, App struct, unit/UI test targets. No dependencies, no architecture.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: connect-ios uses a raw Xcode project (`.xcodeproj`), not Tuist. `ios-app-manager init` generates Tuist manifests (Project.swift, Workspace.swift, Package.swift, Tuist.swift). This is the *intended* replacement — the whole point of the migration is to move from raw xcodeproj to Tuist-managed manifests. Not a blocker, just a fundamental difference to note.
- Severity: LOW (this is by design — we're replacing, not replicating)
- Suggested solution: None needed. `ios-app-manager init` with a proper config replaces this entire commit.

**INSTRUCTION**:
- `ios-app-manager init` — scaffolds the Tuist-based project structure (manifests, App.swift, Info.plist, entitlements, xcconfigs)
- No manual files needed for this commit specifically

---

### 30c2f02 Initial iOS app implementation

**What**: Massive initial implementation (~17K lines). Sets up the entire app: routing, networking (NetworkManager with auth, JWT tokens), storage (Keychain, UserDefaults), localization (L10n), 3 feature modules (Auth/Telegram, CRM with chat/contacts/audio/voice, Organizations, Profile), data models (User, CrmContact, CrmMessage, Organization), UI components (CachedAsyncImage, MessengerIcons, SocialIcons), utilities (DateFormatters, HapticManager, Logger, ToastManager), and unit tests with mocks.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM (partially scaffoldable, many gaps)

**INSTRUCTION** (what ios-app-manager handles):
- `ios-app-manager init` — project skeleton
- `ios-app-manager ioc setup` — IoC container (Registry.swift)
- `ios-app-manager relux setup` — Relux infrastructure
- `ios-app-manager secure-store setup --access-group group.org.xflow.app` — Keychain wrapper (replaces custom KeychainManager)
- `ios-app-manager token-provider setup` — Token storage/refresh (replaces manual JWT handling in NetworkManager)
- `ios-app-manager utilities setup` — shared helpers module
- `ios-app-manager http-client setup` — HTTP client IoC registration
- `ios-app-manager app-config setup` — app config + API configurator
- `ios-app-manager module create Auth --type relux-feature` — Auth module scaffolding
- `ios-app-manager module create CRM --type relux-feature` — CRM module scaffolding
- `ios-app-manager module create Organizations --type relux-feature` — Organizations module scaffolding
- `ios-app-manager module create Profile --type relux-feature` — Profile module scaffolding
- `ios-app-manager dep add Auth --depends-on SecureStore` (etc. for cross-module deps)

Manual files per module (example for CRM):
```
Packages/CRM/Sources/CRM/
    Business/CRM.Business+Action.swift     ← fill with actual CRM actions
    Business/CRM.Business+Effect.swift     ← fill with CRM effects (API calls)
    UI/Views/ContactDetailView.swift       ← all custom UI (~1200 lines)
    UI/Views/ContactListView.swift
    UI/Views/ContactSearchView.swift
    UI/Views/MotivationsPanelView.swift
    UI/Components/AudioMessageView.swift
    UI/Components/CrmContactCard.swift
    UI/Components/MessageBubble.swift
    UI/Components/MessageInputBubble.swift
    UI/Components/VoiceRecordButton.swift
    UI/Components/LeadEmoji.swift
    Data/Api/CrmService.swift              ← networking (replaces monolithic CrmService)

Packages/CRMImpl/Sources/CRMImpl/
    Business/CRM.Business+State.swift      ← fill with CRM state
    Business/CRM.Business+Flow.swift       ← fill with CRM reducer
    Services/AudioPlayerService.swift
    Services/AudioRecorderService.swift
```

**ALARM** (multiple gaps):

1. **No app router scaffolding**
   - What's missing: `AppRouter` with `NavigationPath`, root destination switching (auth vs main), sheet presentation — this is core app infrastructure with no CLI support
   - Severity: MEDIUM (one-time manual setup, but it's a pattern every app needs)
   - Suggested solution: Consider `ios-app-manager router setup` that generates a basic AppRouter with Relux integration (navigation as state)

2. **No networking layer scaffolding beyond IoC registration**
   - What's missing: `http-client setup` only registers HttpClient in IoC. The actual `NetworkManager` here is 700+ lines with: generic request methods (GET/POST/PUT/DELETE), auth header injection, token refresh, error handling, multipart upload, response parsing. None of this is generated.
   - Severity: HIGH (every feature depends on networking; without it, the scaffolded project can't talk to a backend)
   - Suggested solution: `http-client setup` should generate a proper networking layer template, or at minimum stub files in `Packages/HttpClient/` with request/response patterns

3. **No data model scaffolding**
   - What's missing: Models (User, CrmContact, CrmMessage, MessageAttachment, Organization, TelegramSession) are all manually created. No CLI support for generating Codable model files or DTOs.
   - Severity: MEDIUM (models are project-specific, but the namespace hierarchy `Data.Api.DTO` generated by relux templates expects models to live there)
   - Suggested solution: Consider `ios-app-manager model create <Name> --module <Module>` that generates a Codable struct in the correct `Data/Api/DTO/` path

4. **No localization scaffolding**
   - What's missing: L10n.swift (334 lines of localization keys), Localizable.strings files for en/ru. No CLI support.
   - Severity: MEDIUM (workaround: create manually, but file placement in Tuist packages differs from monolithic app)
   - Suggested solution: `ios-app-manager l10n setup` that creates L10n infrastructure and string files in the right Tuist package paths

5. **No test/mock scaffolding**
   - What's missing: Unit tests (ContactDetailViewModelTests, ContactListViewModelTests, ProfileViewModelTests) and mocks (MockCrmService, MockOrganizationService, MockUserService). Known deferred item.
   - Severity: MEDIUM (deferred to future epic)
   - Suggested solution: Already tracked in backlog

6. **No UI component library scaffolding**
   - What's missing: Reusable components (CachedAsyncImage, MessengerIcons, SocialIcons, ImagePicker, ClearBackgroundView) need to live in a shared UI module but there's no scaffolding for component files within modules
   - Severity: LOW (just manual file creation in the right package)
   - Suggested solution: N/A — `module create Components --type ui` creates the module, files go inside manually

7. **No UserDefaults manager scaffolding**
   - What's missing: `secure-store setup` handles Keychain, but UserDefaultsManager (92 lines, stores current user ID, experimental mode flag, etc.) has no equivalent
   - Severity: LOW (small file, manual creation is fine)
   - Suggested solution: N/A

---

### 8b70fc6 Fix: Add ngrok header only for local development

**What**: Conditionalizes the `ngrok-skip-browser-warning` header in NetworkManager to only be sent when `AppConfig.environment == .local`. Fixes 4 places in NetworkManager where the header was hardcoded.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- `ios-app-manager app-config setup` — generates AppConfig with environment management
- Manual: The actual NetworkManager code that uses `AppConfig.environment` is manual. This is a bug fix in custom networking code — no scaffolding needed.
- Notes: `app-config setup` provides the `AppConfig` + `ApiConfigurator` pattern that supports environment-based configuration. The networking layer would use `ApiConfigurator` for base URL / header management rather than hardcoded checks.

---

### f09f89a Redesign contact detail page and fix bugs

**What**: Major UI redesign of ContactDetailView — moves contact info (avatar, name, phone) from inline header into navigation toolbar, adds user profile avatar in trailing toolbar position, adds LinkParser utility for clickable URLs in text, removes CrmSettingsView sheet.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — this is purely custom UI work within an existing feature module
- Manual files:
  - `Packages/CRM/Sources/CRM/` — already scaffolded, this is editing existing views
  - `Packages/Utilities/Sources/Utilities/LinkParser.swift` — new utility, goes in shared utilities module (created by `utilities setup`)
- Notes: The link parsing utility (NSDataDetector + AttributedString caching) is a reusable pattern that could live in the Utilities module.

---

### a0b48e3 Add .gitignore and audio player logging

**What**: Adds project `.gitignore` (DerivedData, xcuserstate, screenshots, .DS_Store), adds debug logging statements to AudioPlayerService, removes accidentally committed screenshot.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: `ios-app-manager init` does not generate a `.gitignore` file. The generated Tuist project needs one that covers DerivedData, .build, Tuist caches, xcuserstate, etc.
- Severity: MEDIUM (easy to create manually, but every project needs it and it's an obvious init-time artifact)
- Suggested solution: `ios-app-manager init` should generate a `.gitignore` with standard Xcode/Tuist ignores

**INSTRUCTION**:
- Logging additions are manual code in the CRM module — no scaffolding needed

---

### 5b6cc30 Implement audio player with local file caching and error handling

**What**: Adds `AudioCacheManager` — file-based audio caching system with 7-day TTL, download state tracking via actor, disk storage in Caches directory. Refactors `AudioPlayerService` to use cached files. Updates `AudioMessageView` UI.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands needed — this is feature-level code within the CRM module
- Manual files:
  - `Packages/CRMImpl/Sources/CRMImpl/Services/AudioCacheManager.swift` — media cache utility specific to CRM (or could be shared if video cache follows same pattern)
  - Edits to existing `AudioPlayerService.swift` and `AudioMessageView.swift`
- Notes: The caching pattern (download + disk cache + TTL cleanup) is repeated for video later (commit `e4cbcef`). Could be a generic `MediaCacheManager<T>` in a shared module, but that's an architectural decision, not a scaffolding gap.

---

### d07129b Kingfisher with manual retry and 30s timeout

**What**: Replaces custom `ImageCacheManager` (200+ lines of manual NSCache/NSLock/host-blocking) with Kingfisher library. Refactors `CachedAsyncImage` to wrap Kingfisher with memory cache sync check, manual retry button, 30s timeout. Adds `KingfisherConfig.setup()` in app entry point.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- `ios-app-manager dep add-external --url https://github.com/onevcat/Kingfisher.git --version 8.6.2 --module <AppTarget>` — adds Kingfisher as external dependency
- Manual files:
  - `Packages/Components/Sources/Components/CachedAsyncImage.swift` — Kingfisher wrapper component (goes in shared UI module)
  - `Packages/Components/Sources/Components/KingfisherConfig.swift` — configuration setup
- Notes: App entry point needs `KingfisherConfig.setup()` call in `init()`.

**ALARM**:
- What's missing: No support for app-level initialization hooks. When adding libraries that need setup (Kingfisher, Firebase, analytics), there's no way to scaffold or manage the "init sequence" in App.swift.
- Severity: LOW (just add a line to App.swift init, but the pattern is common)
- Suggested solution: N/A — too project-specific. Document as a manual step.

---

### 9bec457 Implement automatic navigation to next CRM contact on app startup

**What**: Modifies `AppRouter` startup flow to check CRM access and auto-navigate to next contact. Adds `isCrmAvailable` field to `User` model. Also includes Kingfisher SPM dependency in pbxproj (was missing from previous commit's xcodeproj).

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — this is custom app router logic and model field addition
- Manual edits to:
  - `AppRouter.swift` — startup navigation logic
  - `Models/User.swift` — new `isCrmAvailable` property
- Notes: In Tuist-based project, the SPM dependency (Kingfisher) would be in Package.swift, managed by `dep add-external`, not in pbxproj. So the pbxproj change is irrelevant for our scaffolding target.

---

### e4cbcef Add in-app video playback with caching

**What**: Adds `VideoCacheManager` (copy-paste of AudioCacheManager adapted for video — .mp4, 30s timeout, same TTL/cleanup pattern), `VideoMessageView` (thumbnail + play button + fullscreen AVPlayerViewController), `VideoPlayerHelper` utility. Updates `MessageBubble` to render video messages.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — feature code within CRM module
- Manual files:
  - `Packages/CRMImpl/Sources/CRMImpl/Services/VideoCacheManager.swift`
  - `Packages/CRM/Sources/CRM/UI/Components/VideoMessageView.swift`
  - `Packages/CRMImpl/Sources/CRMImpl/Helpers/VideoPlayerHelper.swift`
- Notes: `VideoPlayerHelper` wraps AVKit. Both Audio and Video cache managers share 90%+ identical code.

**ALARM**:
- What's missing: No media module scaffolding. Audio/video caching, playback, recording are common mobile patterns. The connect-ios codebase already has AudioCacheManager, VideoCacheManager, AudioPlayerService, AudioRecorderService — all of which are duplicated logic.
- Severity: LOW (this is domain code, not scaffolding; but a `module create Media --type kit` with suggested file structure would help)
- Suggested solution: N/A for scaffolding. Architectural recommendation: extract `MediaCacheManager` generic into a shared kit module.

---

### 9aa7644 Fix refresh button to trigger chat analysis

**What**: One-line fix — changes the refresh button in `MotivationsPanelView` toolbar from calling `viewModel.loadContact()` to `viewModel.reanalyzeChat()`, so the button actually triggers AI chat analysis instead of just reloading contact data.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — this is a one-line bug fix in custom UI code
- Manual edit to `MotivationsPanelView.swift`

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| 4db7b36 Initial Commit | YES | `init` replaces this entirely |
| 30c2f02 Initial Implementation | PARTIAL | Networking, models, localization, router, tests — all manual |
| 8b70fc6 Ngrok fix | YES | Manual code fix, `app-config` provides environment support |
| f09f89a Contact redesign | YES | Pure UI work, utilities module for LinkParser |
| a0b48e3 Gitignore + logging | PARTIAL | No .gitignore generation in `init` |
| 5b6cc30 Audio caching | YES | Feature code, manual files in CRM module |
| d07129b Kingfisher | PARTIAL | `dep add-external` adds package, but CachedAsyncImage is manual |
| 9bec457 Auto navigation | YES | Manual router/model code |
| e4cbcef Video playback | YES | Feature code, manual files in CRM module |
| 9aa7644 Refresh fix | YES | One-line manual fix |

### Critical ALARMs (sorted by severity)

1. **HIGH: No networking layer scaffolding** — `http-client setup` only does IoC registration. The actual HTTP client (auth, token refresh, multipart upload, error handling) is 700+ lines of manual code that every app needs.

2. **MEDIUM: No app router scaffolding** — Navigation architecture (NavigationPath, root switching, deep linking) is foundational but not generated.

3. **MEDIUM: No data model scaffolding** — Models/DTOs must be created manually with no file-placement guidance for Tuist package structure.

4. **MEDIUM: No localization scaffolding** — L10n + string files need manual setup with correct Tuist resource paths.

5. **MEDIUM: .gitignore not generated** — `init` should produce a project .gitignore.

6. **MEDIUM: No test/mock scaffolding** — Already known and deferred.
