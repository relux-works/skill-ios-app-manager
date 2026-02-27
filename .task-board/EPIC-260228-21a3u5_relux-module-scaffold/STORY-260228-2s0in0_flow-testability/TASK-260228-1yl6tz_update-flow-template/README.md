# TASK-260228-1yl6tz: update-flow-template

## Description
Update internal/relux/templates/relux_flow.swift.tmpl to add optional dispatcher injection for testability. Current Flow actor takes only State as dependency.

New pattern (from .research/260228_relux_module_internal_layout.md Authoritative Architecture > Testing Pattern):

actor Flow: Relux.Flow {
    let dispatcher: Relux.Dispatcher
    private let state: <Name>.Business.State

    init(
        dispatcher: Relux.Dispatcher? = .none,
        state: <Name>.Business.State
    ) async {
        let defaultDispatcher = await Self.defaultDispatcher
        self.dispatcher = dispatcher ?? defaultDispatcher
        self.state = state
    }

    func apply(_ effect: any Relux.Effect) async -> Relux.ActionResult {
        guard let effect = effect as? <Name>.Business.Effect else { return .success }
        return await internalApply(effect)
    }

    private func internalApply(_ effect: <Name>.Business.Effect) async -> Relux.ActionResult {
        switch effect {}
    }
}

Key changes:
1. Add dispatcher property (let, not private — tests need access)
2. Init takes optional Relux.Dispatcher? = .none
3. Resolve default dispatcher via await Self.defaultDispatcher
4. Init becomes async (already is)

Files to modify:
- internal/relux/templates/relux_flow.swift.tmpl

Verification:
- make test passes
- Demo app compiles after regeneration

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
