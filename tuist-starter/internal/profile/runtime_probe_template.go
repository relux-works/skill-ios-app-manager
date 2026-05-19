package profile

const performanceProbeSwift = `#if DEBUG
import Foundation
import os
import SwiftUI

public enum PerformanceProbe {
    private static let processStart = CFAbsoluteTimeGetCurrent()

    private static let log = OSLog(
        subsystem: Bundle.main.bundleIdentifier ?? "ios-app-manager.profile",
        category: "PerformanceProbe"
    )

    public static func markAppStart(
        _ name: String = "AppStart",
        file: StaticString = #fileID,
        line: UInt = #line
    ) {
        record(
            kind: "app_start",
            name: name,
            durationMS: 0,
            timestamp: processStart,
            file: String(describing: file),
            line: Int(line)
        )
    }

    public static func markFirstRender(
        _ name: String = "FirstRender",
        file: StaticString = #fileID,
        line: UInt = #line
    ) {
        let now = CFAbsoluteTimeGetCurrent()
        record(
            kind: "first_render",
            name: name,
            durationMS: (now - processStart) * 1_000,
            timestamp: now,
            file: String(describing: file),
            line: Int(line)
        )
    }

    @discardableResult
    public static func measure<T>(
        _ name: String,
        file: StaticString = #fileID,
        line: UInt = #line,
        _ operation: () throws -> T
    ) rethrows -> T {
        let signpostID = OSSignpostID(log: log)
        let start = CFAbsoluteTimeGetCurrent()
        os_signpost(.begin, log: log, name: "PerformanceProbe", signpostID: signpostID)
        defer {
            let durationMS = (CFAbsoluteTimeGetCurrent() - start) * 1_000
            os_signpost(.end, log: log, name: "PerformanceProbe", signpostID: signpostID)
            record(
                kind: "function",
                name: name,
                durationMS: durationMS,
                file: String(describing: file),
                line: Int(line)
            )
        }
        return try operation()
    }

    @discardableResult
    public static func measureAsync<T>(
        _ name: String,
        file: StaticString = #fileID,
        line: UInt = #line,
        _ operation: () async throws -> T
    ) async rethrows -> T {
        let signpostID = OSSignpostID(log: log)
        let start = CFAbsoluteTimeGetCurrent()
        os_signpost(.begin, log: log, name: "PerformanceProbe", signpostID: signpostID)
        defer {
            let durationMS = (CFAbsoluteTimeGetCurrent() - start) * 1_000
            os_signpost(.end, log: log, name: "PerformanceProbe", signpostID: signpostID)
            record(
                kind: "async_function",
                name: name,
                durationMS: durationMS,
                file: String(describing: file),
                line: Int(line)
            )
        }
        return try await operation()
    }

    public static func recordViewBody(
        _ name: String,
        durationMS: Double,
        file: StaticString = #fileID,
        line: UInt = #line
    ) {
        record(
            kind: "view_body",
            name: name,
            durationMS: durationMS,
            file: String(describing: file),
            line: Int(line)
        )
    }

    public static func event(_ name: String, file: StaticString = #fileID, line: UInt = #line) {
        record(
            kind: "event",
            name: name,
            durationMS: 0,
            file: String(describing: file),
            line: Int(line)
        )
    }

    public static func error(
        _ name: String,
        message: String,
        severity: String = "error",
        file: StaticString = #fileID,
        line: UInt = #line
    ) {
        let payload: [String: Any] = [
            "source": "performance_probe",
            "severity": severity,
            "name": name,
            "message": message,
            "thread": Thread.isMainThread ? "main" : "background",
            "timestamp": Date().timeIntervalSince1970,
            "file": String(describing: file),
            "line": Int(line)
        ]

        guard
            JSONSerialization.isValidJSONObject(payload),
            let data = try? JSONSerialization.data(withJSONObject: payload, options: [.sortedKeys]),
            let json = String(data: data, encoding: .utf8)
        else {
            print("IAM_ERROR {\"severity\":\"error\",\"message\":\"json-serialization\"}")
            return
        }

        print("IAM_ERROR \(json)")
    }

    private static func record(
        kind: String,
        name: String,
        durationMS: Double,
        timestamp: Double = CFAbsoluteTimeGetCurrent(),
        file: String,
        line: Int
    ) {
        let payload: [String: Any] = [
            "kind": kind,
            "name": name,
            "duration_ms": durationMS,
            "thread": Thread.isMainThread ? "main" : "background",
            "timestamp": timestamp,
            "file": file,
            "line": line
        ]

        guard
            JSONSerialization.isValidJSONObject(payload),
            let data = try? JSONSerialization.data(withJSONObject: payload, options: [.sortedKeys]),
            let json = String(data: data, encoding: .utf8)
        else {
            print("IAM_PROFILE {\"kind\":\"error\",\"name\":\"json-serialization\"}")
            return
        }

        print("IAM_PROFILE \(json)")
    }
}

public struct ProfiledView<Content: View>: View {
    private let name: String
    private let content: Content

    public init(_ name: String, @ViewBuilder content: () -> Content) {
        self.name = name
        self.content = content()
    }

    public var body: some View {
        let start = CFAbsoluteTimeGetCurrent()
        let view = content
        let durationMS = (CFAbsoluteTimeGetCurrent() - start) * 1_000
        PerformanceProbe.recordViewBody(name, durationMS: durationMS)
        return view
    }
}

public extension View {
    func profiled(_ name: String) -> some View {
        ProfiledView(name) {
            self
        }
    }

    func firstRenderProfiled(_ name: String = "FirstRender") -> some View {
        self.onAppear {
            PerformanceProbe.markFirstRender(name)
        }
    }
}
#endif
`
