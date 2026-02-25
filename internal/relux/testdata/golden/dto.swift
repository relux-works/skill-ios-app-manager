import Foundation

public struct NotesDTO: Identifiable, Equatable, Hashable, Sendable {
    public let id: UUID
    public let title: String

    public init(id: UUID = UUID(), title: String) {
        self.id = id
        self.title = title
    }
}
