import PackageDescription

let package = Package(
    name: "Auth",
    platforms: [
        .iOS(.v16),
    ],
    products: [
        .library(name: "Auth", type: .dynamic, targets: ["Auth"]),
        .library(name: "AuthTesting", type: .static, targets: ["AuthTests"]),
    ],
    dependencies: [
        .package(path: "../CoreKit"),
        .package(url: "https://github.com/relux-works/ExternalSDK.git", from: "1.0.0"),
    ],
    targets: [
        .target(
            name: "Auth",
            dependencies: [
                .product(name: "CoreKit", package: "CoreKit"),
            ]
        ),
        .testTarget(
            name: "AuthTests",
            dependencies: ["Auth"]
        ),
    ]
)
