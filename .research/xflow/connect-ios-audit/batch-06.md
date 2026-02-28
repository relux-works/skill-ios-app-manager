# Batch 06: Commits 51-60

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### 3176bc4 Update ngrok URL and skip browser warning for image loading

**What**: Updates local development ngrok URL and adds `ngrok-skip-browser-warning` HTTP header to `CachedAsyncImage` request modifier so images load without ngrok's browser interstitial page.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — local development environment URL change
- Manual edits to `Config.swift` (ngrok URL) and `CachedAsyncImage.swift` (header)
- Notes: In a scaffolded project, environment URLs are managed via `app-config setup` / `ApiConfigurator`. The ngrok header is a dev-only concern; `CachedAsyncImage` likely uses Kingfisher's `AnyModifier`, which is app-specific image loading infrastructure.

---

### 2df2743 Fix voice record button: hit area, haptic feedback and response time

**What**: Three UX improvements to `VoiceRecordButton`:
1. **Hit area fix**: Adds `.contentShape(Circle())` so the tap target matches the visible button (not the frame).
2. **Haptic feedback**: Pre-warms `UIImpactFeedbackGenerator` on appear, fires haptic before recording starts. Defers `recorder.startRecording()` to next run loop (`DispatchQueue.main.async`) so haptic fires before `AVAudioSession` activation suppresses it. Enables `setAllowHapticsAndSystemSoundsDuringRecording(true)` on the audio session.
3. **Response time**: Speeds up scale animation (0.3 → 0.15 response, adds 0.7 damping fraction).

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — UI/UX bug fix within CRM module
- Manual edits to `VoiceRecordButton.swift` (14 lines added) and `AudioRecorderService.swift` (1 line added)
- Notes: The haptic-before-AVAudioSession pattern is a non-obvious iOS gotcha — `AVAudioSession.setActive(true)` briefly blocks the haptic engine. Deferring recording to next run loop is a clever workaround. `setAllowHapticsAndSystemSoundsDuringRecording(true)` is the proper API-level fix.

---

### a95f0a2 no message

**What**: Updates Telegram bot names in `Config.swift` — test environment bot changes from `sourcemap_connect_auth_test_bot` to `xflow_app_test_bot`, production bot changes from `connbot` to `xflow_app_bot`. Continues the connect → xflow rebrand from batch 04.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — environment configuration change
- Manual edit to `Config.swift` (2 lines changed)
- Notes: Bot names are app-specific configuration values, not scaffoldable. In a scaffolded project these would live in `ApiConfigurator` or similar config module.

---

### 73fffa9 Merge pull request #2 from thexflow/feature/XFL-28-new-bots

Merge commit, see feature commit `a95f0a2` (bot name changes) above.

---

### d800850 Telegram onboarding + universal messenger picker UI

**What**: Massive refactoring and feature addition — ~945 lines added, ~244 removed across 30 files. Transforms WhatsApp-only onboarding into a universal multi-messenger onboarding system supporting both WhatsApp and Telegram. Key changes:

1. **Architecture rename**: `WhatsAppOnboarding` → `Onboarding` throughout. `WhatsAppOnboardingService` deleted, replaced by `OnboardingService` (130 lines) with separate methods for WhatsApp and Telegram APIs. `WhatsAppOnboardingViewModel` → `OnboardingViewModel` with `messengerType` state.

2. **New `MessengerType` enum**: `whatsapp`/`telegram` — drives routing logic throughout onboarding.

3. **Messenger picker screen**: New `MessengerPickerView` (86 lines) — shows WhatsApp and Telegram buttons with icons. Displayed when no messenger is pre-selected.

4. **Telegram-specific screens**: New `CodeInputView` (138 lines — verification code input with monospaced font, auto-focus), `PasswordInputView` (122 lines — 2FA password input with secure field). New `OnboardingSessionStatus.awaitingPassword` state.

5. **Telegram API endpoints**: 6 new endpoints (`telegramOnboardingStart`, `telegramOnboardingStatus`, `telegramOnboardingSubmitCode`, `telegramOnboardingSubmitPassword`, `telegramOnboardingResendCode`, `telegramOnboardingCancel`).

6. **Telegram auth flow in ViewModel**: New methods `submitCode()`, `submitPassword()`, `resendCode()`. State machine extended with `.messengerPicker`, `.codeInput`, `.passwordInput` screens.

7. **Router updates**: `AppDestination.whatsAppOnboarding` → `.onboarding` with `initialMessengerType` parameter. `AppRouter` now checks both WhatsApp and Telegram statuses on startup/login via `getActiveOnboardingStatus()`. Active non-terminal onboarding takes priority even if CRM is available.

8. **NetworkManager retry fix**: Adds `isRetry` flag to prevent infinite 401 refresh loops. After refresh+retry still gets 401, sets `refreshDidNotHelp = true` to short-circuit future refresh attempts. Applied to all 4 request methods (`request`, `requestNoResponse`, `requestText`, `requestTextOptional`).

9. **Organization detail**: Adds "Connect WhatsApp" and "Connect Telegram" buttons to `OrganizationDetailView` (41 lines), with navigation to onboarding for the selected messenger.

10. **Version bump**: 0.1.2 → 0.1.5, new localization keys (14 across en/ru), Telegram icon assets.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No multi-service onboarding abstraction support**
   - What's missing: The refactoring from WhatsApp-only to WhatsApp+Telegram onboarding follows a predictable pattern: shared models with enum discriminator (`MessengerType`), unified service with per-messenger methods, unified ViewModel with branching logic, shared views + messenger-specific screens. This "add another service" pattern requires touching ~30 files. No CLI support for scaffolding multi-variant feature flows.
   - Severity: LOW (this is a one-time architectural refactoring, not a repeating pattern. The multi-messenger abstraction is app-specific domain logic.)
   - Suggested solution: N/A for scaffolding. The pattern is too domain-specific. However, the module rename aspect (WhatsAppOnboarding → Onboarding) could be handled by a `module rename` command (related to batch 04's MEDIUM ALARM about no rename tooling).

2. **No token refresh retry guard in NetworkManager scaffolding**
   - What's missing: The `isRetry` flag + `refreshDidNotHelp` guard is a critical networking pattern — prevents infinite 401 → refresh → retry → 401 loops. The `http-client setup` command generates HttpClient infrastructure but doesn't include this retry-guard pattern.
   - Severity: MEDIUM (infinite retry loops are a production reliability issue. Every app with token refresh needs this guard. The fix is ~50 lines across 4 request methods.)
   - Suggested solution: Include retry-guard logic in `http-client setup` generated code, or document it as a required pattern in the HttpClient module.

---

### ec83e0f Merge pull request #3 from thexflow/feature/XFL-16

Merge commit, see feature commit `d800850` (Telegram onboarding) above.

---

### ea4d4a4 XFL-39, XFL-40: placeholder and AI loaders for message input

**What**: Adds AI loading states to the CRM contact detail message input area. Two visual effects:

1. **Message input bubble loading state**: When `isLoading` is true, replaces the text field with a shimmer-animated "Drafting message..." placeholder. The `isLoading` flag is driven by `!viewModel.isAIContentReady && !viewModel.messageText.isEmpty` — shows loading when AI-generated message text is pre-loaded but artificial delay hasn't elapsed.

2. **AI operator request loading state**: When `isAIContentReady` is false, replaces the operator request text with shimmer "Analyzing entire context..." placeholder. When ready, reveals the actual operator request with opacity transition.

3. **Rotating placeholder loading**: Second input field shows "Preparing suggestions..." shimmer instead of rotating placeholder suggestions until AI content is ready.

4. **ViewModel**: New `isAIContentReady` published property, `scheduleAIContentReveal()` method — 3-second delay after contact load, then animated reveal. Task management (cancel on deinit, cancel+restart on next contact).

5. **Shimmer enhancement**: Increases shimmer gradient opacity from 0.3 to 0.6.

6. **Localization**: 4 new keys (en + ru): `message.placeholder`, `message.drafting`, `message.analyzingContext`, `message.preparingSuggestions`.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — CRM feature UI polish with timed AI content reveal
- Manual files:
  - `ContactDetailViewModel.swift` — `isAIContentReady`, `scheduleAIContentReveal()`, task lifecycle
  - `MessageInputBubble.swift` — `isLoading` parameter, conditional content rendering
  - `ContactDetailView.swift` — AI operator request loading/ready states, placeholder loading
  - `View+Shimmer.swift` — opacity adjustment
  - Localization: 4 new keys
- Notes: The 3-second artificial delay before revealing AI content is a deliberate UX choice — makes it look like the AI is "thinking" even though data is already loaded. The shimmer modifier is a reusable component from the shared extensions.

---

### 0990ec5 XFL-48

**What**: Two changes bundled:
1. **Back button for onboarding**: Adds `goBackToMessengerPicker()` method to `OnboardingViewModel` (resets messenger type, phone, country, error, screen). Adds back button UI to `PhoneInputView` — chevron.left + "Back" text, navigates to messenger picker. Reduces header top padding (60 → 20) to accommodate.
2. **Gitignore cleanup**: Removes `.idea/` directory (JetBrains IDE files) from version control. Expands `.gitignore` with proper patterns for xcuserdata, SPM (`.build/`, `.swiftpm/`), and IDE (`.idea/`).

**Scaffolding assessment**:
- [x] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — navigation fix + repo hygiene
- Manual edits:
  - `OnboardingViewModel.swift` — `goBackToMessengerPicker()` method (6 lines)
  - `PhoneInputView.swift` — back button UI (20 lines)
  - `.gitignore` — expanded patterns

**ALARM**:
- What's missing: **Incomplete `.gitignore` in `init` scaffolding.** The `.gitignore` additions here (`.build/`, `.swiftpm/`, `.idea/`, `xcuserdata/`) are standard patterns for any iOS/Swift project. `ios-app-manager init` generates a project, but the generated `.gitignore` may not include all these patterns.
- Severity: LOW (one-time fix, but every new project benefits from a comprehensive `.gitignore`)
- Suggested solution: Ensure `init` generates a comprehensive `.gitignore` covering: DerivedData, xcuserdata, .build, .swiftpm, .idea, .DS_Store, Pods (if applicable), screenshots, and common secrets patterns.

---

### 3f9c0e9 Merge pull request #4 from thexflow/feature/XFL-48-back-button

Merge commit, see feature commit `0990ec5` (back button + gitignore) above.

---

### e5cd20b XFL-48 logout button

**What**: Adds logout button to `MessengerPickerView` (20 lines). A "Logout" text button at the bottom of the messenger picker screen. Logout function: calls `TelegramAuthService().logout()` to clear Telegram tokens, calls `UserDefaultsManager.shared.resetAll()` to clear all user defaults, then navigates to auth screen via `router.navigateToAuth()`. Uses `@EnvironmentObject var router: AppRouter`.

**Scaffolding assessment**:
- [x] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — logout functionality within onboarding feature
- Manual edit to `MessengerPickerView.swift` (20 lines)

**ALARM**:
- What's missing: **No logout/session-clear scaffolding.** Logout in this app requires clearing: Telegram auth tokens (via `TelegramAuthService`), all user defaults (`UserDefaultsManager.resetAll()`), plus navigating to auth. In a scaffolded project with `secure-store` and `token-provider`, logout would also need to clear Keychain tokens. There's no centralized "logout" or "session clear" command/template.
- Severity: LOW (logout logic is app-specific — different apps clear different stores. But the pattern of "clear all credential stores + navigate to auth" is universal.)
- Suggested solution: Consider generating a `SessionManager` or `AuthCoordinator` during `init` or `relux setup` that provides a `logout()` method clearing all registered stores (Keychain, UserDefaults, token provider). Each `setup` command (secure-store, token-provider) would register its cleanup with the coordinator.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| 3176bc4 ngrok URL + header | YES | Dev config, manual |
| 2df2743 Voice record UX fixes | YES | UI/UX fix, manual |
| a95f0a2 Bot name updates | YES | Config change, manual |
| 73fffa9 Merge #2 | SKIP | Merge commit |
| d800850 Telegram onboarding | PARTIAL | No retry-guard in http-client, no module rename |
| ec83e0f Merge #3 | SKIP | Merge commit |
| ea4d4a4 AI loaders for input | YES | CRM feature UI, manual |
| 0990ec5 Back button + gitignore | PARTIAL | Incomplete gitignore in init |
| 3f9c0e9 Merge #4 | SKIP | Merge commit |
| e5cd20b Logout button | PARTIAL | No logout/session-clear pattern |

### Critical ALARMs (sorted by severity)

1. **MEDIUM: No token refresh retry guard in http-client scaffolding** — The `isRetry` + `refreshDidNotHelp` pattern prevents infinite 401 → refresh → retry → 401 loops. This is a production reliability concern. Every app with token-based auth needs this guard. The `http-client setup` command should include this pattern in generated code.

2. **LOW: Incomplete `.gitignore` in `init`** — `.build/`, `.swiftpm/`, `.idea/`, `xcuserdata/` are standard patterns missing from generated gitignore. One-time fix but affects every new project.

3. **LOW: No logout/session-clear scaffolding** — Clearing all credential stores (Keychain, UserDefaults, token provider) + navigating to auth is a universal pattern. Could be a generated `SessionManager` that each setup command registers cleanup with.

4. **LOW: No multi-service onboarding abstraction** — WhatsApp → WhatsApp+Telegram refactoring touched 30 files. Too domain-specific for scaffolding, but module rename (`module rename`) would help with the mechanical rename aspect.

### Observations

- **Batch 06 is dominated by onboarding generalization.** The WhatsApp-only onboarding from batch 05 is now a multi-messenger system supporting WhatsApp + Telegram. This is the largest single-commit change in this batch (d800850: 945+ lines, 30 files). The architectural approach is clean: shared models with enum discriminator, unified service with per-messenger methods, shared views + messenger-specific screens.

- **Telegram auth is significantly different from WhatsApp auth.** WhatsApp uses QR/pairing codes (passive — user enters code on their phone). Telegram uses active code submission (code sent to Telegram app, user types it into xflow) with optional 2FA password. This required new screens (`CodeInputView`, `PasswordInputView`) and new ViewModel states. The shared onboarding infrastructure handles both flows through the `messengerType` discriminator.

- **NetworkManager retry guard is a critical reliability fix.** The `refreshDidNotHelp` flag prevents a specific failure mode: if the refresh token itself is invalid/expired, the app would loop: request fails 401 → refresh succeeds (server returns new access token, but it's also invalid) → retry fails 401 → refresh again → retry again → forever. This is applied to all 4 request methods (55 lines of changes across `request`, `requestNoResponse`, `requestText`, `requestTextOptional`), highlighting the earlier observation (batch 05) that the network layer has duplicated code that should be consolidated.

- **Two developers working in parallel** — Aleksandr Chechenev builds the main Telegram onboarding feature (d800850), while Mikhail Valiev does the infrastructure: bot name updates (a95f0a2), back button + gitignore (0990ec5), and logout (e5cd20b). The feature branch merges (#2, #3, #4) show this parallel workflow.

- **The 3-second AI content delay is pure UX theater** (ea4d4a4). The data is already loaded — the delay creates the perception of AI "thinking". This is a common pattern in AI-powered products where instant responses feel "too easy" and users trust a brief loading state more. Shimmer animations reinforce the effect.

- **Version jumped from 0.1.2 to 0.1.5** (skipping 0.1.3 and 0.1.4), suggesting internal builds or other changes not captured in this commit history.
