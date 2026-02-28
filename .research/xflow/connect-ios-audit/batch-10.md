# Batch 10: Commits 91-102

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### ff2257c XFL-56 Update password login UI

**What**: Major refactoring of password login UI — replaces inline text fields (login + password in a material card) with a native `UIAlertController`-based system prompt. New `LoginPasswordSystemAlert` (~120 lines) — a `UIViewControllerRepresentable` wrapping `UIAlertController` with two text fields (login + password), auto-focus, submit button enabled only when both fields are non-empty. The inline `passwordLoginSection` is replaced by a single "Log In with password" button (lock.fill icon) that triggers the system alert. Adds `isAuthInProgress` computed property to centralize mutual exclusion across all three auth methods (Telegram, Apple, Password). Clears `passwordLoginError` before each auth attempt. Apple Sign-In's `onCompletion` now also clears `passwordLoginError`.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — auth UI refactoring within existing view
- Manual edits to `TelegramAuthView.swift` (197 added, 74 removed — net +123 lines)
- Notes: The `LoginPasswordSystemAlert` using `UIAlertController` is an unusual pattern in SwiftUI — system alerts avoid custom keyboard handling, autocorrect suppression, and secure text entry edge cases that SwiftUI `SecureField` can have. The `isAuthInProgress` centralization is a good UX improvement — prevents triggering multiple auth flows simultaneously.

---

### 4b560da Merge pull request #12 from thexflow/feature/XFL-56

Merge commit, see feature commits in batch 09 (`4971b79`, `5d74a4c`) and this batch (`ff2257c`).

---

### 5b68817 XFL-61 Improve Apple Sign-In client diagnostics

**What**: Major Apple Sign-In reliability improvements. Adds `com.apple.developer.applesignin` entitlement to `xflow-ios.entitlements` (was missing — Apple Sign-In button would silently fail without it). Adds runtime entitlement check via `SecTaskCopyValueForEntitlement` (checks if capability is properly configured). Comprehensive error diagnostics: per-`ASAuthorizationError.Code` human-readable messages (unknown, failed, invalidResponse, notHandled, notInteractive, canceled). Debug logging: credential type, entitlement status, identity token length. Moves `isLoading = true` to top of handler, adds `defer { isLoading = false }`. Handles non-`ASAuthorizationError` errors separately. 80 lines added.

**Scaffolding assessment**:
- [ ] ALARM

**ALARM**:
- What's missing: **Apple Sign-In entitlement was missing.** The `com.apple.developer.applesignin` capability was not in the entitlements file when Apple Sign-In was first added in batch 08 (commit `500300d`). This caused the Sign-In button to fail silently. The fix required adding the entitlement manually. If `ios-app-manager` had an `auth-provider add apple` command, it should automatically add this entitlement via `entitlements add`.
- Severity: LOW (one-time fix, but missing entitlements cause silent failures that are hard to debug)
- Suggested solution: Part of the `auth-provider add apple` command recommended in batch 08 — should auto-add `com.apple.developer.applesignin` entitlement.

---

### 3a5b4ce XFL-61 Avoid Apple Sign-In loading flicker

**What**: Fixes UI flicker during Apple Sign-In. Moves `isLoading = true` from the start of `handleAppleSignIn()` to just before the backend API call (after credential extraction). This avoids showing a loading spinner during the synchronous credential parsing phase. Adds `isHandlingAuthorization` flag to prevent duplicate `handleAppleSignIn` callbacks (can happen on rapid double-tap of Sign-In button). 12 lines added, 2 removed.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — Apple Sign-In UX fix
- Manual edit to `AppleAuthViewModel.swift` (12 lines added)
- Notes: The duplicate callback guard (`isHandlingAuthorization`) is a defensive pattern — Apple's `SignInWithAppleButton` `onCompletion` can fire multiple times in rapid succession. Good UX pattern.

---

### 34a216b XFL-61 Fix Apple Sign-In entitlement check compile

**What**: Removes the `isAppleSignInCapabilityEnabled()` method (30 lines) and `import Security`. The `SecTaskCreateFromSelf` API used for runtime entitlement checking doesn't compile on iOS — it's a macOS-only API (`Security.framework` SecTask functions are not available on iOS). The entitlement check was added just 12 minutes earlier in `5b68817`. Also removes the debug log line that called the method.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — compile error fix (removing macOS-only API usage)
- Manual edit to `AppleAuthViewModel.swift` (30 lines removed)
- Notes: `SecTaskCreateFromSelf` is macOS-only — a platform knowledge gap. On iOS, you can't programmatically check entitlements at runtime. The proper approach is to handle the error cases (ASAuthorizationError.unknown with code 1000 often indicates missing capability) rather than pre-checking. The diagnostic error messages from `5b68817` remain — they provide the same information reactively.

---

### 2a8372f Merge pull request #17 from thexflow/feature/XFL-61

Merge commit, see feature commits `5b68817`, `3a5b4ce`, `34a216b` (Apple Sign-In improvements) above.

---

### b8e831d XFL-63 Remove rocket emoji from production label

**What**: Removes the rocket emoji (🚀) from the production environment display name. `"🚀 Production"` → `"Production"`. Single-line change in `Config.swift`.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — cosmetic config change
- Manual edit to `Config.swift` (1 line changed)

---

### 0c00fbc XFL-63 Hide environment indicators in release builds

**What**: Wraps environment-specific UI elements with `#if DEBUG` conditional compilation. On auth screen: environment badge (colored dot + environment name in capsule) hidden in Release. In Settings: entire environment info section (environment name, API URL, bundle ID, version, build, evaluation status, current user ID, environment picker) hidden in Release. 8 lines added across 2 files.

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — release build hardening
- Manual edits:
  - `TelegramAuthView.swift` — `#if DEBUG` around environment badge (4 lines)
  - `SettingsView.swift` — `#if DEBUG` around environment section and badge color property (4 lines)
- Notes: This is standard practice — debug/environment information should never appear in production builds. In a scaffolded project, the `app-config setup` generated code should use `#if DEBUG` by default for any environment display UI.

---

### 69e3242 Merge pull request #18 from thexflow/feature/XFL-63

Merge commit, see feature commits `b8e831d` and `0c00fbc` (environment indicators) above.

---

### c55b47c Ignore xflow-ios shared scheme file

**What**: Adds `xflow-ios.xcodeproj/xcshareddata/xcschemes/xflow-ios.xcscheme` to `.gitignore` and deletes the 102-line scheme XML file from version control. The scheme was auto-generated by Xcode and shouldn't be tracked (it contains developer-specific settings like debugger selection and test plans).

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — repo hygiene
- Manual edits: `.gitignore` (1 line) + scheme file deletion
- Notes: In a Tuist-managed project, schemes are auto-generated by `tuist generate` and live in DerivedData — never in the repo. This is a non-issue for ios-app-manager projects. Reinforces the gitignore completeness ALARM from batches 01/06/07.

---

### 1288d33 Update TelegramAuthView

**What**: Changes the Privacy Policy URL from `https://xflow.org/privacy-policy` to `https://xflow.org/privacy`. Third URL change for legal links (after `7d597eb` added the original URL and `58d23d1` changed terms URL).

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — URL correction
- Manual edit to `TelegramAuthView.swift` (1 line)
- Notes: This is the THIRD legal URL change across 3 commits. Reinforces batch 09's ALARM about hardcoded URLs — legal URLs should come from config, not source code.

---

### c672712 Bump build number

**What**: Bumps `CURRENT_PROJECT_VERSION` from 1 to 2 across all 8 target build settings in `project.pbxproj` (Debug/Release for: main app, tests, UI tests, Live Activity extension). Version `MARKETING_VERSION` stays at 0.1.9 (main app) and 0.1.2 (extension/tests).

**Scaffolding assessment**:
- [x] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — build number bump
- Manual edit to `project.pbxproj` (8 lines changed)
- Notes: In a Tuist-managed project, build number is set once in the manifest — not in 8 places in pbxproj. This is a non-issue for ios-app-manager projects. The test targets still show old Team ID `CC258N857A` (should be `H446YY77RR` per batch 04 fix — this was missed during the Team ID update).

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| ff2257c Password login UI | YES | Auth UI refactoring, manual |
| 4b560da Merge #12 | SKIP | Merge commit |
| 5b68817 Apple Sign-In diagnostics | PARTIAL | Missing entitlement should be auto-added by auth scaffolding |
| 3a5b4ce Apple loading flicker fix | YES | UX fix, manual |
| 34a216b Entitlement check compile fix | YES | Platform API knowledge gap |
| 2a8372f Merge #17 | SKIP | Merge commit |
| b8e831d Remove rocket emoji | YES | Cosmetic, manual |
| 0c00fbc Hide env indicators | YES | Release hardening, manual |
| 69e3242 Merge #18 | SKIP | Merge commit |
| c55b47c Ignore scheme file | YES | Gitignore, non-issue for Tuist |
| 1288d33 Update privacy URL | YES | Third URL change, reinforces config need |
| c672712 Bump build number | YES | Non-issue for Tuist (single source of truth) |

### Critical ALARMs (sorted by severity)

1. **LOW: Missing Apple Sign-In entitlement** — `com.apple.developer.applesignin` was not added when Apple Sign-In was first implemented (batch 08). If auth provider scaffolding existed, it should auto-add the required entitlement. Not a new gap — extends batch 08's MEDIUM ALARM about auth provider scaffolding.

### Observations

- **Batch 10 is entirely polish and fixes.** No new features, no new modules, no new architectural patterns. All 8 non-merge commits are: UI refinements (password login, Apple Sign-In UX), release hardening (#if DEBUG), repo hygiene (gitignore, scheme deletion), URL corrections, and build number bumps. This is the final batch before the app likely ships to TestFlight/App Store.

- **All commits are by Ivan Oparin** — same-day work (Feb 23-24, 2026) finishing the XFL-56, XFL-61, and XFL-63 feature branches. Rapid iteration with 3-12 minute gaps between commits.

- **The `SecTaskCreateFromSelf` mistake is instructive** (5b68817 → 34a216b, 12 minutes apart). `SecTask` APIs are macOS-only, not available on iOS. This is a common platform confusion when developers come from macOS background or when AI assistants suggest macOS-only APIs. In a scaffolded project, the generated Apple Sign-In code should use error handling (ASAuthorizationError codes) rather than pre-flight entitlement checks.

- **Legal URL volatility is confirmed.** Three changes to legal URLs across batches 09-10 (privacy-policy → privacy, terms-of-use → terms, now privacy-policy → privacy again). These URLs should absolutely live in config, not source code.

- **Test targets have stale Team ID.** `c672712` reveals that test and UI test targets still use the old Team ID `CC258N857A` from the original developer account, while the main app and extension use `H446YY77RR`. This was missed during the batch 04 Team ID update (commit `0073f86`). In a Tuist project, Team ID is set once — this inconsistency couldn't happen.

- **No new scaffolding gaps identified.** This batch confirms existing ALARMs (auth provider scaffolding, hardcoded URLs, gitignore completeness) but introduces no new gaps. The codebase is in a "ship it" phase — all infrastructure is in place, remaining work is polish.
