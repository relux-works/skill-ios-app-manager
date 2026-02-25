# App Intents / Siri / Shortcuts — Research (TASK-260224-1g1c4h)

- **Date:** 2026-02-24
- **Focus:** App Intents framework + Siri/Shortcuts/Spotlight/Widgets (Apple docs + WWDC/Tech Talks 2022–2025)
- **Goal:** reusable intent architecture — define once, use across app + extensions (Tuist-friendly)

## Scope

Cover:

- App Intents vs SiriKit Intents (legacy `INIntent` / `.intentdefinition` + Intents Extension)
- Core App Intents building blocks: `AppIntent`, parameters (`@Parameter` / `IntentParameter`), entities (`AppEntity`) + queries (`EntityQuery`)
- Integrations: Shortcuts app, Siri, Spotlight, widgets (config + interactivity)
- Cross-target sharing patterns: main app, widget extensions, App Intents extension
- Practical module/target organization patterns for a Tuist project

---

## Highlights / Key Takeaways

1. **App Intents are the unified, code-first way to expose app actions across the system** (Shortcuts, Siri, Spotlight; and for widgets: configuration + interactive actions). The same intent type can power multiple surfaces.  
2. **Entities + queries are the “dynamic options” backbone**: model user-selectable objects as `AppEntity` and supply them via `EntityQuery` (and its specializations). This drives parameter pickers, search, and (optionally) Spotlight indexing.  
3. **Widgets have a version split:** older `IntentConfiguration` uses SiriKit intents (`INIntent`); iOS 17+ adds `AppIntentConfiguration` using `WidgetConfigurationIntent` (App Intents) for widget configuration.  
4. **Cross-target reuse is a packaging problem first:** an intent must be present in every bundle that needs to execute it (app, widget extension, App Intents extension). iOS 17+ adds first-class packaging (`AppIntentsPackage`) so frameworks can export intents; WWDC25 expands packaging to Swift packages + static libraries.  
5. **A practical “define once, use everywhere” architecture is:** keep business logic in a pure domain module, define intents/entities/queries in a shared `IntentsKit`, and link that kit into the app + each extension target that needs the intents.

---

## 1) App Intents vs SiriKit Intents (legacy)

### 1.1 SiriKit / Intents framework (legacy pattern)

The older Siri/Shortcuts customization model is based on **SiriKit Intents** (`INIntent`) defined in a `.intentdefinition` file and handled by an **Intents Extension**. This still exists and is still used for:

- **legacy custom intent-based shortcuts** (especially if you already shipped `.intentdefinition`-based actions),
- **widget configuration pre–iOS 17** (`IntentConfiguration`),
- and **SiriKit domains** (messaging, payments, etc.) where App Intents aren’t the replacement.

### 1.2 App Intents (modern pattern)

**App Intents** are **code-first**: you implement actions as `AppIntent` types with parameters and a `perform()` implementation. These intents can then be used by system experiences like:

- **Shortcuts** actions + curated “App Shortcuts”
- **Siri** invocation (via App Shortcuts phrases and system suggestions)
- **Spotlight** actions and entity indexing
- **Widgets** (configuration + interactive widget buttons)

### Sources

- WWDC22 — Dive into App Intents: https://developer.apple.com/videos/play/wwdc2022/10032/
- WWDC22 — Implement App Shortcuts with App Intents: https://developer.apple.com/videos/play/wwdc2022/10170/
- WidgetKit doc — Making a configurable widget (IntentConfiguration vs AppIntentConfiguration): https://developer.apple.com/documentation/widgetkit/making-a-configurable-widget
- AppIntents API collection (entry point): https://developer.apple.com/documentation/appintents

---

## 2) Core App Intents building blocks (what you actually implement)

### 2.1 `AppIntent` (action definition)

At minimum, an App Intent is:

- **metadata** (title/description, etc.),
- **parameters** (via `@Parameter` / `IntentParameter`),
- and `perform()` returning an **intent result** (often including a dialog/snippet for UI surfaces).

The important design implication is that an App Intent is not “a Shortcuts action only” — it’s the core action model reused by multiple system surfaces.

### 2.2 Parameters (`@Parameter` / `IntentParameter`)

Parameters are declared using the `@Parameter` property wrapper (which is backed by the `IntentParameter` type). Parameters can be simple values, enums, or `AppEntity` types.

Key things to design up front:

- **User-facing names** for parameters (because they show up in Shortcuts and other UI pickers).
- **Default values** / optionality and what happens when not provided.
- **Parameter summaries** (how the action reads in Shortcuts).

### 2.3 Entities + Queries (`AppEntity` + `EntityQuery`)

If a parameter represents a **user-selectable object** (project, account, list, document…), model it as an `AppEntity`.

You then supply the picker/search behavior via `EntityQuery` (or its specializations), usually implementing:

- `entities(for:)` (resolve identifiers)
- `suggestedEntities()` (top suggestions)
- optional search support via string/property queries (when needed)

This is how you get:

- good Shortcuts parameter pickers,
- search-as-you-type,
- and (optionally) Spotlight integration for those objects.

### 2.4 Spotlight indexing for entities (App Intents + Spotlight)

Apple provides an App Intents-based path to make entities searchable in Spotlight (indexing entities + their properties).

### Sources

- AppIntents API collection: https://developer.apple.com/documentation/appintents
- `IntentParameter` docs: https://developer.apple.com/documentation/appintents/intentparameter
- `AppEntity` docs: https://developer.apple.com/documentation/appintents/appentity
- `EntityQuery` docs: https://developer.apple.com/documentation/appintents/entityquery
- AppIntents doc — Indexing app entities in Spotlight: https://developer.apple.com/documentation/appintents/indexing-app-entities-in-spotlight
- WWDC25 — Develop for Shortcuts and Spotlight with App Intents: https://developer.apple.com/videos/play/wwdc2025/10179/

---

## 3) Integrations: Shortcuts, Siri, Spotlight

### 3.1 Shortcuts app integration (actions + App Shortcuts)

Two related concepts:

- **Actions**: Any App Intent that’s exposed shows up as an action in the Shortcuts editor.
- **App Shortcuts**: a curated set of shortcuts you define (with invocation phrases) so users can run them immediately — without requiring “donation” from usage.

The key API surface here is `AppShortcutsProvider`, where you declare `AppShortcut` entries and their phrases.

### 3.2 Siri integration (phrases, tips, and results)

Siri can trigger App Shortcuts by phrases, and Apple provides UI patterns to help users discover them in-app (tips / links). WWDC24 covers how to bring your app’s capabilities to Siri, including the role of App Intents and how responses appear.

### 3.3 Spotlight integration (actions + discoverability)

Spotlight can surface App Shortcuts as actions and can also search/index domain entities (via App Intents entity indexing and/or Core Spotlight, depending on your needs).

### Sources

- App Shortcuts API (entry): https://developer.apple.com/documentation/appintents/appshortcuts
- `AppShortcutsProvider` docs: https://developer.apple.com/documentation/appintents/appshortcutsprovider
- WWDC22 — Design App Shortcuts: https://developer.apple.com/videos/play/wwdc2022/10169/
- WWDC22 — Implement App Shortcuts with App Intents: https://developer.apple.com/videos/play/wwdc2022/10170/
- WWDC23 — Spotlight your app with App Shortcuts: https://developer.apple.com/videos/play/wwdc2023/10102/
- WWDC24 — Bring your app to Siri: https://developer.apple.com/videos/play/wwdc2024/10133/
- Apple — Shortcuts for Developers: https://developer.apple.com/shortcuts/

---

## 4) Widgets (“widget intents”): configuration + interactivity

### 4.1 Configurable widgets

Widget configuration has a **timeline split**:

- **iOS 14+ (legacy):** `IntentConfiguration` + a SiriKit intent (`INIntent`) defined in a `.intentdefinition` file.  
- **iOS 17+:** `AppIntentConfiguration` + an App Intent conforming to `WidgetConfigurationIntent`.

The “AppIntentConfiguration” model aligns widget configuration with the same App Intents stack used for Shortcuts/Siri/Spotlight.

### 4.2 Interactive widgets (buttons powered by App Intents)

Interactive widgets use **App Intents** as the execution model for widget UI elements (e.g., a button in the widget can run an App Intent).

### Sources

- WidgetKit doc — Making a configurable widget: https://developer.apple.com/documentation/widgetkit/making-a-configurable-widget
- WidgetKit doc — Developing a WidgetKit strategy (App Intents + interactivity): https://developer.apple.com/documentation/widgetkit/developing-a-widgetkit-strategy
- WWDC23 — Bring widgets to life (interactive widgets with App Intents): https://developer.apple.com/videos/play/wwdc2023/10028/

---

## 5) Cross-target sharing patterns (“define once, use everywhere”)

### 5.1 Reality check: intents live in bundles

An App Intent is discovered/used based on what’s present in a given bundle. So:

- if a widget needs to execute an intent, the widget extension must include that intent’s code,
- if you want intents to run without launching the full app, you should consider an **App Intents Extension** bundle,
- if the main app should advertise app shortcuts, ensure the bundle that provides `AppShortcutsProvider` is installed and discoverable.

### 5.2 Recommended module split

To keep things reusable across app + extensions:

1. **Domain module (pure logic):** business logic + models; no UI, minimal platform APIs.
2. **Intents module (`IntentsKit`):** App Intents + entities + queries; depends on Domain.
3. **UI modules (optional):** if you use snippet views or UI helpers, isolate SwiftUI-only code to keep extensions safe.

Then link `IntentsKit` into:

- Main app target
- Widget extension target
- App Intents extension target (if you add one)

### 5.3 Packaging support: frameworks, Swift packages, and `AppIntentsPackage`

Apple’s tooling has evolved:

- **iOS 17 / Xcode 15**: frameworks can expose App Intents directly, supported via the `AppIntentsPackage` APIs (WWDC23).  
- **WWDC25**: packaging expands to **Swift packages and static libraries** (WWDC25), which is particularly relevant for Tuist modularization.

### 5.4 App Intents Extension (when to use it)

An App Intents Extension is useful when you want:

- better separation of intent execution from the main app,
- or to make App Shortcuts available **without launching the app in the background** (WWDC23 mentions defining `AppShortcutsProvider` in an App Intents extension for this).

### Sources

- WWDC23 — Explore enhancements to App Intents (framework packaging + AppIntentsPackage + AppShortcutsProvider in App Intents extensions): https://developer.apple.com/videos/play/wwdc2023/10103/
- AppIntents docs — `AppIntentsPackage`: https://developer.apple.com/documentation/appintents/appintentspackage
- WWDC25 — Explore new advances in App Intents (Swift packages + static libraries support): https://developer.apple.com/videos/play/wwdc2025/10124/
- AppIntents docs — App Intents Extension (entry): https://developer.apple.com/documentation/appintents/app-intents-extension

---

## 6) Tuist organization patterns (practical)

Below are practical, “Tuist-friendly” patterns for reusable intent architecture. Pick based on your minimum OS support and how modular you want to be.

### Pattern A (simple): keep intents in each bundle, share only domain logic

- Put `AppIntent` types directly in:
  - the app target
  - the widget extension target (for widget actions/config)
  - the App Intents extension target (if you have one)
- Share a “Domain” module across them.

**Pros:** minimal tooling assumptions.  
**Cons:** intent definitions duplicated across targets (not “define once”).

### Pattern B (recommended): `IntentsKit` shared across bundles

- Create a shared module (Tuist framework target or local Swift package) that contains:
  - App intents
  - app entities + queries
  - app shortcuts provider (only in one place — see note below)
- Link that module into the app + each extension.

**Pros:** “define once, use everywhere” becomes real.  
**Cons:** you must keep the module extension-safe and ensure it’s linked into each bundle that needs it.

**Important note about `AppShortcutsProvider`:** avoid defining multiple providers in multiple bundles unless you intentionally want multiple sets of app shortcuts exposed. Prefer a single provider in either the main app bundle or the App Intents extension bundle.

### Pattern C (2025+ modular): Intent packages per feature

If you’re going heavy on modularization, consider:

- `Domain/*` modules
- `Intents/*` modules per feature (each exports an `AppIntentsPackage`)
- a “root” app/extension module that “includes” all packages (via AppIntentsPackage dependency wiring)

This matches Apple’s direction in WWDC23/WWDC25 for packaging and dependency wiring.

### Sources

- WWDC23 — Explore enhancements to App Intents (framework packaging + AppIntentsPackage): https://developer.apple.com/videos/play/wwdc2023/10103/
- WWDC25 — Explore new advances in App Intents (Swift packages + static libraries packaging): https://developer.apple.com/videos/play/wwdc2025/10124/

---

## 7) Migration notes (SiriKit Intents → App Intents)

If you already shipped SiriKit custom intents (or want to support iOS versions where App Intents aren’t available), treat migration as a compatibility project:

- Use Apple’s migration guidance to preserve existing shortcuts and their parameter schema.
- Keep old intent handlers for older OS versions while introducing App Intents for newer OS versions.

### Sources

- Tech Talks — Migrate custom intents to App Intents: https://developer.apple.com/videos/play/tech-talks/10168/
- `CustomIntentMigratedAppIntent` docs (schema compatibility): https://developer.apple.com/documentation/appintents/customintentmigratedappintent

---

## Fact-checking / Version checkpoints (what changed 2022 → 2025)

- **iOS 16 era (WWDC22):** App Intents introduced as the new code-first model; App Shortcuts designed/implemented via App Intents.  
  - Sources: WWDC22 (Dive into App Intents; Design/Implement App Shortcuts).
- **iOS 17 era (WWDC23):** packaging and composition improved — frameworks can expose App Intents; new `AppIntentsPackage` APIs; and guidance around App Intents extensions (including defining `AppShortcutsProvider` there to avoid background app launches).  
  - Sources: WWDC23 — Explore enhancements to App Intents.
- **2024 (WWDC24):** continued API evolution and system integrations for App Intents (including design guidance for system surfaces and deeper Siri integration).  
  - Sources: WWDC24 sessions listed above.
- **2025 (WWDC25):** packaging expands further to Swift packages + static libraries; Spotlight/Shortcuts integration continues to evolve (including on-screen annotations + entity-driven actions).  
  - Sources: WWDC25 — Explore new advances in App Intents; Develop for Shortcuts and Spotlight with App Intents.

