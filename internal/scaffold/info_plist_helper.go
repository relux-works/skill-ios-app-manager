package scaffold

// GenerateInfoPlistHelper returns the Bundle+InfoPlist.swift extension
// that provides typed access to Info.plist values by key.
func GenerateInfoPlistHelper() string {
	return `import Foundation

extension Bundle {
    static func readInfoPlistValue<T>(by key: String, from bundle: Bundle) -> T {
        guard let infoDictionary = bundle.infoDictionary,
              let value = infoDictionary[key] as? T else {
            fatalError("Could not read key from \(bundle.description) Info.plist")
        }
        return value
    }

    func readInfoPlistValue<T>(by key: String) -> T {
        Self.readInfoPlistValue(by: key, from: self)
    }
}
`
}
