# EPIC-260227-2noqai: utilities-module

## Description
Utilities — utility module (type: utility, single package, no interface/impl split). Contains shared utility submodules used across foundation and feature layers.

First submodule: HttpClientUtils
- Standard HTTP header maps (e.g. jsonHeaders, formHeaders, authHeader builder)
- Base JSONEncoder with default config (snakeCase, ISO8601 dates, etc.)
- Base JSONDecoder with matching config
- No dependencies except IoC registration

HttpClientUtils is a dependency of feature relux-modules with backend (alongside HttpClient, TokenProvider, ApiConfigurator).

Future submodules can be added as the project grows.

## Scope
(define epic scope)

## Acceptance Criteria
(define acceptance criteria)
