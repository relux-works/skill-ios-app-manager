import ProjectDescription
import ProjectDescriptionHelpers

let appName = "MatureApp"
let bundleID = "com.example.mature"
let hostedTestRuntimeArguments = Arguments.arguments(
    launchArguments: [
        .launchArgument(name: "--mature-hosted-tests", isEnabled: true),
    ]
)

let project = Project(
    name: appName,
    targets: [
        .target(
            name: appName,
            destinations: .iOS,
            product: .app,
            bundleId: bundleID,
            sources: ["Targets/MatureApp/Sources/**"],
            dependencies: [
                .external(name: "SwiftIoC"),
                .external(name: "SwiftUIRelux"),
                .external(name: "TokenProvider"),
                .external(name: "TokenProviderImpl"),
                .external(name: "MatureFeature"),
            ]
        ),
        .target(
            name: "MatureAppTests",
            destinations: .iOS,
            product: .unitTests,
            bundleId: "\(bundleID).tests",
            sources: ["Targets/MatureAppTests/Sources/**"],
            dependencies: [
                .target(name: appName),
                .external(name: "MatureFeature"),
            ]
        ),
        .target(
            name: "MatureAppUITests",
            destinations: .iOS,
            product: .uiTests,
            bundleId: "\(bundleID).ui-tests",
            sources: ["Targets/MatureAppUITests/Sources/**"],
            dependencies: [
                .target(name: appName),
                .external(name: "MatureFeature"),
            ]
        ),
    ],
    schemes: [
        .scheme(
            name: appName,
            shared: true,
            buildAction: .buildAction(targets: [.target(appName)]),
            testAction: .targets(
                [
                    .testableTarget(target: .target("MatureAppTests")),
                    .testableTarget(target: .target("MatureAppUITests")),
                ],
                arguments: hostedTestRuntimeArguments
            )
        ),
    ]
)
