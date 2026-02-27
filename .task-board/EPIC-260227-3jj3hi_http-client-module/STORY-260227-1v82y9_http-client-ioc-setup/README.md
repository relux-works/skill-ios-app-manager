# STORY-260227-1v82y9: http-client-ioc-setup

## Description
CLI command http-client setup that adds swift-httpclient external dep and registers HTTPClient in IoC Registry.

No separate module packages — just dependency + registration.

Scope:
- internal/httpclient/setup.go — Setup() adds external dep, updates manifests, patches Registry.swift
- internal/cli/http_client.go — CLI command wiring
- Registry template update: support direct type registrations (not Module.Interface pattern) in Foundation section

AC:
- http-client setup adds swift-httpclient to Package.swift and Project.swift
- Registry.swift gets import SwiftHTTPClient + HTTPClient registration in Foundation
- Idempotent
- Tests pass, demo builds

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
