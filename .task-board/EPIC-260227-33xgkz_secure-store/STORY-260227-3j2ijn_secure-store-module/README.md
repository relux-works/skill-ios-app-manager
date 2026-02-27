# STORY-260227-3j2ijn: secure-store-module

## Description
Create SecureStore as a kit module with interface/impl split.

Interface package (SecureStore):
- SecureStoring protocol: save(key: String, data: Data), load(key: String) -> Data?, delete(key: String), clear()
- Generic convenience: save<T: Codable>(key:, value:), load<T: Codable>(key:) -> T?

Impl package (SecureStoreImpl):
- KeychainSecureStore actor implementing SecureStoring
- Uses Security framework (SecItemAdd, SecItemCopyMatching, SecItemDelete, SecItemUpdate)
- Service name scoped to app bundle ID
- kSecClassGenericPassword based

Module type: kit. Generated via ios-app-manager module create SecureStore --type kit, then overwrite templates with SecureStore-specific Swift code.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
