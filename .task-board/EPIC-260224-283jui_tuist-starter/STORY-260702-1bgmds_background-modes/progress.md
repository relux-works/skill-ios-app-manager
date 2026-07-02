# Progress

## Status
done (2026-07-02)

## Checklist
- [x] Story created before implementation
- [x] Config field + validation
- [x] Template rendering
- [x] Sync generator + registration
- [x] Tests green (`make test`, `go vet`)
- [x] Docs updated (cli-reference, tree-tuist)
- [x] Demo app rebuilt and compiles; built Info.plist carries UIBackgroundModes

## Notes
Driven from the Memori integration (issue #6). Capability side needs no work:
`.portal(.pushToTalk)` / `.portal(.pushNotifications, environment:)` already
express the entitlements via the Xcode portal catalog.

Pre-existing demo-pipeline breakages observed while verifying (NOT caused by
this story, each reproduces without it):
- `swift-httpclient` checkout fails under MemberImportVisibility (missing
  `import Combine` in PublishedWSClient) — pulled in via http-client module and
  the Auth blueprint.
- Widget/App Intents template `XFlowWidgetToggleIntent.swift` fails Swift 6
  strict concurrency (`static var title` mutable global state).
- `min_target` with a nonzero minor (e.g. 17.6) renders `.iOS(.v17_6)` in
  module Package.swift manifests, which does not compile.
The minimal pipeline (init + ioc + relux + utilities + foundation-plus +
swiftui-plus + app-config) builds green with UIBackgroundModes rendered.
