import ProjectDescription
import ProjectDescriptionHelpers

let appName = "DemoApp"
let bundleID = "com.example.demo"
let hostedTestRuntimeArguments = Arguments.arguments(
    launchArguments: [
        .launchArgument(name: "--demo-hosted-tests", isEnabled: true),
    ]
)
let configurations: [Configuration] = [
    .debug(name: "Debug"),
    .debug(
        name: "Staging",
        settings: ["SWIFT_ACTIVE_COMPILATION_CONDITIONS": "$(inherited) STAGING"]
    ),
    .release(name: "Release"),
]

let project = Project(
    name: appName,
    settings: .settings(configurations: configurations),
    targets: [
        .target(
            name: appName,
            destinations: .iOS,
            product: .app,
            bundleId: bundleID,
            infoPlist: .extendingDefault(
                with: [
                    "ApplicationConfiguration": .dictionary([
                        "appName": .string("DemoApp"),
                        "applicationBundleIdentifier": .string("com.example.demo"),
                        "developmentTeamID": .string("ABCDE12345"),
                    ]),
                ]
            ),
            sources: ["Targets/DemoApp/Sources/**"],
            dependencies: [
                .external(name: "SwiftIoC"),
                .external(name: "MatureFeature"),
            ]
        ),
        .target(
            name: "DemoAppTests",
            destinations: .iOS,
            product: .unitTests,
            bundleId: "\(bundleID).tests",
            sources: ["Targets/DemoAppTests/Sources/**"],
            dependencies: [
                .target(name: appName),
                .external(name: "MatureFeature")
            ]
        ),
        .target(
            name: "DemoAppUITests",
            destinations: .iOS,
            product: .uiTests,
            bundleId: "\(bundleID).ui-tests",
            sources: ["Targets/DemoAppUITests/Sources/**"],
            dependencies: [
                .target(name: appName),
                .external(name: "MatureFeature")
            ]
        )
    ],
    schemes: [
        .scheme(
            name: appName,
            shared: true,
            buildAction: .buildAction(targets: [.target(appName)]),
            testAction: .targets(
                [
                    .testableTarget(target: .target("DemoAppTests")),
                    .testableTarget(target: .target("DemoAppUITests")),
                ],
                arguments: hostedTestRuntimeArguments,
                configuration: .debug,
                expandVariableFromTarget: .target(appName)
            ),
            runAction: .runAction(configuration: .debug, executable: .target(appName))
        ),
    ]
)
