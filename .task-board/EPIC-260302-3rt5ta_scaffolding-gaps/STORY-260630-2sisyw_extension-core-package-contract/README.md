# Extension Core package contract

## Description
Introduce the canonical extension scaffold pattern. Each concrete extension remains its own plugin, like current widget plugins, but generated extension internals must live in a dedicated SwiftPM package named <ExtensionName>Core. The .appex target should be a thin composition/runtime wrapper that links the Core package. Business logic, models, parsers, payload transformers, timeline builders, and other testable internals belong in the Core package, because app extension targets are a poor direct test host.

## Scope
Update extension-base/app-extensions architecture, extension plugin registry, templates, package wiring, generated Project.swift shape, docs, and tests. The scaffold must be able to create and update extension Core packages idempotently. Existing widget plugins must be migrated to the same pattern in a dependent story. Notification Service Extension must be built with this pattern in its own story.

## Acceptance Criteria
Every new extension plugin has a generated <ExtensionName>Core SwiftPM package; the extension target links the Core product and keeps only app-extension runtime entrypoint code; the Core package has a generated test target or is immediately compatible with the new test-target scaffold; rerunning the extension plugin does not duplicate Package.swift entries, target dependencies, generated files, or Project.swift declarations; changing supported extension config updates the wrapper and Core package without losing handwritten extension logic outside owned regions; SKILL.md documents that extension internals must be in Core packages and tested at package level.
