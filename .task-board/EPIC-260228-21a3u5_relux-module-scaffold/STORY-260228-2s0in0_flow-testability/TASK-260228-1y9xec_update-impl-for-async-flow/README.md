# TASK-260228-1y9xec: update-impl-for-async-flow

## Description
Update relux_impl template if needed to pass dispatcher to Flow init. Current impl does: let flow = <Name>.Business.Flow(state: state). If Flow init changes to async with optional dispatcher, the impl init may need adjustment.

Check:
1. Does the impl template need to pass nil for dispatcher? No — the default is .none so existing call site works IF we keep state as first param or use labeled args.
2. Verify the call site in relux_impl.swift.tmpl still compiles with the new Flow signature.
3. Update if needed.

Files possibly affected:
- internal/relux/templates/relux_impl.swift.tmpl

Verification:
- make test passes
- Demo app compiles

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
