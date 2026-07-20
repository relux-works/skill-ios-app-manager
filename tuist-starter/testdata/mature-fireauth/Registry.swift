import AppCore
import MatureFeature
@_exported import Relux
import SecureStore
import SwiftIoC
import TokenProvider
import TokenProviderImpl

extension MatureApp {
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
            ioc.register(Relux.self, lifecycle: .container, resolver: buildRelux)
            ioc.register(AppCore.Runtime.self, lifecycle: .container, resolver: buildRuntime)

            // MARK: - Foundation (scaffolding anchor: foundation)
            ioc.register(MatureFeature.Persistence.self, lifecycle: .container, resolver: buildPersistence)
            ioc.register(IApiConfigManager.self, lifecycle: .container, resolver: buildAppConfigManager)
            ioc.register(TokenProvider.Module.Interface.self, lifecycle: .container, resolver: buildTokenProvider)

            // MARK: - Features (scaffolding anchor: features)

            // MARK: - Network (scaffolding anchor: network)
            ioc.register(MatureFeature.SyncEngine.self, lifecycle: .container, resolver: buildSyncEngine)
            ioc.register(MatureFeature.APIClient.self, lifecycle: .container, resolver: buildAPIClient)

            // MARK: - Utils (scaffolding anchor: utils)
        }

        static func resolve<T>(_ type: T.Type) -> T {
            guard let value = ioc.get(by: type) else {
                preconditionFailure("Unregistered mature dependency: \(String(reflecting: type))")
            }
            return value
        }
    }
}

// MARK: - Infrastructure Builders (scaffolding anchor: infra-builders)
extension MatureApp.Registry {
    private static func buildRelux() async -> Relux {
        let syncEngine: MatureFeature.SyncEngine = switch runtimeMode {
        case .application:
            resolve(MatureFeature.SyncEngine.self)
        case .hostedTests:
            .disabledForHostedTests()
        }

        let relux = await Relux(
            logger: MatureApp.ReluxLogger(),
            appStore: resolve(Relux.Store.self),
            rootSaga: resolve(Relux.RootSaga.self)
        )
        let module = await MatureFeature.Module(
            persistence: resolve(MatureFeature.Persistence.self),
            syncEngine: syncEngine,
            dispatcher: relux.dispatcher
        )
        return relux.register(module)
    }

    private static func buildRuntime() -> AppCore.Runtime { .live() }
}

// MARK: - Foundation Builders (scaffolding anchor: foundation-builders)
extension MatureApp.Registry {
    private static func buildPersistence() -> MatureFeature.Persistence { .init() }

    private static func buildAppConfigManager() -> IApiConfigManager {
        AppConfig.Business.Manager(secureStore: resolve(SecureStoring.self))
    }

    private static func buildTokenProvider() -> TokenProvider.Module.Interface {
        TokenProvider.Module.Impl()
    }
}

// MARK: - Network Builders (scaffolding anchor: network-builders)
extension MatureApp.Registry {
    private static func buildSyncEngine() -> MatureFeature.SyncEngine { .live() }

    private static func buildAPIClient() -> MatureFeature.APIClient {
        .init(runtime: resolve(AppCore.Runtime.self))
    }
}
