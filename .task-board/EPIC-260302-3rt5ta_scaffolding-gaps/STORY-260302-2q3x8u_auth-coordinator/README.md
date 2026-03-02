# STORY-260302-2q3x8u: auth-coordinator

## Description
[A4 MEDIUM, Tier 2] Scaffold AuthCoordinator with completeLogin(tokens:userId:) and SessionManager with clearAll(). Each setup command (secure-store, token-provider) registers its cleanup step. Login and logout become symmetric single-method calls. Prevents 3-way cleanup duplication and the bug class where one cleanup site diverges. New auth setup command or extend relux setup.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
