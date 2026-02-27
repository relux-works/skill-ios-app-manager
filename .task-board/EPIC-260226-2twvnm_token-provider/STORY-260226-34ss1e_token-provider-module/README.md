# STORY-260226-34ss1e: token-provider-module

## Description
Create TokenProvider as a feature package with interface/impl split. Interface package exposes protocol with setAuthData and getAccessToken.

AuthData model: accessToken (String), refreshToken (String), acquireDate (Date), ttl (TimeInterval).

setAuthData stores the full AuthData struct AND persists it via SecureStore (SecureStoring protocol). getAccessToken returns the current access token with concurrency safety — loads from SecureStore if in-memory state is nil (cold start recovery).

Impl package provides actor-based implementation with reentrancy safety. Generated files follow Module/ subdirectory convention. Includes detailed documentation explaining the module purpose: isolated concurrent access to access tokens.

DEPENDENCY: TokenProvider depends on SecureStore module (SecureStoring protocol for persistent token storage). SecureStore must be scaffolded first.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
