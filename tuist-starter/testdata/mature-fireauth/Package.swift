// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "MatureAppDependencies",
    dependencies: [
        .package(path: "Packages/MatureFeature"),
        .package(name: "FireAuthRelux", url: "https://github.com/relux-works/FireAuthRelux.git", .exact("1.0.0")),
        .package(name: "FireAuthKit", url: "https://github.com/relux-works/FireAuthKit.git", .exact("1.0.0")),
    ],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "MatureFeature": .framework,
        "FireAuthRelux": .framework,
    ],
    targetSettings: [
        "MatureFeature": .settings(base: [
            "CUSTOM_SETTING": "preserved",
        ]),
        "FireAuthRelux": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "18.0",
        ]),
    ]
)
#endif
