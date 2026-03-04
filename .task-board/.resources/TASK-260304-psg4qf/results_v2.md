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
- Adds widget-base CLI module registration with dependency on app-extensions.
- Scaffolds extension target at Extensions/<AppName>Widget/.
- Renders <AppName>WidgetBundle.swift with @main WidgetBundle.
- Adds WidgetKit dependency to extension target.
- Adds App Groups entitlement to extension target and host app capability entry.
- Embeds extension in host app Project.swift.
- Adds extension project path to Workspace.swift when present.

## Validation
- go test ./internal/widgetbase
- make test
- make lint
- make build
