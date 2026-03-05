import SwiftIoC

extension Notes {
    @MainActor
    enum Resolver {
        static func optionalResolve<T>(_ type: T.Type, from ioc: IoC) -> T? {
            ioc.get(by: type)
        }

        static func resolve<T>(_ type: T.Type, from ioc: IoC) -> T {
            ioc.get(by: type)!
        }

        static func optionalResolveAsync<T: Sendable>(_ type: T.Type, from ioc: IoC) async -> T? where T.Type: Sendable {
            await ioc.getAsync(by: type)
        }

        static func resolveAsync<T: Sendable>(_ type: T.Type, from ioc: IoC) async -> T where T.Type: Sendable {
            await ioc.getAsync(by: type)!
        }

        @discardableResult
        static func waitForResolve<T: Sendable>(_ type: T.Type, from ioc: IoC) async -> T where T.Type: Sendable {
            await ioc.waitForResolve(type)
        }

        static func resolveService(from ioc: IoC) -> NotesService {
            resolve(NotesService.self, from: ioc)
        }
        static func preflightDependencies(in ioc: IoC) {
            _ = resolve(AuthModule.self, from: ioc)
            _ = resolve(NetworkModule.self, from: ioc)
        }
    }
}
