import Foundation
import NotesInterface

public struct NotesState: Equatable, Sendable {
    public static let moduleKey = "notes"

    public var isLoading: Bool
    public var items: [NotesDTO]
    public var errorMessage: String?

    public init(
        isLoading: Bool = false,
        items: [NotesDTO] = [],
        errorMessage: String? = nil
    ) {
        self.isLoading = isLoading
        self.items = items
        self.errorMessage = errorMessage
    }

    public static let initial = NotesState()
}
