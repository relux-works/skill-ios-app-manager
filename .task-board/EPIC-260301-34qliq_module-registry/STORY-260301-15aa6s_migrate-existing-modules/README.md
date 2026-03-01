# STORY-260301-15aa6s: migrate-existing-modules

## Description
Migrate all 7 existing setup modules to use registry.Register().

Modules to migrate:
1. ioc (internal/ioc/) → registry.IoC
2. relux (internal/relux/) → registry.Relux
3. secure-store (internal/securestore/) → registry.SecureStore
4. token-provider (internal/tokenprovider/) → registry.TokenProvider
5. http-client (internal/httpclient/) → registry.HttpClient
6. app-config (internal/appconfig/) → registry.AppConfig
7. utilities (internal/utilities/) → registry.Utilities

For each:
- Add init() with registry.Register() call
- Move SetupInput to use registry.SetupInput (or adapt)
- Add stub PlanFunc and UsageGuide (placeholder text like: TODO: add usage instructions for SecureStore)
- Declare Dependencies (e.g. SecureStore depends on IoC, AppConfig depends on IoC+SecureStore)
- Set Category (infra/foundation/network/utils)

CLI commands (cli/*.go) become thin wrappers that resolve from registry.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
