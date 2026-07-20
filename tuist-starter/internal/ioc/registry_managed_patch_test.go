package ioc

import (
	"strings"
	"testing"
)

const matureRegistryFixture = `import CustomRuntime
import SwiftIoC
@_exported import Relux

extension DemoApp {
    @MainActor
    enum Registry {
        static let ioc = IoC()
        private(set) static var runtimeMode = CustomRuntimeMode.application

        static func configure(runtimeMode: CustomRuntimeMode = .current()) {
            self.runtimeMode = runtimeMode

            // MARK: - Infrastructure (scaffolding anchor: infra)
            ioc.register(Relux.self, lifecycle: .container, resolver: buildRelux)
            ioc.register(CustomRuntime.self, lifecycle: .container, resolver: buildCustomRuntime)

            // MARK: - Foundation (scaffolding anchor: foundation)
            ioc.register(CustomPersistence.self, lifecycle: .container, resolver: buildCustomPersistence)

            // MARK: - Features (scaffolding anchor: features)

            // MARK: - Network (scaffolding anchor: network)
            ioc.register(CustomAPIClient.self, lifecycle: .container, resolver: buildCustomAPIClient)

            // MARK: - Utils (scaffolding anchor: utils)
        }

        static func resolve<T>(_ type: T.Type) -> T {
            guard let value = ioc.get(by: type) else {
                preconditionFailure("custom resolve")
            }
            return value
        }
    }
}

// MARK: - Infrastructure Builders (scaffolding anchor: infra-builders)
extension DemoApp.Registry {
    private static func buildRelux() async -> Relux {
        let relux = await Relux(
            logger: CustomLogger(),
            appStore: .init(),
            rootSaga: .init()
        )
        let feature = await CustomFeature(dispatcher: relux.dispatcher)
        return relux.register(feature)
    }

    private static func buildCustomRuntime() -> CustomRuntime { .init() }
}

// MARK: - Foundation Builders (scaffolding anchor: foundation-builders)
extension DemoApp.Registry {
    private static func buildCustomPersistence() -> CustomPersistence { .init() }
}
`

func TestConvergeManagedFoundationRegistryContentPreservesMatureComposition(t *testing.T) {
	t.Parallel()

	patch := RegistryManagedFoundationPatch{
		ID:                       "token-provider",
		Imports:                  []string{"TokenProvider", "TokenProviderImpl"},
		Registration:             "ioc.register(TokenProvider.Module.Interface.self, lifecycle: .container, resolver: Self.buildTokenProvider)",
		LegacyRegistrationMarker: "TokenProvider.Module.Interface.self",
		Builder: `private static func buildTokenProvider() -> TokenProvider.Module.Interface {
    TokenProvider.Module.Impl()
}`,
		LegacyBuilderMarker: "func buildTokenProvider()",
	}

	first, err := ConvergeManagedFoundationRegistryContent(matureRegistryFixture, "DemoApp", patch)
	if err != nil {
		t.Fatalf("ConvergeManagedFoundationRegistryContent() error = %v", err)
	}
	second, err := ConvergeManagedFoundationRegistryContent(first, "DemoApp", patch)
	if err != nil {
		t.Fatalf("second ConvergeManagedFoundationRegistryContent() error = %v", err)
	}
	if second != first {
		t.Fatalf("managed foundation patch is not byte-idempotent:\n%s", second)
	}

	for _, preserved := range []string{
		"private(set) static var runtimeMode",
		"ioc.register(CustomRuntime.self",
		"ioc.register(CustomPersistence.self",
		"ioc.register(CustomAPIClient.self",
		"preconditionFailure(\"custom resolve\")",
		"let feature = await CustomFeature",
		"return relux.register(feature)",
	} {
		if !strings.Contains(first, preserved) {
			t.Fatalf("managed foundation patch lost %q:\n%s", preserved, first)
		}
	}
	for _, integrated := range []string{
		"import TokenProvider",
		"import TokenProviderImpl",
		"// ios-app-manager:token-provider-registration:begin",
		"TokenProvider.Module.Interface.self",
		"// ios-app-manager:token-provider-builder:begin",
		"func buildTokenProvider()",
	} {
		if !strings.Contains(first, integrated) {
			t.Fatalf("managed foundation patch missing %q:\n%s", integrated, first)
		}
	}
}

func TestConvergeManagedFoundationRegistryContentAdoptsLegacyOwnedSlices(t *testing.T) {
	t.Parallel()

	legacy := strings.Replace(
		matureRegistryFixture,
		"            // MARK: - Features (scaffolding anchor: features)",
		"            ioc.register(TokenProvider.Module.Interface.self, lifecycle: .container, resolver: buildLegacyTokenProvider)\n\n            // MARK: - Features (scaffolding anchor: features)",
		1,
	)
	legacy = strings.Replace(
		legacy,
		"    private static func buildCustomPersistence() -> CustomPersistence { .init() }",
		`    private static func buildCustomPersistence() -> CustomPersistence { .init() }

    private static func buildTokenProvider() -> TokenProvider.Module.Interface {
        LegacyTokenProvider()
    }`,
		1,
	)

	patch := RegistryManagedFoundationPatch{
		ID:                       "token-provider",
		Imports:                  []string{"TokenProvider", "TokenProviderImpl"},
		Registration:             "ioc.register(TokenProvider.Module.Interface.self, lifecycle: .container, resolver: Self.buildTokenProvider)",
		LegacyRegistrationMarker: "TokenProvider.Module.Interface.self",
		Builder: `private static func buildTokenProvider() -> TokenProvider.Module.Interface {
    TokenProvider.Module.Impl()
}`,
		LegacyBuilderMarker: "func buildTokenProvider()",
	}

	updated, err := ConvergeManagedFoundationRegistryContent(legacy, "DemoApp", patch)
	if err != nil {
		t.Fatalf("ConvergeManagedFoundationRegistryContent() error = %v", err)
	}
	if strings.Contains(updated, "buildLegacyTokenProvider") || strings.Contains(updated, "LegacyTokenProvider") {
		t.Fatalf("legacy owned slices were not replaced:\n%s", updated)
	}
	if count := strings.Count(updated, "TokenProvider.Module.Interface.self"); count != 1 {
		t.Fatalf("TokenProvider registration count = %d, want 1:\n%s", count, updated)
	}
	if count := strings.Count(updated, "func buildTokenProvider()"); count != 1 {
		t.Fatalf("TokenProvider builder count = %d, want 1:\n%s", count, updated)
	}
}

func TestConvergeManagedFoundationRegistryContentExpandsEmptySingleLineExtension(t *testing.T) {
	t.Parallel()

	content := strings.Replace(
		matureRegistryFixture,
		`extension DemoApp.Registry {
    private static func buildCustomPersistence() -> CustomPersistence { .init() }
}`,
		"extension DemoApp.Registry {}",
		1,
	)
	patch := RegistryManagedFoundationPatch{
		ID:                       "token-provider",
		Imports:                  []string{"TokenProvider", "TokenProviderImpl"},
		Registration:             "ioc.register(TokenProvider.Module.Interface.self, lifecycle: .container, resolver: Self.buildTokenProvider)",
		LegacyRegistrationMarker: "TokenProvider.Module.Interface.self",
		Builder: `private static func buildTokenProvider() -> TokenProvider.Module.Interface {
    TokenProvider.Module.Impl()
}`,
		LegacyBuilderMarker: "func buildTokenProvider()",
	}

	updated, err := ConvergeManagedFoundationRegistryContent(content, "DemoApp", patch)
	if err != nil {
		t.Fatalf("ConvergeManagedFoundationRegistryContent() error = %v", err)
	}
	if !strings.Contains(updated, "extension DemoApp.Registry {\n    // ios-app-manager:token-provider-builder:begin") {
		t.Fatalf("single-line extension was not expanded safely:\n%s", updated)
	}
}

func TestConvergeManagedReluxWrapperContentWrapsExistingBuilderWithoutChangingIt(t *testing.T) {
	t.Parallel()

	patch := RegistryManagedReluxPatch{
		ID:               "fireauth-relux",
		WrapperName:      "buildReluxWithFireAuthRelux",
		ModuleExpression: "resolve(FireAuthRelux.Module.Interface.self)",
	}

	first, err := ConvergeManagedReluxWrapperContent(matureRegistryFixture, "DemoApp", patch)
	if err != nil {
		t.Fatalf("ConvergeManagedReluxWrapperContent() error = %v", err)
	}
	second, err := ConvergeManagedReluxWrapperContent(first, "DemoApp", patch)
	if err != nil {
		t.Fatalf("second ConvergeManagedReluxWrapperContent() error = %v", err)
	}
	if second != first {
		t.Fatalf("managed Relux wrapper is not byte-idempotent:\n%s", second)
	}

	for _, preserved := range []string{
		"private static func buildRelux() async -> Relux",
		"let feature = await CustomFeature(dispatcher: relux.dispatcher)",
		"return relux.register(feature)",
	} {
		if !strings.Contains(first, preserved) {
			t.Fatalf("Relux wrapper changed existing builder value %q:\n%s", preserved, first)
		}
	}
	for _, integrated := range []string{
		`// ios-app-manager:fireauth-relux-relux-registration:begin original="buildRelux"`,
		"resolver: Self.buildReluxWithFireAuthRelux",
		"// ios-app-manager:fireauth-relux-relux-wrapper:begin",
		"let relux = await buildRelux()",
		"return relux.register(resolve(FireAuthRelux.Module.Interface.self))",
	} {
		if !strings.Contains(first, integrated) {
			t.Fatalf("Relux wrapper missing %q:\n%s", integrated, first)
		}
	}
}

func TestConvergeManagedReluxWrapperContentRejectsUnknownResolverShape(t *testing.T) {
	t.Parallel()

	content := strings.Replace(
		matureRegistryFixture,
		"ioc.register(Relux.self, lifecycle: .container, resolver: buildRelux)",
		"ioc.register(Relux.self, lifecycle: .container) { await makeRelux(using: runtimeMode) }",
		1,
	)

	_, err := ConvergeManagedReluxWrapperContent(content, "DemoApp", RegistryManagedReluxPatch{
		ID:               "fireauth-relux",
		WrapperName:      "buildReluxWithFireAuthRelux",
		ModuleExpression: "resolve(FireAuthRelux.Module.Interface.self)",
	})
	if err == nil || !strings.Contains(err.Error(), "supported Relux registration") {
		t.Fatalf("error = %v, want explicit unsupported resolver error", err)
	}
}
