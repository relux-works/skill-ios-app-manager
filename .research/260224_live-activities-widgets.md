# Live Activities & Widgets — Research (TASK-260224-nvgfzd)

- **Date:** 2026-02-24
- **Audience:** iOS engineers working in a Tuist-based modular codebase
- **Goal:** Understand how to build Widgets (WidgetKit) + Live Activities (ActivityKit), including timelines, interactive widgets, and push-to-update via APNs.
- **Fact-checking (local):** Verified key API names + OS availability against the iPhoneSimulator SDK that ships with **Xcode 26.2** (`ActivityKit.swiftinterface`, `WidgetKit.swiftinterface`) and Xcode’s **Widget Extension** template sources.
- **Network note:** This environment has no internet access; references point to Apple docs/WWDC for follow-up reading.

---

## Scope

In scope:

- **WidgetKit architecture**: widget extensions, `Widget`/`WidgetBundle`, timeline providers, refresh policies, and configuration.
- **Interactive widgets (iOS 17+)** using **App Intents**.
- **Live Activities (iOS 16.1+)** using **ActivityKit** + presentation via **WidgetKit** (`ActivityConfiguration` + `DynamicIsland`).
- **Push-to-update** Live Activities with **APNs** (token flow + high-level payload model).
- **Push-to-start** token availability (iOS 17.2+).
- **Shared data** patterns between app ↔ widget/live activity (App Groups, files, and SwiftData considerations).
- **Tuist integration considerations**: targets, dependency boundaries, Info.plist, and embedding extensions.

Out of scope:

- watchOS complications, legacy Today extensions
- deep design/HIG guidance beyond engineering constraints
- production server/APNs auth implementation details (covered only at the integration boundary level)

---

## Highlights / Key Takeaways

1. **Widgets are timeline-driven snapshots**: the system asks your extension for entries; you *don’t* “run a widget continuously”. Design for caching, small data, and infrequent refresh.
2. **Interactive widgets (iOS 17+) are App Intents**: tap actions don’t execute arbitrary SwiftUI code; they execute an `AppIntent` in an extension-friendly environment and then the widget UI refreshes.
3. **Live Activities are defined in the widget extension** (`ActivityConfiguration`), but **started/updated/ended by the app** via `Activity.request`, `Activity.update(_:)`, `Activity.end(_:)`.
4. **Push-to-update requires a per-activity push token** (requested at start via `pushType: .token`), which your server uses to send APNs “liveactivity” pushes.
5. **Push-to-start is a separate token stream (iOS 17.2+)**: `Activity<Attributes>.pushToStartTokenUpdates` is distinct from an activity’s per-instance `pushTokenUpdates`.
6. **Tuist boundary tip:** put `ActivityAttributes` + shared models in a **shared target** used by both the app and the widget extension; don’t duplicate types (server JSON mapping depends on them).

---

## 1) Mental Model: Widgets vs Live Activities

### Widgets (WidgetKit)

- **Where:** Home Screen, Lock Screen widgets, StandBy (varies by device/OS).
- **Execution model:** your extension is invoked to produce a **timeline** of entries; the system renders SwiftUI from the chosen entry.
- **Update model:** refreshes are best-effort and system-budgeted; request reload via `WidgetCenter`, but don’t rely on immediate execution.

### Live Activities (ActivityKit + WidgetKit)

- **Where:** Lock Screen + Dynamic Island (supported devices).
- **Execution model:** the system presents a Live Activity UI that you define with `ActivityConfiguration` in the widget extension.
- **Update model:**
  - local updates via `Activity.update(_:)` while your app is able to run
  - remote updates via APNs using the activity’s push token

**Sources**

- Apple Developer Documentation: ActivityKit, WidgetKit
  - `https://developer.apple.com/documentation/activitykit`
  - `https://developer.apple.com/documentation/widgetkit`
- WWDC (start points to search in Apple Developer Videos, 2023–2025):
  - “Bring widgets to life” (interactive widgets/App Intents)
  - “Meet Live Activities” + “Update Live Activities with push notifications” (ActivityKit + remote updates)

---

## 2) WidgetKit Architecture Deep Dive (timelines, providers, bundles)

### 2.1 Widget Extension Target

- A **Widget Extension** is an **app extension target** with:
  - `NSExtensionPointIdentifier = com.apple.widgetkit-extension` in its extension Info.plist section (matches Xcode template).
  - one `@main` entry point:
    - either a single `Widget`, or
    - a `WidgetBundle` containing multiple widgets (including a Live Activity widget).
- One widget extension can ship:
  - multiple widgets (`StaticConfiguration`, `AppIntentConfiguration`, etc.)
  - a Live Activity widget (`ActivityConfiguration`)

### 2.2 Timeline Provider & Entry

Core types:

- `TimelineProvider` supplies:
  - `placeholder(in:)` (used in the widget gallery)
  - `getSnapshot(in:completion:)` (used for previews)
  - `getTimeline(in:completion:)` (the real data)
- `TimelineEntry` is your per-point-in-time snapshot (must include `date`).
- `TimelineReloadPolicy` controls refresh:
  - `.atEnd`, `.never`, `.after(Date)` (API verified in SDK interface).

Guidance:

- Keep entries small and deterministic; load heavy data in the provider, not in the SwiftUI view body.
- Use caching (shared container/App Group) to avoid repeated network work.

### 2.3 Configuration: Static vs (App) Intent

Common patterns:

- **Static widgets**: `StaticConfiguration(kind:provider:content:)`
- **Configurable widgets** (modern): `AppIntentConfiguration(kind:intent:provider:content:)` with:
  - `WidgetConfigurationIntent` (e.g., `ConfigurationAppIntent` in Xcode template)
  - `AppIntentTimelineProvider` (async `snapshot`/`timeline`)
- **Configurable widgets (legacy)**: `IntentConfiguration(kind:intent:provider:content:)` using **SiriKit Intents**
  - Uses an `.intentdefinition` file that generates an intent type (e.g., `ConfigurationIntent`) and an `IntentTimelineProvider`.
  - This pattern still exists for backwards compatibility, but Apple’s current direction for new widget configuration is **App Intents**.

### 2.4 Widget UI specifics

- For iOS 17+, the template uses `.containerBackground(..., for: .widget)` for proper widget background material.
- Use `widgetURL(_:)` to deep-link when the widget (outside controls) is tapped.

**Sources**

- Apple Developer Documentation: WidgetKit
  - `https://developer.apple.com/documentation/widgetkit`
  - `https://developer.apple.com/documentation/widgetkit/timelineprovider`
  - `https://developer.apple.com/documentation/widgetkit/widgetbundle`
- Local API verification:
  - WidgetKit SDK interface: `.../iPhoneSimulator26.2.sdk/.../WidgetKit.swiftinterface` (confirms `TimelineProvider`, `WidgetCenter.reloadTimelines`, `TimelineReloadPolicy`)
  - Xcode template: `Widget Extension.xctemplate/Widget-Static.swift`, `Widget-Configurable.swift`
- WWDC (2023–2025): search “WidgetKit timeline”, “Widget bundles”, “App Intent widgets”

---

## 3) Interactive Widgets (iOS 17+)

### 3.1 What “interactive” means

- Interactions run **App Intents** (not arbitrary view code).
- Common controls:
  - `Button(intent:)`
  - `Toggle(intent:)`
  - custom controls via intents + shared state

### 3.2 Data flow pattern (recommended)

1. Store “source of truth” in a shared location (App Group UserDefaults / file / database).
2. `AppIntent.perform()` mutates shared state (keep it fast).
3. Request refresh if needed:
   - `WidgetCenter.shared.reloadTimelines(ofKind:)` or `reloadAllTimelines()`

### 3.3 Constraints / pitfalls

- Intents must complete quickly; long-running work risks termination.
- Don’t depend on network availability during intent execution; prefer cached/offline updates and reconcile later in the main app.
- Avoid PII leakage in widget UI (Lock Screen visibility).

**Sources**

- Apple Developer Documentation: App Intents, WidgetKit
  - `https://developer.apple.com/documentation/appintents`
  - `https://developer.apple.com/documentation/widgetkit`
- Local API verification:
  - Compiles against iPhoneSimulator SDK with `Button(intent:)` availability checks (`@available(iOS 17.0, *)`)
  - Xcode template: `Widget Extension.xctemplate/AppIntent.swift` (configuration intent)
- WWDC (2023–2025): search “interactive widgets”, “App Intents in widgets”

---

## 4) Live Activities (ActivityKit) Deep Dive

### 4.1 Data model: `ActivityAttributes` and `ContentState`

- `ActivityAttributes` defines:
  - **fixed** attributes (don’t change for the activity lifetime)
  - nested `ContentState` for **dynamic** state (Codable + Hashable)
- Keep both **small**; large payloads can fail with `ActivityAuthorizationError.attributesTooLarge` (exists in ActivityKit).

### 4.2 Lifecycle in the app target

Modern API (iOS 16.2+; verified via SDK interface):

- Start: `Activity.request(attributes:content:pushType:)`
- Update: `activity.update(_:)`
- End: `activity.end(_:dismissalPolicy:)`

If you must support **iOS 16.1**, you’ll need availability-guarded fallbacks:

- Start (16.1-only): `Activity.request(attributes:contentState:pushType:)` (deprecated in 16.2)
- Update/end (16.1-only): `update(using:)` / `end(using:)` (deprecated in 16.2)

Key observations (verified in SDK interface):

- Legacy `contentState` / `update(using:)` / `end(using:)` were deprecated in iOS 16.2 in favor of `ActivityContent`.
- `ActivityAuthorizationInfo` exposes:
  - `areActivitiesEnabled`
  - `frequentPushesEnabled` (iOS 16.2+)

### 4.3 UI in the widget extension

Live Activity UI is a `Widget` whose configuration is:

- `ActivityConfiguration(for: Attributes.self) { context in ... } dynamicIsland: { context in ... }`

The Xcode template demonstrates:

- Lock screen/banner UI in the first closure
- Dynamic Island regions via `DynamicIslandExpandedRegion` + compact/minimal variants
- Deep link: `.widgetURL(...)`
- Styling: `.activityBackgroundTint`, `.activitySystemActionForegroundColor`, `.keylineTint`
- Preview: `#Preview("Notification", as: .content, using: ...) { ... } contentStates: { ... }`

**Sources**

- Apple Developer Documentation: ActivityKit, Live Activities
  - `https://developer.apple.com/documentation/activitykit`
- Local API verification:
  - ActivityKit SDK interface: `.../ActivityKit.swiftinterface` (confirms availability/deprecations and `ActivityAuthorizationInfo`)
  - Xcode template: `Widget Extension.xctemplate/LiveActivity.swift`
- WWDC (2023–2025): search “ActivityKit”, “Dynamic Island”, “Live Activities”

---

## 5) Push-to-Update Live Activities via APNs

### 5.1 Token flow (app → server)

To enable remote updates:

1. Request the activity with `pushType: .token`.
2. Observe the activity’s token stream:
   - `activity.pushTokenUpdates` (`AsyncSequence<Data>`, verified in SDK interface)
3. Send the token to your server (treat it as secret-ish; tie it to the activity ID and user/session).

Notes:

- Tokens can rotate; keep listening and update your backend.
- There is also a **push-to-start** token stream (below) that is *not* per-activity.

### 5.2 Sending updates (server → APNs)

High-level requirements (verify exact headers/payload shape in Apple docs before implementing):

- APNs push type: **liveactivity**
- APNs topic: typically `$(APP_BUNDLE_ID).push-type.liveactivity`
- The APNs “device token” you send to is the **Live Activity’s push token** (from `pushTokenUpdates`), not the normal device token used for user-visible notifications.
- Payload encodes the Live Activity state:
  - “attributes” (for start)
  - “content-state” for updates (must match your `ContentState` Codable keys)
  - an “event” field that tells APNs whether this is start/update/end

### 5.3 Frequent update considerations

ActivityKit exposes:

- `ActivityAuthorizationInfo().frequentPushesEnabled` (iOS 16.2+) and an async updates stream.

Implication:

- The OS/user can gate “more frequent” Live Activity update behavior; treat frequent updates as opt-in and design graceful degradation.

### 5.4 Push-to-start (iOS 17.2+)

SDK interface confirms availability:

- `Activity<Attributes>.pushToStartTokenUpdates` (iOS 17.2+)
- `Activity<Attributes>.pushToStartToken` (iOS 17.2+)

Use case:

- Your app can obtain a push-to-start token and provide it to your server so the server can **start** a Live Activity remotely (per Apple docs).

**Sources**

- Apple Developer Documentation: ActivityKit (push updates / push-to-start documentation lives under ActivityKit)
  - `https://developer.apple.com/documentation/activitykit`
- Local API verification:
  - ActivityKit SDK interface: `pushTokenUpdates` + `pushToStartTokenUpdates` availability (iOS 17.2+), `frequentPushesEnabled` (iOS 16.2+)
- WWDC (2023–2025): search “Live Activities push”, “push-to-start Live Activities”

---

## 6) Sharing Data Between App ↔ Widget / Live Activity

### 6.1 App Groups (baseline)

Common shared mechanisms:

- **UserDefaults**: `UserDefaults(suiteName: "group.your.app")`
- **File container**: `FileManager.default.containerURL(forSecurityApplicationGroupIdentifier:)`

Recommended:

- Define a small “shared storage” layer in a shared module (e.g., `SharedKit`) used by:
  - the app
  - the widget extension (widgets + live activities)
  - App Intents (if implemented in the extension target)

### 6.2 SwiftData considerations

SwiftData can be used as a shared persistence layer, but be careful:

- Extension processes are short-lived; keep reads small and avoid migrations at widget runtime.
- If sharing a store, place it in the App Group container and ensure both targets use the same model schema + configuration.
- Prefer a “read-optimized” view for widgets; write/maintenance should usually happen in the main app.

**Sources**

- Apple Developer Documentation: App Groups, SwiftData
  - `https://developer.apple.com/documentation/swiftdata`
- WWDC (2023–2025): search “SwiftData app group”, “data sharing with widgets”

---

## 7) Tuist Integration Considerations

### 7.1 Target layout (recommended)

At minimum:

- `App` target (`.app`) — owns:
  - starting/updating/ending Live Activities (ActivityKit calls)
  - push token registration with backend
- `Widgets` target (`.appExtension`) — owns:
  - widgets (`WidgetKit`)
  - Live Activity UI (`ActivityConfiguration`)
- `Shared` target (framework or Swift package) — owns:
  - `ActivityAttributes` + `ContentState` types
  - shared storage (App Group paths, serialization, etc.)

Why shared types matter:

- The app and widget extension must compile against the *same* `ActivityAttributes` type.
- Your server payload schema must match the Codable representation of `ContentState`.

### 7.2 Embedding the extension

In Tuist, you typically embed an extension by adding it as a dependency of the app target.

Example pattern (from a Tuist project in this workspace: `.temp/repos/membrana-app/Project.swift`):

- Extension target: `product: .appExtension`
- App target dependencies include: `.target(name: "<extension-target-name>")`

Widget extension sketch (Tuist `Project.swift`-style pseudocode; adapt to your project conventions):

```swift
let widgets = Target.target(
  name: "app-widgets",
  destinations: .iOS,
  product: .appExtension,
  bundleId: "\(bundleId).widgets",
  infoPlist: .dictionary([
    "NSExtension": .dictionary([
      "NSExtensionPointIdentifier": .string("com.apple.widgetkit-extension")
    ])
  ]),
  entitlements: .dictionary([
    "com.apple.security.application-groups": .array([
      .string("group.\(bundleId)")
    ])
  ]),
  dependencies: [
    .target(name: "shared") // ActivityAttributes + shared storage
  ]
)
```

### 7.3 Info.plist & capabilities

Widget extension:

- `NSExtensionPointIdentifier = com.apple.widgetkit-extension`

App target Info.plist (Live Activities):

- Set `NSSupportsLiveActivities = YES`
- If you want “frequent updates”: set `NSSupportsLiveActivitiesFrequentUpdates = YES`

(Both keys are recognized by Xcode’s build system spec and link to Apple docs.)

Entitlements & signing (high-level):

- App Groups entitlement for shared data.
- Push Notifications capability if you use remote push to update Live Activities.

**Sources**

- Apple Developer Documentation: Info.plist keys
  - `https://developer.apple.com/documentation/bundleresources/information_property_list/nssupportsliveactivities`
  - `https://developer.apple.com/documentation/bundleresources/information_property_list/nssupportsliveactivitiesfrequentupdates`
- Local references:
  - Xcode Widget Extension template: `TemplateInfo.plist` (widget extension point identifier)
  - Xcode build spec: `CoreBuildSystem.xcspec` includes the Live Activities Info.plist keys
  - Example Tuist app extension setup: `.temp/repos/membrana-app/Project.swift`

---

## 8) Practical Engineering Checklist (when implementing)

- Decide what is a **Widget** vs a **Live Activity**:
  - Widgets: periodic snapshots; good for “status at a glance”
  - Live Activities: time-sensitive, ongoing event; good for “in progress right now”
- Define `ActivityAttributes` / `ContentState` in a **shared module**.
- Add widget extension target; include:
  - `WidgetBundle` for multiple widgets and the Live Activity widget
  - `ActivityConfiguration` + `DynamicIsland` UI
- App target:
  - set `NSSupportsLiveActivities` (+ FrequentUpdates if needed)
  - start activities with `pushType: .token` if remote updates are needed
  - pipe `pushTokenUpdates` to backend
- Backend:
  - store per-activity token, handle rotation
  - implement APNs “liveactivity” updates (start/update/end)
- Interactive widgets:
  - implement App Intents that mutate shared state and reload timelines
- Testing:
  - use `#Preview` for widgets and live activities
  - validate app-group data access from extension

---

## Appendix: Fact-check Notes (local, verifiable)

These points were checked against local SDK/template sources (useful when you’re reviewing API availability or wiring in Tuist):

- **ActivityKit 16.1 vs 16.2 API split**
  - `Activity.request(attributes:contentState:pushType:)` is present but deprecated in iOS 16.2.
  - `Activity.request(attributes:content:pushType:)` is available starting iOS 16.2.
  - Source: `ActivityKit.swiftinterface` (iPhoneSimulator26.2 SDK), lines **42–55**.
- **Push-to-start token is iOS 17.2+**
  - `Activity<Attributes>.pushToStartTokenUpdates` / `.pushToStartToken` are iOS 17.2+.
  - Source: `ActivityKit.swiftinterface`, lines **137–152**.
- **Widget extension point identifier**
  - Widget extension Info.plist uses `NSExtensionPointIdentifier = com.apple.widgetkit-extension`.
  - Source: Xcode template `Widget Extension.xctemplate/TemplateInfo.plist`, lines **65–70**.
- **Widget timeline reload policies**
  - `TimelineReloadPolicy` exposes `.atEnd`, `.never`, `.after(Date)`.
  - Source: `WidgetKit.swiftinterface`, lines **1703–1706**.
- **Live Activities Info.plist keys**
  - `NSSupportsLiveActivities` and `NSSupportsLiveActivitiesFrequentUpdates` are recognized Info.plist keys.
  - Source: Xcode build spec `CoreBuildSystem.xcspec`, lines **4412–4429**.
