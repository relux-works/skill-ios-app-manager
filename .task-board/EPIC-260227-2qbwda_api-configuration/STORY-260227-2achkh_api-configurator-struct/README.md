# STORY-260227-2achkh: api-configurator-struct

## Description
Scaffold ApiConfigurator struct pattern: struct with resolveConfig closure field (@Sendable () -> Configuration.Api), UrlComponents enum for shared path segments. Per-module Config pattern: struct conforming to ApiConfigurator storing the closure, computed endpoint properties that call resolveConfig() lazily. Template generates example Config for a module Fetcher. Wired into module create so new relux-feature/kit modules get a Data/Api/ folder with Fetcher + Config scaffolded.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
