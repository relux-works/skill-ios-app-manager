# TASK-260227-2c7kje: tests-and-verification

## Description
Tests:
- internal/httpclient/setup_test.go — unit tests for Setup()
- internal/cli/http_client_test.go — integration test
- Verify Package.swift has swift-httpclient dep
- Verify Project.swift has SwiftHTTPClient
- Verify Registry.swift has HTTPClient registration in Foundation
- Update e2e pipeline to include http-client setup

AC: make test green, demo pipeline builds with http-client setup included.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
