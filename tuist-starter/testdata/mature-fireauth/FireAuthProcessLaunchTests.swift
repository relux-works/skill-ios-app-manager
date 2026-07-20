import XCTest

final class GeneratedFireAuthReluxProcessLaunchTests: XCTestCase {
    func testDeterministicLaunchRunsInApplicationProcess() {
        let app = XCUIApplication()
        app.launchArguments +=
            GeneratedFireAuthReluxTestLaunch.deterministicLaunchArguments
        app.launch()

        XCTAssertTrue(app.wait(for: .runningForeground, timeout: 15))

        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "deterministic-fireauth-application-process"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }
}
