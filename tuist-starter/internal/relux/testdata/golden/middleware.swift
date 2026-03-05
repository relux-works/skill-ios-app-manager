import Foundation
import NotesInterface

public protocol NotesMiddlewareProtocol: Actor {
    func handle(action: NotesAction, state: NotesState) async -> NotesAction?
}

public actor NotesMiddleware: NotesMiddlewareProtocol {
    private let service: any NotesService

    public init(service: any NotesService) {
        self.service = service
    }

    public func handle(action: NotesAction, state: NotesState) async -> NotesAction? {
        switch action {
        case .appeared:
            do {
                let items = try await service.fetchItems()
                return .setItems(items)
            } catch {
                return .setError(String(describing: error))
            }
        case .setLoading, .setItems, .setError, .forward:
            return nil
        }
    }
}
