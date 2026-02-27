# SecureStore: KeychainAccess-based abstraction

## Description
Replace raw Security framework SecureStore impl with proper abstraction. Copy KeychainAccess (Keychain.swift, single file from darwin-keychainaccess) into SecureStoreImpl. Enrich ISecureStore protocol to match membrana pattern: String/Data get/set with ignoringAttributeSynchronizable, LAContext reset, Config struct (serviceName, accessibility, authPolicy, cloudSyncEnabled). App depends only on ISecureStore protocol — KeychainAccess is internal to SecureStoreImpl, never exposed.

## Scope
(define epic scope)

## Acceptance Criteria
(define acceptance criteria)
