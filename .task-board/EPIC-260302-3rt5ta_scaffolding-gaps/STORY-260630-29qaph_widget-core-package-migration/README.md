# Widget Core package migration

## Description
Refactor widget-base and concrete widget plugins to use the extension Core package pattern. WidgetBundle/appex entrypoint code remains in the extension wrapper; timeline providers, entries, views that can be unit-tested, intent adapters, ActivityAttributes helpers, and shared widget logic move into generated Core packages.

## Scope
Update widget-base, static-widget, configurable-widget, app-intents widget integration, and live-activity where they generate extension internals. Preserve current commands while changing generated layout. Add migration/idempotency tests so rerunning old widget setup converges to the new structure without duplicate registrations.

## Acceptance Criteria
Widget extension target links a generated widget Core package; static widget internals are generated into the Core package; configurable/app-intents widget internals use the same pattern where applicable; live activity scaffolding keeps ActivityKit runtime glue thin and moves testable internals to Core/shared packages; rerunning widget commands after migration is idempotent; existing widget registration into WidgetBundle remains valid; docs explain the new widget extension layout.
