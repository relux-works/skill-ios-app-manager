# STORY-260302-rp45y3: http-client-middleware

## Description
[A1 HIGH, Tier 1] Redesign http-client setup from empty IoC stub to middleware-based networking layer. Current connect-ios NetworkManager is 700+ lines with auth injection, token refresh with retry guard, suspension detection (423/451), multipart upload, custom date decoding — all copy-pasted across 4 request methods. Generate middleware/interceptor chain: base request methods + composable interceptors (auth header, token refresh, error/suspension detection, logging). Each interceptor is a separate file. Covers the biggest scaffolding gap — every feature depends on networking.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
