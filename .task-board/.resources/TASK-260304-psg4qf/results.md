# Widget Base Implementation

Implemented widget-base module and setup flow.

## Files
- internal/widgetbase/register.go
- internal/widgetbase/setup.go
- internal/widgetbase/setup_templates/widget_bundle.swift.tmpl
- internal/widgetbase/setup_test.go
- internal/registry/registry.go
- internal/cli/root.go
- cmd/ios-app-manager/main.go

## Behavior
- Adds  CLI module registration with dependency on .
- Scaffolds extension target at .
- Renders  with .
- Adds WidgetKit dependency to extension target.
- Adds App Groups entitlement to extension target and host app capability entry.
- Embeds extension in host app .
- Adds extension project path to  when present.

## Validation
- go test ./internal/widgetbase
- make test
- make lint
- make build
