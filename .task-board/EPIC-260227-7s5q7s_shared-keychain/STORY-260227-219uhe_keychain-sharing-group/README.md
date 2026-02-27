# STORY-260227-219uhe: keychain-sharing-group

## Description
Add keychain-sharing-group entitlement scaffolding. Format: $(AppIdentifierPrefix)<bundleId>.shared — CLI command adds the group to entitlements plist. Constants for service name and group go into Configuration namespace. This must be done BEFORE SecureStore upgrade because SecureStore will depend on the shared group being present.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
