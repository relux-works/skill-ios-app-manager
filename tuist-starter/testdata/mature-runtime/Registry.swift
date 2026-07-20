import MatureFeature
import SwiftIoC

extension DemoApp {
    enum RuntimeMode {
        case application
        case hostedTests
    }

    @MainActor
    enum Registry {
        static let ioc = IoC()
        private(set) static var runtimeMode = RuntimeMode.application

        static func configure(runtimeMode: RuntimeMode = .application) {
            self.runtimeMode = runtimeMode

            // MARK: - Infrastructure (scaffolding anchor: infra)

            // MARK: - Foundation (scaffolding anchor: foundation)
            ioc.register(MatureFeature.Persistence.self, lifecycle: .container, resolver: buildPersistence)

            // MARK: - Features (scaffolding anchor: features)

            // MARK: - Network (scaffolding anchor: network)
            ioc.register(MatureFeature.APIClient.self, lifecycle: .container, resolver: buildAPIClient)

            // MARK: - Utils (scaffolding anchor: utils)
        }

        static func resolve<T>(_ type: T.Type) -> T {
            guard let value = ioc.get(by: type) else {
                preconditionFailure("Unregistered custom dependency")
            }
            return value
        }
    }
}

// MARK: - Infrastructure Builders (scaffolding anchor: infra-builders)
extension DemoApp.Registry {
    private static func buildPersistence() -> MatureFeature.Persistence {
        .init()
    }

    private static func buildAPIClient() -> MatureFeature.APIClient {
        .init(persistence: resolve(MatureFeature.Persistence.self))
    }
}
