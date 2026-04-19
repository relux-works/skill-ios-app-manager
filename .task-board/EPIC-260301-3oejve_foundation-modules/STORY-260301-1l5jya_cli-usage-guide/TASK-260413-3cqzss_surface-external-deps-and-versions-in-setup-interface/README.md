# TASK-260413-3cqzss: surface-external-deps-and-versions-in-setup-interface

## Description
Expose each setup plugin's external Swift package dependencies directly in the user-facing plan/interface before confirmation so it is immediately readable which package/product/version will be added. Scope: registry-backed setup commands should render dependency summary lines with package name, product name, source URL, and version selector (from/exact/branch/revision as applicable) for modules like ioc, relux, http-client and relux-feature blueprint wiring. AC: setup command output shows external deps and versions before apply; tests cover representative modules; docs/help text reflect the behavior.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
