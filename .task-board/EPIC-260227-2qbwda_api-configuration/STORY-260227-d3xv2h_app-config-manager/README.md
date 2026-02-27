# STORY-260227-d3xv2h: app-config-manager

## Description
CLI command (app-config setup or similar) that scaffolds AppConfigManager module: Env enum (prod/stage/dev) with Keychain persistence, Configuration model with Api/SSO/SupportChat sub-configs, protocol composition IAppConfigManager = IApiConfigManager + ISSOConfigManager + ISupportChatsConfigManager, IoC registration of all narrow protocols resolving to same singleton. Manager reads env from Keychain on init, caches it, exposes updateEnvConfig for runtime switching.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
