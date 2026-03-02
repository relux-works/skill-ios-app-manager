# STORY-260302-1cwr9a: capabilities-dsl

## Description
Port Capability DSL + EntitlementsFactory from tuist-akme into generated projects. Replace current plist-based entitlements (7 hardcoded aliases, manual entitlements add/remove) with declarative typed Swift capabilities in ProjectDescriptionHelpers/. During init, generate Capability.swift + EntitlementsFactory.swift + Capability+PortalCapability.swift (118 Apple portal capabilities) + project-specific AppCapabilities.swift from config. Update Project.swift template to use capabilities DSL instead of .file() entitlements. Deprecate entitlements add/remove CLI commands (keep list as read-only verification). Source: tuist-akme TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
