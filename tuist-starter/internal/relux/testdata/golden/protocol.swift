import Foundation

public protocol NotesService: Sendable {
    func fetchItems() async throws -> [NotesDTO]
}
