package profile

// GenerateLayoutHierarchyProbeSwift returns a debug UI-test helper for rendered hierarchy XML dumps.
func GenerateLayoutHierarchyProbeSwift() string {
	return `#if DEBUG
import CoreGraphics
import XCTest

public enum LayoutHierarchyProbe {
    public struct Options {
        public var maxDepth: Int
        public var includeValues: Bool

        public init(maxDepth: Int = 32, includeValues: Bool = true) {
            self.maxDepth = maxDepth
            self.includeValues = includeValues
        }
    }

    @MainActor
    public static func xml(
        for app: XCUIApplication,
        screenName: String? = nil,
        options: Options = Options()
    ) -> String {
        var lines: [String] = [
            "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
        ]
        let frame = app.frame
        var rootAttributes: [(String, String)] = [
            ("source", "XCTest"),
            ("platform", "iOS"),
            ("screenWidth", format(frame.width)),
            ("screenHeight", format(frame.height))
        ]
        if let screenName, !screenName.isEmpty {
            rootAttributes.append(("screen", screenName))
        }

        lines.append("<layout \(attributes(rootAttributes))>")
        appendElement(app, depth: 0, path: "/application[1]", into: &lines, options: options)
        lines.append("</layout>")
        return lines.joined(separator: "\n")
    }

    @MainActor
    private static func appendElement(
        _ element: XCUIElement,
        depth: Int,
        path: String,
        into lines: inout [String],
        options: Options
    ) {
        let frame = element.frame
        let type = elementTypeName(element.elementType)
        var elementAttributes: [(String, String)] = [
            ("type", type),
            ("path", path),
            ("depth", "\(depth)"),
            ("identifier", element.identifier),
            ("label", element.label),
            ("enabled", element.isEnabled ? "true" : "false"),
            ("hittable", element.isHittable ? "true" : "false"),
            ("x", format(frame.origin.x)),
            ("y", format(frame.origin.y)),
            ("width", format(frame.size.width)),
            ("height", format(frame.size.height))
        ]

        if let placeholder = element.placeholderValue, !placeholder.isEmpty {
            elementAttributes.append(("placeholder", placeholder))
        }
        if options.includeValues, let value = element.value {
            elementAttributes.append(("value", String(describing: value)))
        }

        let indent = String(repeating: "  ", count: depth + 1)
        let children = depth < options.maxDepth ? element.children(matching: .any).allElementsBoundByIndex : []
        if children.isEmpty {
            lines.append("\(indent)<element \(attributes(elementAttributes)) />")
            return
        }

        lines.append("\(indent)<element \(attributes(elementAttributes))>")
        var counters: [String: Int] = [:]
        for child in children {
            let childType = elementTypeName(child.elementType)
            counters[childType, default: 0] += 1
            let childPath = "\(path)/\(pathToken(childType))[\(counters[childType, default: 0])]"
            appendElement(child, depth: depth + 1, path: childPath, into: &lines, options: options)
        }
        lines.append("\(indent)</element>")
    }

    private static func elementTypeName(_ type: XCUIElement.ElementType) -> String {
        switch type {
        case .application: return "application"
        case .window: return "window"
        case .sheet: return "sheet"
        case .alert: return "alert"
        case .button: return "button"
        case .cell: return "cell"
        case .collectionView: return "collectionView"
        case .image: return "image"
        case .link: return "link"
        case .navigationBar: return "navigationBar"
        case .scrollView: return "scrollView"
        case .searchField: return "searchField"
        case .secureTextField: return "secureTextField"
        case .staticText: return "staticText"
        case .switch: return "switch"
        case .tab: return "tab"
        case .tabBar: return "tabBar"
        case .table: return "table"
        case .textField: return "textField"
        case .textView: return "textView"
        case .toolbar: return "toolbar"
        case .webView: return "webView"
        default:
            return String(describing: type)
        }
    }

    private static func attributes(_ values: [(String, String)]) -> String {
        values
            .filter { !$0.1.isEmpty }
            .map { "\($0.0)=\"\(escape($0.1))\"" }
            .joined(separator: " ")
    }

    private static func escape(_ value: String) -> String {
        value
            .replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "\"", with: "&quot;")
            .replacingOccurrences(of: "'", with: "&apos;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
    }

    private static func format(_ value: CGFloat) -> String {
        String(format: "%.2f", Double(value))
    }

    private static func pathToken(_ value: String) -> String {
        let allowed = value.unicodeScalars.map { scalar -> Character in
            if CharacterSet.alphanumerics.contains(scalar) {
                return Character(scalar)
            }
            return "-"
        }
        return String(allowed).lowercased()
    }
}

public extension XCTestCase {
    @MainActor
    @discardableResult
    func attachLayoutHierarchyXML(
        _ app: XCUIApplication,
        name: String = "layout-hierarchy",
        screenName: String? = nil,
        options: LayoutHierarchyProbe.Options = LayoutHierarchyProbe.Options()
    ) -> String {
        let xml = LayoutHierarchyProbe.xml(for: app, screenName: screenName, options: options)
        let attachment = XCTAttachment(string: xml)
        attachment.name = "\(name).xml"
        attachment.lifetime = .keepAlways
        add(attachment)

        print("IAM_LAYOUT_XML_START \(name)")
        print(xml)
        print("IAM_LAYOUT_XML_END \(name)")
        return xml
    }
}
#endif
`
}
