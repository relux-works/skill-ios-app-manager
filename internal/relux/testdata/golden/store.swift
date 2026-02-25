import Foundation
import Observation

@MainActor
@Observable
public final class NotesStore {
    public private(set) var state: NotesState

    private let reducer: NotesReducer
    private let middleware: any NotesMiddlewareProtocol
    private var middlewareTask: Task<Void, Never>?

    public init(
        initialState: NotesState = .initial,
        reducer: NotesReducer = .init(),
        middleware: any NotesMiddlewareProtocol
    ) {
        self.state = initialState
        self.reducer = reducer
        self.middleware = middleware
    }

    deinit {
        middlewareTask?.cancel()
    }

    public func dispatch(_ action: NotesAction) {
        state = reducer.reduce(state: state, action: action)

        middlewareTask?.cancel()
        middlewareTask = Task { [middleware, state] in
            if let followUpAction = await middleware.handle(action: action, state: state) {
                await MainActor.run { [weak self] in
                    self?.dispatch(followUpAction)
                }
            }
        }
    }
}
