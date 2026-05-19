# Rendered Layout Hierarchy Diagnostics Research

## Goal

Give agents a machine-readable view of what actually rendered on an iOS simulator/device:

- rendered accessibility hierarchy;
- element type, path, identifier/name/label/value;
- screen-coordinate frames;
- agent-facing diagnostics for duplicate identities, missing accessibility identity, tiny tap targets, and offscreen elements.

## Primary Sources

- Apple `XCUIElement.debugDescription`: can include descendant tree and attributes, but Apple marks the string as debugging-only and unsupported for test dependencies.
- Apple `XCUIElementAttributes`: exposes accessibility-backed attributes available during UI query matching, including identifier, label, value, element type, and frame.
- Apple `XCUIApplication`: is the UI-test application proxy and inherits `XCUIElement`, so it is a stable root for a rendered hierarchy dump after launch.
- Apple `XCUIElementQuery.descendants(matching:)` / `children(matching:)`: public XCTest query APIs for traversing currently observable UI elements.
- Appium/WebDriverAgent page source commonly serializes the iOS accessibility tree as XML with `XCUIElementType*` tags and `x/y/width/height` attributes.

## Findings

- There is no stable first-party CLI command that dumps the full iOS rendered view hierarchy as XML from an arbitrary running app.
- XCTest exposes the rendered accessibility tree through public APIs. That tree is not the private UIKit/SwiftUI view graph, but it is the right agent-facing surface because it matches what UI tests, accessibility, and many device automation tools can inspect.
- `debugDescription` is useful for manual debugging only. Depending on its exact text format would be brittle.
- Appium/WDA XML and a custom XCTest XML dump can share one analyzer if the parser normalizes:
  - tag/type names such as `XCUIElementTypeButton` to `button`;
  - `identifier` and Appium `name` into an effective identity;
  - frame attributes into screen-coordinate rectangles;
  - explicit `visible`, `enabled`, `hittable`, and `accessible` booleans when present.

## MVP Architecture

Add `ios-app-manager profile layout` with two subcommands:

1. `profile layout scaffold`
   - writes `LayoutHierarchyProbe.swift` into `Targets/<AppName>UITests/Sources/Diagnostics/` by default;
   - helper is `#if DEBUG` and imports XCTest;
   - helper serializes `XCUIApplication` + recursive `children(matching: .any)` to XML;
   - helper attaches XML to the `.xcresult` and prints it between `IAM_LAYOUT_XML_START` / `IAM_LAYOUT_XML_END` markers for log extraction.

2. `profile layout analyze --input <xml-or-log>`
   - accepts raw `LayoutHierarchyProbe` XML, Appium/WDA XML, or logs containing IAM layout markers;
   - emits a text tree for humans/agents and stable JSON for future dashboards;
   - reports type counts, max depth, duplicate identities, missing interactive identity, tiny tap targets, and offscreen frames.

Generated usage example:

```swift
final class FeedUITests: XCTestCase {
    @MainActor
    func testFeedLayoutDump() {
        let app = XCUIApplication()
        app.launch()
        attachLayoutHierarchyXML(app, name: "feed", screenName: "Feed")
    }
}
```

CLI usage:

```bash
ios-app-manager profile layout scaffold
ios-app-manager profile layout analyze --input .temp/layout/feed.xml
ios-app-manager profile layout analyze --input .temp/layout/ui-test.log --format json
```

## Non-Goals For MVP

- pixel-level screenshot OCR;
- private UIKit/SwiftUI view graph dumping;
- automatic UI test target creation;
- replacing visual screenshot review.

## Follow-Up

- Add `profile layout extract` for pulling XML attachments from `.xcresult`.
- Add optional screenshot pairing so agents can cross-reference XML nodes with pixels.
- Add overlap diagnostics after tuning false positives for nested controls.
- Add Appium source collection when a local WDA/Appium endpoint is already running.
