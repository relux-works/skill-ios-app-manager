// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "BSimTOTP",
    platforms: [
        .iOS(.v13),
        .macOS(.v11),
        .tvOS(.v13),
        .watchOS(.v6)
    ],
    products: [
        .library(
            name: "BSimTOTP",
            targets: ["BSimTOTP"]
        ),
        .library(
            name: "BSimTOTPImpl",
            targets: ["BSimTOTPImpl"]
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
            name: "BSimTOTP",
            dependencies: [
                .product(name: "BSimSDK", package: "BSimSDK"),
            ],
            path: "Interface"
        ),

        // implementation
        .target(
            name: "BSimTOTPImpl",
            dependencies: [
                .target(name: "BSimTOTP"),
                .product(name: "BSimSDK", package: "BSimSDK"),
                .product(name: "BSimSDKImpl", package: "BSimSDK"),
                .product(name: "HttpClient", package: "darwin-httpclient"),
            ],
            path: "Impl"
        ),

        // tests
        .testTarget(
            name: "BSimTOTPTests",
            dependencies: [
                "BSimTOTP",
                "BSimTOTPImpl",
            ],
            path: "Tests"
        ),
    ]
)
