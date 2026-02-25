import Foundation
import NotesInterface

public enum NotesAction: Sendable, Equatable {
    case appeared
    case setLoading(Bool)
    case setItems([NotesDTO])
    case setError(String?)
    case forward(NotesPublicAction)
}
