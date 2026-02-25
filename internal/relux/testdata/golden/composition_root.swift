import SwiftIoC

extension Notes {
    @MainActor
    enum CompositionRoot {
        static let ioc = IoC(logger: IoC.Logger(enabled: false))

        static func configure() {
            AuthModule.Registry.register(in: ioc)
            NetworkModule.Registry.register(in: ioc)
            Notes.Registry.register(in: ioc)

            ioc.register(NotesService.self, lifecycle: .container, resolver: {
                NotesServiceImpl()
            })
        }

        static func resolve<T>(_ type: T.Type) -> T {
            ioc.get(by: type)!
        }

        static func resolveAsync<T: Sendable>(_ type: T.Type) async -> T where T.Type: Sendable {
            await ioc.getAsync(by: type)!
        }

        static func optionalResolve<T>(_ type: T.Type) -> T? {
            ioc.get(by: type)
        }

        static func optionalResolveAsync<T: Sendable>(_ type: T.Type) async -> T? where T.Type: Sendable {
            await ioc.getAsync(by: type)
        }
    }
}
