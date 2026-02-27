# TASK-260227-35ag3p: secure-store-cli-command

## Description
Add secure-store setup command to CLI (Go code).

- New package internal/securestore/ following internal/ioc/ pattern
- setup.go — orchestrator: create kit module dirs, render templates, update IoC registry
- setup_templates/*.tmpl — Swift templates for SecureStoring protocol and KeychainSecureStore actor impl
- CLI command secure-store setup in internal/cli/secure_store.go
- Register command in root command tree

Templates to generate:
1. SecureStore/Sources/SecureStore/SecureStore.swift — namespace enum
2. SecureStore/Sources/SecureStore/Module/SecureStore.Module.swift — module definition
3. SecureStore/Sources/SecureStore/Module/SecureStore.Module+Interface.swift — SecureStoring protocol
4. SecureStoreImpl/Sources/SecureStoreImpl/Module/SecureStore.Module+Impl.swift — KeychainSecureStore actor
5. Update IoC registration/resolver with SecureStoring binding

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
