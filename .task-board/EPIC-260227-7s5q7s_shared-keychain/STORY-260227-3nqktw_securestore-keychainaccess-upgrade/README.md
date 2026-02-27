# STORY-260227-3nqktw: securestore-keychainaccess-upgrade

## Description
Replace raw Security framework SecureStore with KeychainAccess-based implementation. Copy Keychain.swift from membrana darwin-keychainaccess into SecureStoreImpl. Enrich ISecureStore protocol (String/Data get/set, ignoringAttributeSynchronizable, LAContext, Config struct). SecureStore init checks/adds shared keychain group from previous story.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
