# EPIC-260227-2qbwda: api-configuration

## Description
Scaffold AppConfigManager + ApiConfigurator pattern: env-based config (prod/stage/dev), protocol composition (IApiConfigManager narrowed per module), struct ApiConfigurator with closure injection for dynamic endpoint resolution. Runtime env switching supported (testers switch prod->dev without restart). Pattern from membrana, adapted: struct instead of open class.

## Scope
(define epic scope)

## Acceptance Criteria
1. AppConfigManager singleton stores env in Keychain, exposes narrow protocols (IApiConfigManager, ISSOConfigManager etc)
2. ApiConfigurator is a struct with resolveConfig closure, not open class
3. Per-module Config inherits ApiConfigurator pattern, defines computed endpoint properties
4. Modules receive IApiConfigManager via IoC constructor injection
5. Env switch in runtime propagates to all modules via lazy closure resolution
6. CLI scaffolds all required files via setup command
7. Generated code compiles in demo app
