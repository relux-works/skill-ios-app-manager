# Runtime Profiling Diagnostics Research

## Goal

Provide a local workflow for app runtime profiling:

- identify SwiftUI subtrees whose `body` is evaluated too often;
- identify slow or repeated function calls;
- distinguish main-thread blocking work from background work;
- emit local reports that can be used without a hosted analytics platform.

## Primary Sources

- Apple's "Demystify SwiftUI performance" explains the feedback loop: measure, identify cause, optimize, and re-measure. It highlights SwiftUI dependency scoping, `Self._printChanges()` for debugging body updates, expensive work in `body`, expensive dynamic property instantiation, list identity, and constant row counts.
- `Self._printChanges()` is a debug-only SwiftUI facility. Apple explicitly warns that the underscore API is not guaranteed and must not ship to the App Store.
- `OSSignposter` records signposted intervals and events through unified logging; Instruments can display those intervals on a timeline.
- MetricKit can deliver daily metrics and immediate diagnostics for crashes/hangs, and supports custom signpost metrics. It is useful for long-term app telemetry, not quick local repro loops.
- `xcrun xctrace` can record templates such as `SwiftUI`, `Time Profiler`, `Animation Hitches`, `Swift Concurrency`, and `Logging`, then export trace data as XML.

## Findings

- There is no stable public API that globally counts every SwiftUI view render from a CLI. Instruments can do it interactively or through `xctrace`, but automated parsing varies by template/model version.
- A practical local workflow needs explicit debug instrumentation:
  - a lightweight helper for `View` body/subtree counters;
  - a helper for synchronous and async function intervals;
  - structured console/unified-log output that the CLI can parse.
- `Self._printChanges()` is useful during manual debugging, but because it is private/debug-only, generated helper code should not depend on it by default.
- Signposts are the right bridge to Instruments; structured `IAM_PROFILE` lines are the right bridge to deterministic CLI reports and tests.

## MVP Architecture

Add `ios-app-manager profile runtime` with two subcommands:

1. `profile runtime scaffold`
   - writes a debug-only `PerformanceProbe.swift` helper into `Targets/<AppName>/Sources/Diagnostics/`;
   - helper provides:
     - `PerformanceProbe.measure(_:)` sync intervals;
     - `PerformanceProbe.measureAsync(_:)` async intervals;
     - `ProfiledView` and `View.profiled(_:)` for SwiftUI subtree/body counters;
     - `os_signpost` intervals/events for Instruments;
     - `IAM_PROFILE {json}` debug lines for CLI parsing.

2. `profile runtime analyze --input <log>`
   - parses `IAM_PROFILE` lines from console/log captures;
   - groups by name and kind;
   - reports count, total duration, average, max, main-thread slow calls, and excessive repeated calls.

Generated usage example:

```swift
var body: some View {
    content
        .profiled("FeedScreen")
}

let items = PerformanceProbe.measure("Feed.filter") {
    allItems.filter { $0.isVisible }
}

let payload = await PerformanceProbe.measureAsync("Feed.refresh") {
    await service.refresh()
}
```

## Non-Goals For MVP

- automatic source rewriting;
- shipping production analytics;
- relying on private SwiftUI APIs;
- replacing Instruments.

## Follow-Up

- Add `profile runtime capture` wrapping `log stream` by subsystem/category.
- Add optional `xctrace record --template SwiftUI|Time Profiler|Animation Hitches`.
- Add a Tuist generator for a reusable diagnostics package if multiple apps need the same helper.
