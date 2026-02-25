import Foundation

public struct NotesReducer: Sendable {
    public init() {}

    public nonisolated func reduce(state: NotesState, action: NotesAction) -> NotesState {
        var nextState = state

        switch action {
        case .appeared:
            nextState.isLoading = true
            nextState.errorMessage = nil
        case .setLoading(let isLoading):
            nextState.isLoading = isLoading
        case .setItems(let items):
            nextState.isLoading = false
            nextState.items = items
            nextState.errorMessage = nil
        case .setError(let message):
            nextState.isLoading = false
            nextState.errorMessage = message
        case .forward:
            break
        }

        return nextState
    }
}
