# STORY-260301-3mmztn: registry-package

## Description
Create internal/registry/ package — the shared lightweight module that all scaffolding packages import.

Contents:
- ModuleID type + constants for all existing modules (ioc, relux, secure-store, token-provider, http-client, app-config, utilities, defaults-store)
- Module struct with fields: ID, Name, Description, Dependencies []ModuleID, Category string, SetupFunc, PlanFunc, UsageGuide string
- Global registry: Register(), Get(), All(), DependenciesOf()
- SetupInput struct (currently duplicated across packages — consolidate here)
- Common utilities that currently duplicate across module packages (e.g. registry patching helpers, file scaffolding helpers)

This package must be lightweight — no heavy deps. Every module package will import it.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
