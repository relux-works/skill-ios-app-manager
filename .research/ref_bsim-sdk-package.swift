// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "BSimSDK",
    platforms: [
        .iOS(.v13),
        .macOS(.v10_15),
        .tvOS(.v13),
        .watchOS(.v6)
    ],
    products: [
        .library(
            name: "BSimSDK",
            targets: ["BSimSDK"]
        ),
        .library(
            name: "BSimSDKImpl",
            targets: ["BSimSDKImpl"]
        ),
    ],

    targets: [
        // interfcases
        .target(
            name: "BSimSDK",
            path: "Interface"
        ),

        // implementations
        .target(
            name: "BSimSDKImpl",
            dependencies: [
                .target(name: "BSimSDK")
            ],
            path: "Impl"
        ),

        // tests for impls
        .testTarget(
            name: "BSimSDKTests",
            dependencies: ["BSimSDKImpl"],
            path: "Tests"
        ),
    ]
)
