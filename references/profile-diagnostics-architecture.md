# Profile Diagnostics Architecture

## Command Surface

```bash
ios-app-manager profile build [flags]
ios-app-manager profile layout scaffold [flags]
ios-app-manager profile layout analyze --input <xml-or-log> [flags]
ios-app-manager profile runtime scaffold [flags]
ios-app-manager profile runtime analyze --input <log> [flags]
```

## Package Layout

```text
tuist-starter/internal/profile/
  build.go              # xcodebuild runner and build report model
  build_parser.go       # timing summary parser
  graph.go              # Tuist graph parser and critical path
  layout.go             # rendered layout XML parser and diagnostics
  layout_probe_template.go # XCTest LayoutHierarchyProbe.swift writer
  runtime.go            # runtime log parser and report model
  runtime_probe_template.go # PerformanceProbe.swift writer

tuist-starter/internal/cli/profile.go
  Cobra command wiring and report rendering
```

## Data Flow

Build profiling:

1. Resolve config and project root.
2. Generate project unless disabled.
3. Capture Tuist target graph.
4. Run or load `xcodebuild -showBuildTimingSummary`.
5. Parse timing entries.
6. Join target timing with graph.
7. Render text/JSON report.

Runtime profiling:

1. Scaffold debug-only helper.
2. Developer wraps suspicious views/functions.
3. App emits `IAM_PROFILE` structured lines and signposts.
4. CLI parses captured logs.
5. Report highlights hot views/functions and blocking main-thread calls.

Rendered layout diagnostics:

1. Scaffold XCTest helper into the UI test target.
2. UI test launches the app and calls `attachLayoutHierarchyXML(app, ...)`.
3. Helper emits XML from the rendered accessibility tree and stores it as an `.xcresult` attachment.
4. CLI reads raw XML or logs with `IAM_LAYOUT_XML_*` markers.
5. Report prints an agent-readable tree and flags duplicate identities, missing interactive labels/ids, tiny tap targets, and offscreen elements.

## Design Principles

- Local first. Reports must work without Tuist Cloud or any hosted backend.
- Deterministic artifacts. Build runs write into `.temp/build-profile/`.
- Optional integrations. Tuist Build Insights, `xctrace`, and MetricKit are complementary, not required.
- Debug-only runtime instrumentation. Generated code uses `#if DEBUG`; no private API dependency.
- Rendered hierarchy dumps use public XCTest/accessibility APIs; no private UIKit/SwiftUI tree scraping.
- Machine-readable output. JSON reports must be stable enough for CI and future dashboards.
