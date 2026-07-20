// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoAppDependencies",
    dependencies: [
        .package(path: "Packages/MatureFeature"),
    ],
    targets: []
)

#if TUIST
    import ProjectDescription

    let packageConfigurations: [Configuration] = [
        .debug(name: "Debug"),
        .debug(name: "Staging"),
        .release(name: "Release"),
    ]
    let swiftPackageTargetSettings: Settings = .settings(
        base: ["SWIFT_VERSION": "6.0"],
        configurations: packageConfigurations
    )
    let packageSettings = PackageSettings(
        productTypes: [
            "MatureFeature": .framework,
        ],
        baseSettings: .settings(configurations: packageConfigurations),
        targetSettings: [
            "MatureFeature": swiftPackageTargetSettings,
        ]
    )
#endif
