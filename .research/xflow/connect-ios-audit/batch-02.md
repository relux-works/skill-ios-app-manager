# Batch 02: Commits 11-20

Audit of connect-ios commits against ios-app-manager scaffolding capabilities.

---

### 5a45ce9 Add bottom padding when message input is focused

**What**: Adds keyboard-aware bottom padding to the message input area. Introduces `onFocusChange` callback on `MessageInputBubble` and a `@FocusState` binding so `ContactDetailView` can track when the message TextField is focused and add 12pt bottom padding.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — this is a UI polish fix within the CRM feature module
- Manual edits to:
  - `Packages/CRM/Sources/CRM/UI/Components/MessageInputBubble.swift` — add `onFocusChange` callback + `@FocusState`
  - `Packages/CRM/Sources/CRM/UI/Views/ContactDetailView.swift` — wire focus tracking for bottom padding

---

### 5b07f5b Add bottom padding on message input focus and show back button on profile

**What**: Follow-up fix — changes `isMessageInputFocused` from `@FocusState` to `@State` (managed via callback, not direct binding), adds debug logging for focus changes, removes `navigationBarBackButtonHidden(true)` from `ProfileView` so the back button is visible.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — minor UI fixes across existing feature modules
- Manual edits to:
  - `Packages/CRM/Sources/CRM/UI/Components/MessageInputBubble.swift` — debug logging
  - `Packages/CRM/Sources/CRM/UI/Views/ContactDetailView.swift` — `@State` instead of `@FocusState`
  - `Packages/Profile/Sources/Profile/UI/Views/ProfileView.swift` — remove `navigationBarBackButtonHidden`

---

### 5f82632 Deletion of contexts

**What**: Large feature — implements CRUD for AI context management. Adds 4 new API endpoints (delete chat context item, delete all chat context, delete global context item, delete all global context) to `APIEndpoint`, `CrmService`, and `ContactDetailViewModel`. Updates `MotivationsPanelView` with delete buttons per-item and "Delete All" buttons per-section, with confirmation dialogs. Adds `CollapsibleSectionView` delete callback. Adds analysis metrics display (execution time, cost in USD). New localization keys for delete confirmations (en + ru). Also adds `Package.resolved` and deletes leftover screenshot.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — feature-level code within CRM module (API calls, ViewModel methods, UI)
- Manual files:
  - `Packages/CRM/Sources/CRM/UI/Views/MotivationsPanelView.swift` — delete UI + confirmation dialogs + analysis metrics section
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+Flow.swift` — context deletion effects (in Relux, these would be effects/actions)
  - `Packages/CRM/Sources/CRM/Data/Api/CrmService.swift` — 4 new delete methods
  - `Packages/CRM/Sources/CRM/Business/CRM.Business+Action.swift` — context delete actions
  - Localization files: new keys for `button.deleteAll`, `crm.confirmDelete*`, `crm.deleteAllContextWarning`
- Notes: New model fields `analysisExecutionTimeMs` and `analysisTotalCostUsd` on `CrmContactExtended` with computed formatting — manual model additions.

---

### 8302843 Group chats

**What**: Adds group chat support to CRM. New `GroupSender` model (id, identifier, identifierType, name, avatar fields + computed `initials`). Adds `isGroup` field to `CrmContact` and `CrmContactExtended`. Updates `MessageBubble` to accept `isGroupChat` parameter, shows sender avatar (with Kingfisher image loading or initials placeholder) and sender name below incoming messages in groups. Adds `groupSender` field to `CrmMessage`. Updates previews with new required fields. Includes debug logging and ngrok URL change.

**Scaffolding assessment**:
- [ ] INSTRUCTION

**INSTRUCTION**:
- No ios-app-manager commands — feature code within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/GroupSender.swift` — new model (would live in `Data.Api.DTO` namespace)
  - `Packages/CRM/Sources/CRM/UI/Components/MessageBubble.swift` — group chat avatar, sender name display
  - Model updates: `isGroup` on `CrmContact`/`CrmContactExtended`, `groupSender` on `CrmMessage`
- Notes: The `GroupSender` model with initials computation is a good example of a DTO that would live in the scaffolded `Data.Api.DTO` namespace hierarchy. Bundle ID change (`connect-ios` → `connect-ios1`) appears to be a debug workaround — irrelevant for scaffolding.

---

### 09dbf19 Merge pull request #1 from connect-app/feature/CNTFRONT-401

Merge commit, see feature commit `8302843` (Group chats) above.

---

### 92aefd3 Загрузка чата постранично

**What**: Major refactoring — implements paginated message loading. Splits the monolithic "load contact with all messages" into v4 contact endpoint (without messages) + separate paginated messages endpoint. Adds `getContactByIdV4`, `getContactMessages` to `CrmService` with cursor-based pagination (`beforeTimestamp`/`afterTimestamp`). Rewrites `ContactDetailViewModel` to manage messages separately from contact: `loadInitialMessages()`, `loadOlderMessages()`, `pollForNewMessages()`, prefetch trigger at 10 messages from top. New `MessagePageResponse` model. Custom ISO8601 date decoder handling fractional seconds and timezone brackets. Fixes token refresh retry bug (was using `finalQueryItems` with duplicate params). New `InfoButtonWithMenu` component (211 lines) with long-press context menu and circular progress animation for analysis. Makes `phoneNumber` optional on search contacts. Reduces polling interval from 60s to 20s. Removes message input line limit cap.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands directly — this is a deep networking/ViewModel refactoring
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/MessagePageResponse.swift` — pagination response model
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+State.swift` — paginated messages state (`messages`, `hasMoreMessages`, `oldestTimestamp`, `newestTimestamp`)
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+Flow.swift` — pagination effects (loadInitial, loadOlder, pollNew)
  - `Packages/CRM/Sources/CRM/UI/Components/InfoButtonWithMenu.swift` — new component with analysis progress
  - Networking layer: custom date decoder, token refresh fix

**ALARM**:
- What's missing: **No pagination infrastructure scaffolding.** Paginated data loading (cursor-based with before/after timestamps, prefetch triggers, polling for new items) is a recurring mobile pattern. The ViewModel had to be substantially rewritten (~200+ lines of pagination logic). This pattern will be repeated for every list-based feature (contacts, organizations, notifications).
- Severity: MEDIUM (one-time per feature, but it's complex and error-prone boilerplate)
- Suggested solution: Consider a generic `PaginatedDataSource<T>` utility or at least document the pagination pattern in a reference guide. Not necessarily a CLI command, but a template or reusable component that `relux-feature` modules can import.

---

### 83d641e Merge pull request #2 from connect-app/feature/CNTFRONT-425

Merge commit, see feature commit `92aefd3` (Paginated chat loading) above.

---

### b8ff5f9 CNTFRONT-428: Lazy load attachments without URL on scroll

**What**: Implements lazy attachment URL loading — when messages arrive without `attachmentUrl` (backend optimization), the URL is fetched on-demand when the message bubble appears on screen. Adds `getAttachmentUrl` API endpoint and `CrmService` method. Adds attachment URL cache, loading state, and error tracking in `ContactDetailViewModel` (`attachmentUrlCache`, `loadingAttachmentUrls`, `attachmentUrlErrors`). Massive `MessageBubble` refactoring (~300+ lines changed) — every attachment type (image, video, audio, document) now handles 6 states: URL available, loading URL, error, deleted, auto-load on appear, fallback. Extracts `attachmentRetryPlaceholder` and `deletedAttachmentPlaceholder` helpers. New `AttachmentUrlResponse` model. Adds `message.deleted` localization key.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — complex feature code within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/AttachmentUrlResponse.swift` — new response model
  - `Packages/CRM/Sources/CRM/UI/Components/MessageBubble.swift` — massive refactoring for lazy loading
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+State.swift` — attachment URL cache state
  - New API endpoint + service method for attachment URL fetching
  - Localization: `message.deleted` key

**ALARM**:
- What's missing: **No lazy-loading / on-demand data fetching pattern.** The attachment URL loading implements a cache-check-fetch-retry pattern that is generic: check cache → show loading → fetch → update cache / handle error. This was entirely hand-rolled (~100 lines in ViewModel + ~300 lines in View refactoring). Similar patterns exist for profile images, file previews, etc.
- Severity: LOW (very domain-specific despite the repeating pattern — hard to generalize into scaffolding)
- Suggested solution: N/A for scaffolding. This is an architectural pattern better served by documentation or a shared utility class, not a CLI command.

---

### 71ebfef Merge pull request #3 from connect-app/feature/CNTFRONT-428

Merge commit, see feature commit `b8ff5f9` (Lazy load attachments) above.

---

### 668b5ab Add optimistic UI for message sending with inline status indicators

**What**: Implements optimistic UI for message sending. New models: `PendingMessage` (id, text, attachments, status, timestamp, isConfirmed) and `MessageStatus` enum (sending, sent, failed with error). New components: `MessageStatusView` (sending spinner, sent checkmark, failed with retry/cancel buttons) and `PendingMessageBubble` (full message bubble for pending messages with status indicator, red-tinted background on failure, opacity animation while sending). Rewrites `sendMessage()` in ViewModel: immediately creates pending message, clears input, sends to server in background, updates status on success/failure, matches with server response via polling to remove duplicates. Adds retry and cancel functionality for failed messages. Replaces toast notifications with inline status indicators.

**Scaffolding assessment**:
- [ ] INSTRUCTION + ALARM

**INSTRUCTION**:
- No ios-app-manager commands — feature code within CRM module
- Manual files:
  - `Packages/CRM/Sources/CRM/UI/Components/MessageStatusView.swift` — status indicator component (117 lines)
  - `Packages/CRM/Sources/CRM/UI/Components/PendingMessageBubble.swift` — pending message bubble (239 lines)
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/PendingMessage.swift` — pending message model (68 lines)
  - `Packages/CRM/Sources/CRM/Data/Api/DTO/MessageStatus.swift` — status enum (55 lines)
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+State.swift` — pending messages state
  - `Packages/CRMImpl/Sources/CRMImpl/Business/CRM.Business+Flow.swift` — optimistic send/retry/cancel effects
  - Localization: `status.sending`, `status.sent` keys

**ALARM**:
- What's missing: **No optimistic UI pattern support.** Optimistic UI (show result immediately, sync with server, handle failure with retry) is a standard mobile UX pattern used for sending messages, creating records, toggling states. The implementation here is ~700 lines across 4 new files + ViewModel changes. This pattern repeats wherever user actions need instant feedback.
- Severity: LOW (pattern is too domain-specific for CLI scaffolding, but worth documenting)
- Suggested solution: N/A for scaffolding. Document as an architectural pattern. The Relux architecture could potentially support this with a generic `OptimisticEffect<T>` in the framework itself, but that's a framework feature, not a scaffolding tool.

---

## Key Takeaways

### Scaffolding Coverage Summary

| Commit | Scaffoldable? | Key Gap |
|--------|:---:|---------|
| 5a45ce9 Bottom padding | YES | Pure UI fix, manual |
| 5b07f5b Focus fix + back button | YES | Pure UI fix, manual |
| 5f82632 Context deletion | YES | Feature code — API, ViewModel, UI all manual |
| 8302843 Group chats | YES | New model + UI, manual |
| 09dbf19 Merge #1 | SKIP | Merge commit |
| 92aefd3 Paginated messages | PARTIAL | No pagination pattern support |
| 83d641e Merge #2 | SKIP | Merge commit |
| b8ff5f9 Lazy load attachments | YES | Lazy loading pattern, but too domain-specific |
| 71ebfef Merge #3 | SKIP | Merge commit |
| 668b5ab Optimistic UI | YES | Optimistic UI pattern, but too domain-specific |

### Critical ALARMs (sorted by severity)

1. **MEDIUM: No pagination infrastructure** — Cursor-based pagination with prefetch, polling for new items, and deduplication is ~200+ lines of boilerplate per feature. This will repeat for contacts list, organizations list, notifications, etc. A generic `PaginatedDataSource<T>` or documented pattern would help.

2. **LOW: No lazy-loading pattern** — On-demand data fetching with cache/loading/error states is hand-rolled. Generic enough to warrant a utility, but too specific for scaffolding.

3. **LOW: No optimistic UI pattern** — Send-immediately-sync-later with retry/cancel is ~700 lines. Valuable pattern but better as framework support (Relux) than scaffolding.

### Observations

- **Batch 02 is heavily CRM-focused** — all 7 non-merge commits are within the CRM feature. No new modules created, no new dependencies added. This is deep feature development, not project infrastructure.
- **Pattern complexity is increasing** — pagination, lazy loading, optimistic UI are all significantly more complex than the module scaffolding and IoC setup that ios-app-manager handles. The gap between "scaffolded skeleton" and "production feature" is substantial.
- **The scaffolding tool covers the initial setup well** — module creation, dependency wiring, namespace hierarchy are all handled. But once you're past scaffolding, 95%+ of the code is manual feature development. This is expected and correct — the tool should not try to generate business logic.
- **Custom date decoding fix is noteworthy** — ISO8601 with fractional seconds and timezone brackets (`[Europe/Moscow]`) required a custom decoder. This is a common backend compatibility issue that could be documented as a gotcha for networking setup.
