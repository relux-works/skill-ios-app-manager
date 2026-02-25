import Foundation
import SwiftIoC
import NotesInterface
import NotesImpl

public enum NotesModuleRegistration {
    public static func register(
        into ioc: IoC,
        serviceFactory: @escaping @Sendable () -> any NotesService
    ) {
        ioc.register((any NotesService).self, lifecycle: .container) {
            serviceFactory()
        }

        ioc.register((any NotesMiddlewareProtocol).self, lifecycle: .container) {
            let service = ioc.get(by: (any NotesService).self)
                ?? serviceFactory()
            return NotesMiddleware(service: service)
        }

        ioc.register(NotesStore.self, lifecycle: .container) {
            let middleware = ioc.get(by: (any NotesMiddlewareProtocol).self)
                ?? NotesMiddleware(service: serviceFactory())
            return NotesStore(middleware: middleware)
        }
    }
}
