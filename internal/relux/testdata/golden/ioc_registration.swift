import SwiftIoC

extension Notes {
    @MainActor
    enum Registry {
        static func register(in ioc: IoC) {
            // Interface package exposes protocol; implementation package binds concrete type.
            ioc.register(NotesService.self, lifecycle: .container, resolver: {
                NotesServiceImpl(
                    ioc.get(by: AuthModule.self)!,
                    ioc.get(by: NetworkModule.self)!,
                )
            })
        }
    }
}
