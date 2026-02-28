# Batch 09: Commits 81-90

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### e0a5dfe XFL-57 Await push unregister before logout

**What**: Fixes the account deletion cleanup flow in `SettingsView.performDeleteAccount()` — replaces the fire-and-forget detached `Task { await PushNotificationService.shared.onUserLogout() }` with a direct `await` call, ensuring push token unregistration completes before tokens are cleared locally. This was the divergence flagged in batch 08 (commit `9e1090d`): `performDeleteAccount` used fire-and-forget while `performLogout` awaited properly.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — bug fix for async ordering in cleanup flow
- Manual edit to `SettingsView.swift` (2 lines added, 4 removed)
- Notes: This fix directly addresses the cleanup divergence flagged in batch 08. The detached `Task` meant push unregistration could race with token clearing — if tokens are wiped before the push unregister request fires, the server can't identify which device to unregister. Now both `performDeleteAccount` and `performLogout` await push unregistration. The underlying ALARM still stands: three independent cleanup sites need a centralized `SessionManager.clearAll()`.

---

### 62dc80f Merge pull request #10 from thexflow/feature/XFL-57

Merge commit, see feature commit `e0a5dfe` (push unregister fix) above.

---

### 475bbc0 XFL-55 Remove financial network references

**What**: Removes the "Invitation-only financial network" subtitle from the login/auth screen (`TelegramAuthView`). The 5-line `Text(L10n.Auth.invitationOnly)` block is deleted from the view. Localization strings in both en/ru are emptied (set to `""`) rather than deleted — the key `auth.invitationOnly` still exists in `.strings` files with empty values.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — product rebranding (removing "financial network" positioning)
- Manual edits:
  - `TelegramAuthView.swift` — remove 5-line subtitle block
  - `en.lproj/Localizable.strings` — empty `auth.invitationOnly` value
  - `ru.lproj/Localizable.strings` — empty `auth.invitationOnly` value
- Notes: The localization keys are emptied rather than deleted — a half-cleanup. The follow-up commit `0383991` finishes the job by removing the `L10n.Auth.invitationOnly` accessor entirely. This is a product pivot: the app is no longer positioning itself as a "financial network." Combined with the evaluation/trial features from batch 08, this suggests a shift from invite-only finance app to a more general-purpose platform.

---

### 0383991 XFL-55 Remove invitation-only label

**What**: Completes the cleanup from `475bbc0` — removes the `L10n.Auth.invitationOnly` static accessor from the `L10n` enum and cleans up a trailing blank line in both `.strings` files. Now the `auth.invitationOnly` key is fully dead code (strings files still have the empty key, but nothing references it).

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — code cleanup completing the removal from `475bbc0`
- Manual edits:
  - `L10n.swift` — remove `invitationOnly` accessor (1 line)
  - `en.lproj/Localizable.strings` — remove trailing blank line
  - `ru.lproj/Localizable.strings` — remove trailing blank line
- Notes: Should have been part of `475bbc0` — two commits for what's effectively a single "remove feature label" change. The empty localization keys (`auth.invitationOnly = ""`) remain in `.strings` files as dead weight, though this is harmless.

---

### 278aa2f Merge pull request #9 from thexflow/feature/XFL-55

Merge commit, see feature commits `475bbc0` and `0383991` (financial network removal) above.

---

### 7d597eb XFL-58 Add legal links on login screen

**What**: Adds a "Privacy Policy" and "Terms of Use" link footer to the login screen. New `legalLinks` computed property in `TelegramAuthView` renders an `HStack` with two `Link` views separated by a `•` bullet, pointing to `https://xflow.org/privacy-policy` and `https://xflow.org/terms-of-use`. Styled in `.footnote` gray with `.tint(.gray)`. Added below the login buttons with 16pt top padding.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: **No legal/compliance links scaffolding in `init`.** Privacy Policy and Terms of Use links on the login screen are an App Store requirement for most apps. The URLs are hardcoded strings, not configurable. In a scaffolded project, these URLs should come from the project config (e.g., `ios-app-manager.json` could have a `legal` section with `privacy_policy_url` and `terms_url`), and the `init` command could scaffold the legal links view component.
- Severity: LOW (one-time setup, ~23 lines, but required for virtually every app going to the App Store)
- Suggested solution: Add optional `legal.privacy_policy_url` and `legal.terms_url` fields to the project config. If present, `init` scaffolds a reusable `LegalLinksView` component that reads URLs from config-injected constants. This keeps URLs configurable without code changes when deploying to different environments or white-label apps.

---

### 58d23d1 XFL-58 Update terms URL

**What**: Quick follow-up to `7d597eb` — changes the Terms of Use URL from `https://xflow.org/terms-of-use` to `https://xflow.org/terms`. A 30-minute follow-up (17:33 → 18:02) suggesting the first URL was either wrong or the route was renamed on the website.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — URL correction
- Manual edit to `TelegramAuthView.swift` (1 line changed)
- Notes: This commit reinforces the ALARM from `7d597eb` — hardcoded URLs require code changes when they change. If the URL came from a config or remote config, this would be a config update, not a code change.

---

### 6e4877a Merge pull request #11 from thexflow/feature/XFL-58

Merge commit, see feature commits `7d597eb` and `58d23d1` (legal links) above.

---

### 4971b79 XFL-56 Add login/password auth flow

**What**: Adds username/password authentication as a third login method (alongside Telegram and Apple Sign-In). Four changes across 4 files (142 lines added):

1. **API endpoint**: New `.login` case in `APIEndpoint` → `POST /login`.

2. **PasswordAuthService** (45 lines, NEW file): `@MainActor` class with injected `NetworkManager`, `KeychainManager`, `UserDefaultsManager`. `login(login:password:)` method posts `LoginPasswordRequest` to `.login`, saves JWT tokens to Keychain, stores `userId` in UserDefaults, triggers push token registration via `PushNotificationService.shared.onUserLogin()`. Near-identical structure to `AppleAuthService` from batch 08.

3. **LoginPasswordRequest model** (14 lines, NEW file): Simple `Codable` struct with `login` and `password` fields.

4. **TelegramAuthView expansion** (81 lines): New `@State` vars (`loginInput`, `passwordInput`, `isPasswordLoggingIn`, `passwordLoginError`). New `passwordLoginSection` computed property renders two text fields (Login + Password) with labels, a Login button with loading state, all in a `.ultraThinMaterial` rounded card. `loginWithPassword()` function calls `passwordAuthService.login()`, navigates to main app on success, shows error alert on failure. The password error is chained into the existing error alert (`passwordLoginError ?? viewModel.errorMessage ?? appleViewModel.errorMessage`). UI strings ("Login", "Password") are hardcoded English — not using L10n.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **Auth provider proliferation without abstraction**
   - What's missing: This is now the THIRD independent auth service: `TelegramAuthService`, `AppleAuthService` (batch 08), `PasswordAuthService`. Each follows the same pattern: (a) call backend endpoint, (b) save JWT tokens to Keychain, (c) save userId to UserDefaults, (d) trigger push registration. Steps b-d are identical across all three — pure code duplication. There is no auth provider protocol or shared post-login handler.
   - Severity: MEDIUM (three auth services with duplicated post-login logic. Adding a fourth provider — Google, Facebook — means copying the same 10 lines again. If a new post-login step is needed — e.g., clearing secure-store on fresh login — all three services must be updated independently.)
   - Suggested solution: (Extension of batch 08 ALARM) The `auth-provider add <type>` command should scaffold services that conform to an `AuthProvider` protocol and delegate post-login steps (token save, userId save, push registration) to a shared `AuthCoordinator.completeLogin(tokens:userId:)`. This ties into the `SessionManager.clearAll()` need from batch 08 — login and logout should be symmetric operations managed by a single coordinator.

2. **Auth screen is a monolithic view growing unbounded**
   - What's missing: `TelegramAuthView` now contains: Telegram login, Apple Sign-In button, password login form, legal links, environment badge, debug login, GIF loading — all in a single file. Each new auth method adds ~80 lines directly to this view. There's no composition pattern — it's a vertical stack of everything.
   - Severity: LOW (the file is getting large but still manageable. However, it's named `TelegramAuthView` while handling Apple and password login too — naming no longer reflects content.)
   - Suggested solution: Not a scaffolding issue per se, but the view should be renamed to `AuthView` or `LoginView`, and each auth method's UI should be extracted into subviews. Scaffolding could help by generating each auth method's UI as a separate view component.

3. **Hardcoded English strings in password login UI**
   - What's missing: "Login", "Password", and the "Login" button label are hardcoded English strings, not using `L10n` localization keys. Inconsistent with the rest of the app's localization practice (flagged for account deletion in batch 08).
   - Severity: LOW (localization gap, same pattern as batch 08)
   - Suggested solution: Repeated observation — template-generated UI should include L10n keys by default.

---

### 5d74a4c XFL-56 Fix password login threading

**What**: Adds `@MainActor` annotation to the `Task` closure in `loginWithPassword()`. Without it, the `defer { isPasswordLoggingIn = false }` and error state updates could execute off the main thread, causing potential UI update issues (SwiftUI state mutations must happen on main actor). The `PasswordAuthService` is already `@MainActor`, but the `Task` closure inherits the calling context's actor isolation only if explicitly annotated.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — threading bug fix in auth flow
- Manual edit to `TelegramAuthView.swift` (1 line changed: `Task {` → `Task { @MainActor in`)
- Notes: A same-day follow-up fix (both commits at 19:07, same timestamp as `e0a5dfe`). This is a common Swift concurrency pitfall — `Task { }` closures don't inherit `@MainActor` from the enclosing view's body, so state mutations inside them need explicit `@MainActor` annotation. In a scaffolded project, templates should always use `Task { @MainActor in` for UI state mutations.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| e0a5dfe Await push unregister | YES | Cleanup ordering fix, manual |
| 62dc80f Merge #10 | SKIP | Merge commit |
| 475bbc0 Remove financial refs | YES | Product rebranding, manual |
| 0383991 Remove invitation label | YES | Code cleanup, manual |
| 278aa2f Merge #9 | SKIP | Merge commit |
| 7d597eb Legal links on login | PARTIAL | No legal links config/scaffolding |
| 58d23d1 Update terms URL | YES | URL fix, reinforces config need |
| 6e4877a Merge #11 | SKIP | Merge commit |
| 4971b79 Login/password auth | PARTIAL | No auth provider abstraction, duplicated post-login logic |
| 5d74a4c Fix password threading | YES | Threading fix, manual |

### Critical ALARMs (sorted by severity)

1. **MEDIUM: Auth provider proliferation without abstraction** — Three independent auth services (Telegram, Apple, Password) with duplicated post-login logic (save tokens, save userId, register push). Adding a fourth provider means copying 10 identical lines. An `AuthProvider` protocol + shared `AuthCoordinator.completeLogin()` is needed. This is the auth-side mirror of the `SessionManager.clearAll()` problem for logout (batch 08). Login and logout should be symmetric operations managed by a single coordinator.

2. **LOW: No legal links configuration in project config** — Privacy Policy and Terms of Use URLs are hardcoded. These are App Store requirements and should be configurable via `ios-app-manager.json` (a `legal` section). The URL change in `58d23d1` proves hardcoded URLs require code changes.

3. **LOW: Auth screen monolith** — `TelegramAuthView` now handles three auth methods + legal links + debug login + environment badge. Named after one auth method while containing all three. Growing unbounded with each new provider.

4. **LOW: Hardcoded English strings in password login** — Third occurrence of missing L10n usage (after evaluation status strings in batch 08 and account deletion strings in batch 08). Pattern suggests L10n discipline breaks down for quickly-added features.

### Observations

- **All 7 non-merge commits are by Ivan Oparin (ivan@relux.works).** This is a solo day of work — all timestamps are Feb 23, 2026, spanning 17:22 to 19:07 (under 2 hours). Three separate features (XFL-55 rebranding, XFL-58 legal links, XFL-56 password login) plus a fix from XFL-57 (push unregister ordering from batch 08).

- **Product pivot is becoming clearer.** The removal of "Invitation-only financial network" branding (XFL-55) combined with adding generic login/password auth (XFL-56) and legal links (XFL-58) signals a shift from an exclusive, Telegram-gated financial network to a more conventional app with multiple auth methods and proper legal compliance. The app is preparing for broader distribution.

- **Auth method count has tripled since batch 08.** In batch 06, there was only Telegram. Batch 08 added Apple Sign-In. Now batch 09 adds password login. Three auth methods, three independent services, zero shared infrastructure. The post-login ceremony (save tokens → save userId → register push) is duplicated verbatim across all three. This is the strongest argument yet for an `auth-provider` scaffolding command.

- **The push unregister fix (`e0a5dfe`) validates batch 08's ALARM.** The cleanup divergence between `performDeleteAccount` (fire-and-forget) and `performLogout` (awaited) was flagged in batch 08's analysis of commit `9e1090d`. This fix proves the divergence was indeed a bug, not a deliberate design choice. A centralized `SessionManager` would have prevented this class of bug entirely.

- **Same-timestamp commits suggest rebasing or batch work.** `e0a5dfe` and `5d74a4c` have identical timestamps (Mon Feb 23 19:07:15 2026 +0400) but are separate commits touching different features (XFL-57 vs XFL-56). This could indicate: (a) commits were rebased/amended to the same time, (b) both fixes were made simultaneously and committed in rapid succession, or (c) tooling artifacts.

- **The `TelegramAuthView` naming problem is telling.** The file started as a Telegram-specific auth view and has accumulated Apple Sign-In (batch 08), password login (this batch), and legal links (this batch). It's now the app's primary auth screen, but the name still says "Telegram." In a scaffolded project with proper module boundaries, auth methods would be separate components composed in a generic `AuthView`. This organic growth pattern — one file accumulating unrelated responsibilities — is exactly what scaffolding is designed to prevent.

- **Template `@MainActor` pattern for UI Tasks.** The threading fix in `5d74a4c` (`Task {` → `Task { @MainActor in`) is a common Swift concurrency mistake. ios-app-manager templates that generate `Task` closures modifying `@State`/`@Published` should always include `@MainActor` annotation. Worth adding to the template conventions.
