# STORY-260227-1tgzwm: http-client-utils

## Description
HttpClientUtils submodule inside Utilities package.

Contents:
- Standard HTTP header maps (JSON content-type, accept, form-urlencoded, auth header builder)
- Base JSONEncoder: snakeCase keys, ISO8601 date strategy, configured for API communication
- Base JSONDecoder: matching config (snakeCase keys, ISO8601 dates)
- Possibly: common HTTP status code helpers, request/response logging utilities

Module type: utility (single package, no interface/impl split).
Depends on: IoC only.
Used by: feature relux-modules with backend (injected or imported directly).

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
