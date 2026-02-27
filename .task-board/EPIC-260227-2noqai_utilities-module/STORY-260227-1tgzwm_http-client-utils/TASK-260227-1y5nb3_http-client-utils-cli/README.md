# TASK-260227-1y5nb3: http-client-utils-cli

## Description
Add Utilities module scaffolding to CLI.

1. New package internal/utilities/ following existing module patterns
2. setup.go — create utility module dir, render templates
3. Templates for HttpClientUtils submodule:
   - Utilities/Sources/Utilities/HttpClientUtils/HeaderMaps.swift — standard header dictionaries (jsonHeaders, formHeaders, authHeader builder func)
   - Utilities/Sources/Utilities/HttpClientUtils/BaseEncoder.swift — JSONEncoder with snakeCase keys, ISO8601 dates
   - Utilities/Sources/Utilities/HttpClientUtils/BaseDecoder.swift — JSONDecoder with matching config
4. CLI command: utilities setup in internal/cli/utilities.go
5. Register in root command tree
6. Module type: utility (single package, no interface/impl split)

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
