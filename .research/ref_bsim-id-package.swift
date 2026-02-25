// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "BSimID",
    platforms: [
        .iOS(.v13),
        .macOS(.v11),
        .tvOS(.v13),
        .watchOS(.v6)
    ],
    products: [
        .library(
            name: "BSimID",
            targets: ["BSimID"]
        ),
        .library(
            name: "BSimIDImpl",
            targets: ["BSimIDImpl"]
        ),
    ],
    dependencies: [
        .package(name: "BSimSDK", path: "../bsimsdk"),
//        .package(url: "https://gitlab.services.mts.ru/rnd-telecom/projects/bsim/mobile/ios/bsimsdk.git", .upToNextMajor(from: "0.0.2")),
        .package(url: "https://github.com/ivalx1s/darwin-httpclient.git", .upToNextMajor(from: "5.5.0")),
    ],
    targets: [
        // interfaces
        .target(
            name: "BSimID",
            dependencies: [
                .product(name: "BSimSDK", package: "BSimSDK"),
            ],
            path: "Interface"
        ),

        // implementations
        .target(
            name: "BSimIDImpl",
            dependencies: [
                .target(name: "BSimID"),
                .product(name: "BSimSDK", package: "BSimSDK"),
                .product(name: "BSimSDKImpl", package: "BSimSDK"),
                .product(name: "HttpClient", package: "darwin-httpclient"),
            ],
            path: "Impl"
        ),

        // tests
        .testTarget(
            name: "BSimIDImplTests",
            dependencies: ["BSimIDImpl"],
            path: "Tests"
        ),
    ]
)
