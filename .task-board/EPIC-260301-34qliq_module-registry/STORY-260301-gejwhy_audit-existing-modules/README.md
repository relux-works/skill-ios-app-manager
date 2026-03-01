# STORY-260301-gejwhy: audit-existing-modules

## Description
Before building the registry, audit all 7 existing setup modules to verify the registry design fits them all.

For each module (ioc, relux, secure-store, token-provider, http-client, app-config, utilities):
- What does Setup() do? (files created, files patched, external deps added)
- What are its real prerequisites? (which other modules must exist)
- What SetupInput fields does it use? (can we unify?)
- What would Plan() output look like?
- What would UsageGuide contain?
- Any special flags (e.g. secure-store --access-group)?
- Does it fit the Module struct cleanly or needs exceptions?

Output: design adjustment doc — confirm registry design works OR propose changes.
This is a research task, no code.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
