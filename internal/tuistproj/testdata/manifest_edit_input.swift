import PackageDescription

let package = Package(
    name: "Auth",
    products: [
        .library(name: "Auth", type: .dynamic, targets: ["Auth"]),
    ],
    dependencies: [
        .package(path: "../CoreKit"),
    ],
    targets: [
        .target(
            name: "Auth",
            dependencies: [
                .product(name: "CoreKit", package: "CoreKit"),
            ]
        ),
    ]
)
