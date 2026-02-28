# Batch 08: Commits 71-80

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### 500300d XFL-25 Add Apple Sign-In with loading state on auth screen

**What**: Adds Apple Sign-In as a second authentication method alongside Telegram. New `AppleAuthService` (56 lines) sends Apple identity token to backend `/login/apple` endpoint and stores JWT tokens. New `AppleAuthViewModel` (82 lines) handles `ASAuthorization` result, extracts identity token + optional full name + email, manages loading/error state. `TelegramAuthView` gains a `SignInWithAppleButton` above the Telegram login button, with a loading state (ProgressView + "Authorizing..." text) while backend responds. Also bumps version from 0.1.6 to 0.1.9 and adds `LinkedAccounts` model (hasApple, hasTelegram) to `User`.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **No Apple Sign-In scaffolding or auth provider abstraction**
   - What's missing: Apple Sign-In requires: (a) `AuthenticationServices` framework import, (b) a service that posts the identity token to backend and stores JWT response, (c) a ViewModel handling `ASAuthorization` result with error/loading state, (d) `SignInWithAppleButton` integration in the auth screen, (e) the `Sign in with Apple` entitlement. None of this is scaffoldable via ios-app-manager. More broadly, there is no auth provider abstraction — each auth method (Telegram, Apple) is implemented as an independent service+ViewModel pair with parallel code paths.
   - Severity: MEDIUM (Apple Sign-In is increasingly required by Apple for App Store approval if any third-party login is offered. The entitlement, endpoint, service, and ViewModel pattern is repeatable across projects.)
   - Suggested solution: Consider an `auth-provider add apple` command that: (a) adds `Sign in with Apple` entitlement via `entitlements add`, (b) scaffolds `AppleAuthService` with identity token → backend JWT flow, (c) scaffolds `AppleAuthViewModel` with `ASAuthorization` handling, (d) generates the `SignInWithAppleButton` SwiftUI wrapper. A generic `auth-provider add <type>` could also support `telegram`, `google`, etc.

2. **No auth screen scaffolding**
   - What's missing: The auth/login screen (`TelegramAuthView`) is hand-built with no scaffolding. Adding a new auth provider means manually editing the view to insert another button. In a scaffolded project, the auth screen should be composable — each `auth-provider add` command would register its button in the auth view.
   - Severity: LOW (auth screen UI is highly app-specific; composability adds complexity)
   - Suggested solution: N/A — the auth screen layout is too design-specific for generic scaffolding. The service+ViewModel layer is more scaffoldable than the UI.

---

### c07337a Merge pull request #15 from thexflow/feature/XFL-25

Merge commit, see feature commit `500300d` (Apple Sign-In) above.

---

### 90936d6 XFL-54 Ignore widget deep links during onboarding

**What**: Adds a guard in `AppRouter.handleDeepLink()` that checks if onboarding is currently in the navigation path. If any `.onboarding` destination exists in `path`, the deep link to a contact is silently ignored with a log message. Prevents widget-triggered deep links from interrupting the onboarding flow (which could leave the user in a broken state with onboarding abandoned mid-flow).

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — navigation guard logic within AppRouter
- Manual edit to `AppRouter.swift` (8 lines added)
- Notes: This is a defensive navigation pattern — checking navigation stack state before allowing deep link handling. In a Relux-scaffolded project, this logic would live in a navigation Flow that guards state transitions. The pattern (check if specific destination is in nav stack → reject certain actions) is common but too context-specific for generic scaffolding.

---

### 623dd42 Merge pull request #13 from thexflow/feature/XFL-54

Merge commit, see feature commit `90936d6` (widget deep link guard) above.

---

### d2bbd0e XFL-59 Evaluation status on startup

**What**: Implements evaluation/trial expiry detection. On app startup and login, reads `isEvaluationDisabled` boolean from `GET /users/me` response. If `true`, shows a one-time global alert ("Evaluation finished — contact support@xflow.org") with a "Copy email" button. Persists the flag in `UserDefaultsManager` for diagnostics and displays it in Settings as a status badge. Also includes a `.spec/xfl-59-evaluation-status.md` spec document describing the backend contract and expected behavior. 75 lines across 9 files.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: **No spec-driven feature scaffolding.** This commit includes a `.spec/xfl-59-evaluation-status.md` file that describes the feature requirements and backend contract. This is the first time we see a spec document in the repo. The pattern of "spec file → implementation" could be supported by ios-app-manager — a command that reads a structured spec and scaffolds the boilerplate (model field, UserDefaults key, L10n entries, alert plumbing). However, this is very aspirational.
- Severity: LOW (the spec-to-code pipeline is too varied for generic scaffolding; each feature has unique requirements)
- Suggested solution: N/A for now. The spec file pattern (`.spec/` directory with feature specs) is good practice regardless of tooling. More practical: the repeated pattern of "global one-time alert with support email + copy button" (seen in both XFL-59 and XFL-60) could be a reusable component scaffolded by a utility.

---

### 079be77 XFL-59 Improve evaluation status display

**What**: Quick follow-up to `d2bbd0e` — replaces the raw `isEvaluationDisabled: true/false` text in Settings with a polished status badge. Extracts `evaluationStatusBadge` computed property: capsule-shaped badge with "Active" (green) or "Finished" (orange) text. Uses `Label("Evaluation", systemImage: "checkmark.seal")` as the row label. 16 lines added, 5 removed.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — UI polish for evaluation status display in Settings
- Manual edit to `SettingsView.swift` (extracted `evaluationStatusBadge` view property)
- Notes: A 6-minute follow-up commit (`19:39 → 19:45`) that turns a debug-style `true/false` label into a proper capsule badge. Good UX iteration — the initial commit prioritized functionality, the follow-up improved presentation.

---

### 47e1995 Merge pull request #16 from thexflow/feature/XFL-59

Merge commit, see feature commits `d2bbd0e` and `079be77` (evaluation status) above.

---

### a2d7c3e XFL-60 Handle suspended account

**What**: Implements global account suspension handling across the entire network layer. Key changes:

1. **Network layer detection**: New `throwIfAccountSuspended(statusCode:data:)` method added to all 4 request methods in `NetworkManager` (same duplication pattern flagged in batch 06). Detection logic: (a) explicit HTTP 423/451 status codes → suspended, (b) for non-2xx responses, scans JSON fields (`status`, `code`, `error`, `message`, `detail`) for "suspend" substring (case-insensitive), (c) fallback: raw text body contains "suspend".

2. **New error case**: `APIError.accountSuspended` added with status code mapping (423, 451 → `.accountSuspended`).

3. **Global notification**: On detection, posts `Notification.Name.accountSuspended` via NotificationCenter (main actor dispatch).

4. **App-level alert**: `XFlowApp` subscribes to `.accountSuspended` notification, shows one-time alert ("Account suspended — contact support@xflow.org") with "Copy email" and "Dismiss" buttons. `hasShownAccountSuspendedAlert` flag prevents spam.

5. **Spec document**: `.spec/xfl-60-account-suspended.md` (22 lines) describing the feature requirements.

119 lines added across 7 files.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **Network error interception pattern not scaffoldable**
   - What's missing: The `throwIfAccountSuspended` method is inserted into all 4 request methods in `NetworkManager`, adding yet another cross-cutting concern to the already duplicated request methods. In a well-structured network layer, this would be middleware/interceptor pattern. The `http-client setup` command doesn't scaffold any interceptor/middleware chain for request processing.
   - Severity: MEDIUM (every cross-cutting network concern — retry guard from batch 06, suspension detection here — requires touching 4 identical methods. An interceptor chain would make this N*1 instead of N*M.)
   - Suggested solution: Redesign `http-client setup` to generate a middleware/interceptor-based network layer where cross-cutting concerns (auth retry, suspension check, logging) are composed as interceptors rather than duplicated in each request method.

2. **Global alert pattern duplicated**
   - What's missing: The "show one-time global alert with support email + copy button" pattern is now used in both XFL-59 (evaluation finished) and XFL-60 (account suspended). Nearly identical code: `@State showAlert`, `@State hasShownAlert`, `supportEmail` constant, `.alert()` modifier with Copy/Dismiss buttons. No reusable component exists.
   - Severity: LOW (two occurrences so far, but the pattern is clear)
   - Suggested solution: Extract a reusable `GlobalSupportAlert` view modifier or component that handles one-time alert display with support email copy. Could be part of a `utilities` module.

---

### 3bef5d9 Merge pull request #14 from thexflow/feature/XFL-60

Merge commit, see feature commit `a2d7c3e` (suspended account) above.

---

### 9e1090d XFL-57 Add account deletion flow

**What**: Adds account self-deletion from Settings. Three changes:

1. **API**: New `deleteCurrentUser` endpoint (DELETE `/users/me`) in `APIEndpoint`. `UserService.deleteAccount()` calls `networkManager.deleteVoid()`.

2. **Settings UI**: New "Delete account" destructive button above the existing logout button. Shows confirmation alert ("This action is irreversible and will permanently delete your account.") with Cancel/Delete buttons. Loading state with ProgressView during deletion. Error alert if deletion fails. Both delete and logout buttons disabled during either operation.

3. **Post-deletion cleanup**: Identical to logout flow — unregisters push token (`PushNotificationService.onUserLogout()`), clears local tokens (`authService.logout()`), resets UserDefaults, navigates to auth screen.

76 lines across 3 files.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
1. **Duplicated cleanup logic between logout and account deletion**
   - What's missing: `performDeleteAccount()` (36 lines) and `performLogout()` (25 lines) in `SettingsView` share nearly identical cleanup code: unregister push → clear tokens → reset UserDefaults → navigate to auth. The only difference is that deletion calls `userService.deleteAccount()` first. This is the same "no centralized session clear" issue flagged in batch 06's e5cd20b (logout button). Now there are THREE places doing cleanup: MessengerPickerView logout, SettingsView logout, SettingsView delete account.
   - Severity: MEDIUM (three places with diverging cleanup logic is a maintenance hazard. If a new store is added — e.g., secure-store Keychain data — all three must be updated independently.)
   - Suggested solution: (Reiteration of batch 06 ALARM) Generate a centralized `SessionManager.clearAll()` or `AuthCoordinator.logout()` that each `setup` command registers cleanup steps with. `relux setup` could scaffold this as part of the app infrastructure. The deletion flow would call `sessionManager.clearAll()` after the API delete; the logout flow would just call `sessionManager.clearAll()`.

2. **Account deletion is not localized**
   - What's missing: "Delete account", "Deleting account...", "This action is irreversible...", and "Failed to delete account" are hardcoded English strings, unlike the rest of the app which uses `L10n` localization keys. The developer (Ivan Oparin) who implemented evaluation status (`d2bbd0e`) with proper L10n missed localization here.
   - Severity: LOW (localization gap, but easily fixed)
   - Suggested solution: Not a scaffolding issue per se, but a template could enforce localization by generating `L10n` entries for standard UI patterns.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| 500300d Apple Sign-In | PARTIAL | No auth provider scaffolding |
| c07337a Merge #15 | SKIP | Merge commit |
| 90936d6 Widget deep link guard | YES | Navigation guard, manual |
| 623dd42 Merge #13 | SKIP | Merge commit |
| d2bbd0e Evaluation status | PARTIAL | Global alert pattern not reusable |
| 079be77 Evaluation status UI polish | YES | UI refinement, manual |
| 47e1995 Merge #16 | SKIP | Merge commit |
| a2d7c3e Suspended account | PARTIAL | No network interceptor chain, duplicated alert pattern |
| 3bef5d9 Merge #14 | SKIP | Merge commit |
| 9e1090d Account deletion | PARTIAL | Duplicated cleanup logic (3 places now) |

### Critical ALARMs (sorted by severity)

1. **MEDIUM: No Apple Sign-In / auth provider scaffolding** — Apple Sign-In is a near-universal requirement for App Store apps with third-party login. The service + ViewModel + entitlement pattern is repeatable. An `auth-provider add apple` command could scaffold the boilerplate.

2. **MEDIUM: Network interceptor/middleware chain missing from http-client** — Cross-cutting network concerns (retry guard from batch 06, suspension detection here) are copy-pasted into 4 identical request methods. `http-client setup` should generate middleware-based architecture where each concern is a composable interceptor.

3. **MEDIUM: Session cleanup duplicated in 3 places** — Logout cleanup code (clear tokens, reset UserDefaults, unregister push, navigate to auth) now exists in: MessengerPickerView, SettingsView.performLogout(), SettingsView.performDeleteAccount(). A centralized `SessionManager.clearAll()` is needed (escalated from batch 06 LOW to MEDIUM given 3 occurrences).

4. **LOW: Global one-time alert pattern duplicated** — Evaluation finished alert (ContentView) and account suspended alert (XFlowApp) use nearly identical code. Could be a reusable view modifier.

5. **LOW: Account deletion strings not localized** — Hardcoded English strings in account deletion UI, inconsistent with the rest of the app's L10n usage.

### Observations

- **Batch 08 is feature-heavy with security/compliance focus.** Apple Sign-In, evaluation/trial management, account suspension, and account deletion are all trust & safety features. This suggests the app is approaching App Store submission or enterprise compliance requirements (Apple requires Sign in with Apple if any third-party login exists; GDPR requires account deletion).

- **Ivan Oparin (ivan@relux.works) is the primary contributor** — 7 of 10 commits. He handles: widget deep links, evaluation status (with spec), account suspension (with spec), and account deletion. Aleksandr Chechenev does Apple Sign-In. The `.spec/` directory pattern (seen in d2bbd0e and a2d7c3e) appears to be Ivan's practice — writing structured specs before implementation.

- **Version jumped from 0.1.6 to 0.1.9** (skipping 0.1.7, 0.1.8), matching the pattern from batch 06 where version jumped 0.1.2 → 0.1.5. Internal builds between captured commits.

- **The session cleanup problem is escalating.** What was a single occurrence in batch 06 (MessengerPickerView logout) is now three independent cleanup sites with subtly different implementations. The `SettingsView.performDeleteAccount` does push token unregister in a detached `Task` (fire-and-forget), while `performLogout` does it inline with `await`. This kind of divergence is exactly why a centralized `SessionManager` is needed.

- **NetworkManager duplication continues to compound.** The 4-method duplication problem (flagged in batches 05 and 06) gets worse: `throwIfAccountSuspended` is now added to all 4 methods (4 new insertion points). Combined with the retry guard from batch 06, each request method now has ~15 lines of cross-cutting boilerplate. This is the strongest argument yet for a middleware/interceptor refactor in `http-client setup`.

- **The `.spec/` directory is promising.** Two specs in this batch (`xfl-59-evaluation-status.md`, `xfl-60-account-suspended.md`) show a pattern of writing structured requirements before implementation. The specs are concise (15-22 lines), focused, and include backend contracts. This is good engineering practice, though the specs are not referenced by the code — they're purely for human readers.
