# Registry support for direct type registration

## Description
Registry template currently only supports Module.Interface pattern. HttpClient needs direct type registration: ioc.register(IRpcAsyncClient.self, ...) with a custom builder.

Approach: add DirectRegistrations to RegistryTemplateData — list of {Import, TypeName, BuilderName, Section} — rendered in the appropriate section.

Or: use anchor-based post-render insertion into Foundation section.

Prefer RegistryTemplateData approach — cleaner, template handles rendering.

AC: Registry.swift can render direct type registrations alongside Module.Interface ones in any section.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
