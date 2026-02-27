# Configuration shared config folder

## Description
Configuration — shared config folder inside the main app target sources.
Path: Targets/<AppName>/Sources/Configuration/

Not a separate package, not a separate target. Just a folder with Swift files compiled as part of the app target. Contains app-wide constants organized as nested enums under Configuration namespace.

Structure:
  Targets/XFlow/Sources/Configuration/
    Configuration.swift          — public enum Configuration {} + doc comment that this is shared config
    HttpClient/
      Configuration+HttpClient.swift  — extension Configuration { enum HttpClient { ... } }
    Api/
      Configuration+Api.swift         — (future) extension Configuration { enum Api { ... } }

Created by init as empty namespace. Setup commands add nested extensions (e.g. http-client setup adds Configuration+HttpClient.swift).

## Scope
Configuration folder scaffolded at init in app target sources

## Acceptance Criteria
1. init creates Targets/<AppName>/Sources/Configuration/Configuration.swift
2. Namespace: enum Configuration {} with doc comment
3. Setup commands can add extension files
4. make test green
