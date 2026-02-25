import Foundation

public enum NotesPublicAction: Sendable, Equatable {
    case refresh
    case select(id: UUID)
    case track(source: String = "notes")
}
