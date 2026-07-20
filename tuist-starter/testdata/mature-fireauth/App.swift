import CustomRuntime
import SwiftUI

@main
struct MatureApp: App {
    init() {
        CustomRuntime.prepare()
        Registry.configure(runtimeMode: .application)
    }

    var body: some Scene {
        WindowGroup {
            Text("Mature app")
        }
    }
}
