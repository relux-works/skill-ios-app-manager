import ProjectDescription

/// A typed representation of an Xcode "capability" (primarily entitlements).
///
/// Capabilities are declared in app / app extension manifests and are translated into
/// `.entitlements` via `EntitlementsFactory`.
public struct Capability: Hashable, Sendable {
    /// The build/signing environment for entitlements that differ between development and production.
    ///
    /// This is used by capabilities like Push Notifications and App Attest, which require a
    /// `development` or `production` entitlement value depending on the provisioning profile.
    public enum DistributionEnvironment: String, Hashable, Sendable {
        /// Development / debug environment.
        case development

        /// Production / distribution environment.
        case production
    }

    /// iCloud services used by the target.
    ///
    /// Mirrors the service checkboxes shown by Xcode under the iCloud capability.
    public struct ICloudServices: OptionSet, Hashable, Sendable {
        /// Bitmask raw value backing the option set.
        ///
        /// Prefer using the predefined options (`.keyValueStorage`, `.documents`, `.cloudKit`).
        public let rawValue: Int

        /// Creates a service set from a raw bitmask.
        ///
        /// Prefer using the predefined options (`.keyValueStorage`, `.documents`, `.cloudKit`).
        public init(rawValue: Int) {
            self.rawValue = rawValue
        }

        /// iCloud key-value storage (`com.apple.developer.ubiquity-kvstore-identifier`).
        public static let keyValueStorage = ICloudServices(rawValue: 1 << 0)

        /// iCloud Documents (`com.apple.developer.ubiquity-container-identifiers`).
        public static let documents = ICloudServices(rawValue: 1 << 1)

        /// CloudKit (`com.apple.developer.icloud-container-identifiers`).
        public static let cloudKit = ICloudServices(rawValue: 1 << 2)
    }

    /// HealthKit options enabled for the target.
    ///
    /// These mirror the checkboxes shown by Xcode under the HealthKit capability.
    public struct HealthKitOptions: OptionSet, Hashable, Sendable {
        /// Bitmask raw value backing the option set.
        public let rawValue: Int

        /// Creates an options set from a raw bitmask.
        public init(rawValue: Int) {
            self.rawValue = rawValue
        }

        /// Enables access to clinical data types (Clinical Health Records).
        ///
        /// This translates to:
        /// - `com.apple.developer.healthkit.access` = `["health-records"]`
        public static let clinicalHealthRecords = HealthKitOptions(rawValue: 1 << 0)

        /// Allows Background Delivery of HealthKit Observer Queries.
        ///
        /// This translates to:
        /// - `com.apple.developer.healthkit.background-delivery` = `true`
        public static let backgroundDelivery = HealthKitOptions(rawValue: 1 << 1)
    }

    /// A capability identifier that can be derived from the host bundle ID or provided explicitly.
    ///
    /// Many capabilities use a "default" identifier convention (for example `group.<bundle id>`
    /// for App Groups or `iCloud.<bundle id>` for iCloud containers).
    public enum Identifier: Hashable, Sendable {
        /// Uses the capability's default identifier derived from the host bundle ID.
        case `default`

        /// Uses an explicit identifier.
        case custom(id: String)
    }

    /// App Group identifiers used by the App Groups capability.
    ///
    /// App Group identifiers must start with `group.`.
    public enum AppGroupIdentifier: Hashable, Sendable {
        /// An explicitly provided identifier value.
        ///
        /// Use `prependingGroupPrefix(_:)` to build an App Group identifier from a reverse-DNS
        /// string without manually typing `group.`.
        public struct Value: Hashable, Sendable, ExpressibleByStringLiteral {
            /// Underlying identifier string.
            public let rawValue: String

            /// Creates a value from a raw identifier string.
            ///
            /// Any leading/trailing whitespace is trimmed.
            public init(_ rawValue: String) {
                self.rawValue = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
            }

            /// Creates a value from a string literal.
            ///
            /// Any leading/trailing whitespace is trimmed.
            public init(stringLiteral value: String) {
                self.init(value)
            }

            /// Creates an App Group identifier by prepending the `group.` prefix when missing.
            ///
            /// Example:
            /// - Input: `com.acme.shared` -> Output: `group.com.acme.shared`
            public static func prependingGroupPrefix(_ value: String) -> Self {
                let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
                guard !trimmed.hasPrefix("group.") else { return Self(trimmed) }
                return Self("group.\(trimmed)")
            }
        }

        /// Uses the capability's default identifier derived from the host bundle ID.
        case `default`

        /// Uses an explicit identifier.
        ///
        /// - Important: The value must start with `group.`. Prefer using
        ///   `Value.prependingGroupPrefix(_:)`.
        case custom(id: Value)

        /// Converts the App Group identifier into a generic capability identifier.
        var identifier: Identifier {
            switch self {
            case .default:
                return .default
            case let .custom(id: id):
                return .custom(id: id.rawValue)
            }
        }
    }

    /// App category for the Carrier-Constrained Network capability.
    ///
    /// This maps to `com.apple.developer.networking.carrier-constrained.appcategory`.
    public enum CarrierConstrainedNetworkAppCategory: String, Hashable, Sendable {
        case messaging = "messaging-8001"
        case maps = "maps-8002"
        case hikingAdventure = "hiking-adventure-8003"
        case music = "music-8004"
        case weather = "weather-8005"
        case voip = "voip-8006"
        case emergency = "emergency-8007"
        case lightSocial = "light-social-8008"
        case rideShare = "ride-share-8009"
        case foodDelivery = "food-delivery-8010"
        case news = "news-8011"
        case productivity = "productivity-8012"
        case interactiveProductivity = "interactive-productivity-8013"
        case healthAndFitness = "health-fitness-8014"
    }

    /// Fonts permissions for the Fonts capability.
    ///
    /// This maps to `com.apple.developer.user-fonts`.
    public enum FontsPermission: String, Hashable, Sendable {
        /// Allows the app to use installed fonts.
        case appUsage = "app-usage"

        /// Allows the app to install system-wide fonts.
        case systemInstallation = "system-installation"
    }

    /// iCloud Extended Share Access options.
    ///
    /// This maps to `com.apple.developer.icloud-extended-share-access`.
    public enum ICloudExtendedShareAccessOption: String, Hashable, Sendable {
        case inProcessShareAccessRequests = "InProcessShareAccessRequests"
        case inProcessShareOwnerParticipantInfo = "InProcessShareOwnerParticipantInfo"
        case inProcessOneTimeLinks = "InProcessOneTimeLinks"
    }

    /// Supported document types for the Digital Credentials API - Mobile Document Provider capability.
    ///
    /// This maps to `com.apple.developer.identity-document-services.document-provider.mobile-document-types`.
    public enum MobileDocumentType: String, Hashable, Sendable {
        /// Mobile Driver's License (mDL) (`org.iso.18013.5.1.mDL`).
        case mobileDriversLicense = "org.iso.18013.5.1.mDL"

        /// My Number Card (Japan) (`org.iso.23220.1.jp.mnc`).
        case myNumberCardJapan = "org.iso.23220.1.jp.mnc"

        /// National ID / Digital ID (passport-verified) (`org.iso.23220.photoid.1`).
        case nationalIDPassportVerified = "org.iso.23220.photoid.1"

        /// EU PID (eIDAS 2.0) (`eu.europa.ec.eudi.pid.1`).
        case euPid = "eu.europa.ec.eudi.pid.1"
    }

    /// Network Extension entitlement types.
    ///
    /// This maps to `com.apple.developer.networking.networkextension`.
    public enum NetworkExtensionType: String, Hashable, Sendable {
        case appProxyProvider = "app-proxy-provider"
        case contentFilterProvider = "content-filter-provider"
        case dnsProxy = "dns-proxy"
        case packetTunnelProvider = "packet-tunnel-provider"
        case transparentProxyProvider = "transparent-proxy-provider"
        case filterDataProvider = "filter-data-provider"
        case packetTunnelProviderSystem = "packet-tunnel-provider-system"
        case appProxyProviderSystem = "app-proxy-provider-system"
    }

    /// App category for the 5G Network Slicing capability.
    ///
    /// This maps to `com.apple.developer.networking.slicing.appcategory`.
    public enum FiveGNetworkSlicingAppCategory: String, Hashable, Sendable {
        case communication = "communication-9000"
        case games = "games-6014"
        case streaming = "streaming-9001"
    }

    /// Traffic categories for the 5G Network Slicing capability.
    ///
    /// This maps to `com.apple.developer.networking.slicing.trafficcategory`.
    public enum FiveGNetworkSlicingTrafficCategory: String, Hashable, Sendable {
        case defaultSlice = "defaultslice-1"
        case video = "video-2"
        case background = "background-3"
        case voice = "voice-4"
        case callSignaling = "callsignaling-5"
        case responsiveData = "responsivedata-6"
        case avStreaming = "avstreaming-7"
        case responsiveAv = "responsiveav-8"
    }

    /// Supported NFC reader session formats for NFC Tag Reading.
    ///
    /// This maps to `com.apple.developer.nfc.readersession.formats`.
    public enum NFCTagReaderSessionFormat: String, Hashable, Sendable {
        case ndef = "NDEF"
        case tag = "TAG"
        case pace = "PACE"
    }

    /// User management options (tvOS).
    ///
    /// This maps to `com.apple.developer.user-management`.
    public enum UserManagementOption: String, Hashable, Sendable {
        case getCurrentUser = "get-current-user"
        case runsAsCurrentUser = "runs-as-current-user"
        case runsAsCurrentUserWithUserIndependentKeychain = "runs-as-current-user-with-user-independent-keychain"
    }

    /// Roles for Wi-Fi Aware (Publish / Subscribe).
    ///
    /// This maps to `com.apple.developer.wifi-aware`.
    public enum WiFiAwareRole: String, Hashable, Sendable {
        case publish = "Publish"
        case subscribe = "Subscribe"
    }

    /// A raw entitlement key/value pair.
    ///
    /// This is an escape hatch for capabilities that are not yet modeled by the DSL.
    public struct CustomEntitlement: Hashable, Sendable {
        /// Entitlement key, for example `com.apple.security.hardened-process`.
        public let key: String

        /// Entitlement value.
        public let value: Value

        /// Supported entitlement value types.
        public enum Value: Hashable, Sendable {
            case boolean(Bool)
            case string(String)
            case strings([String])
        }

        /// Creates a custom entitlement from a key/value pair.
        public init(key: String, value: Value) {
            self.key = key.trimmingCharacters(in: .whitespacesAndNewlines)
            self.value = value
        }
    }

    /// Internal representation used by `EntitlementsFactory` for entitlement generation.
    enum Kind: Hashable, Sendable {
        case iCloud(services: ICloudServices, containers: [Identifier], keyValueStoreIdentifier: Identifier)
        case healthKit(options: HealthKitOptions)
        case keychainSharing(group: Identifier)
        case appGroups(group: Identifier)
        case iCloudUbiquityContainer(container: Identifier)
        case iCloudCloudKitContainer(container: Identifier)
        case iCloudKeyValueStore(id: Identifier)
        case applePayMerchant(id: Identifier)
        case walletPassType(id: Identifier)
        case associatedDomains(domains: [String])
        case portal(capability: PortalCapability, environment: DistributionEnvironment)
        case carrierConstrainedNetwork(appCategory: CarrierConstrainedNetworkAppCategory)
        case fonts(permissions: [FontsPermission])
        case iCloudExtendedShareAccess(options: [ICloudExtendedShareAccessOption])
        case mobileDocumentProvider(types: [MobileDocumentType])
        case networkExtensions(types: [NetworkExtensionType])
        case fiveGNetworkSlicing(appCategory: FiveGNetworkSlicingAppCategory, trafficCategories: [FiveGNetworkSlicingTrafficCategory])
        case nfcTagReading(formats: [NFCTagReaderSessionFormat])
        case userManagement(options: [UserManagementOption])
        case wiFiAware(roles: [WiFiAwareRole])
        case customEntitlements(entitlements: [CustomEntitlement])
    }

    /// Backing payload for the capability.
    let kind: Kind

    /// Creates a capability from an internal payload.
    ///
    /// Prefer using the static factory methods (e.g. `.appGroups()`, `.iCloud(services:)`).
    private init(_ kind: Kind) {
        self.kind = kind
    }

    /// Keychain Sharing (`keychain-access-groups`).
    ///
    /// Default: `$(AppIdentifierPrefix)<host bundle id>`
    public static func keychainSharing(group: Identifier = .default) -> Self {
        .init(.keychainSharing(group: group))
    }

    /// iCloud capability (services + containers).
    ///
    /// This is the high-level, "Xcode-like" API. It expands to the underlying entitlements:
    /// - `com.apple.developer.icloud-services` for `.documents` and `.cloudKit`
    /// - `com.apple.developer.ubiquity-container-identifiers` for `.documents`
    /// - `com.apple.developer.icloud-container-identifiers` for `.cloudKit`
    /// - `com.apple.developer.ubiquity-kvstore-identifier` for `.keyValueStorage`
    ///
    /// Defaults:
    /// - Containers: `[iCloud.<host bundle id>]`
    /// - Key-value store identifier: `$(TeamIdentifierPrefix)<host bundle id>`
    public static func iCloud(
        services: ICloudServices,
        containers: [Identifier] = [.default],
        keyValueStoreIdentifier: Identifier = .default
    ) -> Self {
        .init(.iCloud(services: services, containers: containers, keyValueStoreIdentifier: keyValueStoreIdentifier))
    }

    /// HealthKit (`com.apple.developer.healthkit`).
    ///
    /// By default only the base HealthKit entitlement is enabled.
    ///
    /// Options:
    /// - `.clinicalHealthRecords` enables clinical records access (`com.apple.developer.healthkit.access`).
    /// - `.backgroundDelivery` enables HealthKit background delivery (`com.apple.developer.healthkit.background-delivery`).
    public static func healthKit(_ options: HealthKitOptions = []) -> Self {
        .init(.healthKit(options: options))
    }

    /// App Groups (`com.apple.security.application-groups`).
    ///
    /// Default: `group.<host bundle id>`
    public static func appGroups(group: AppGroupIdentifier = .default) -> Self {
        .init(.appGroups(group: group.identifier))
    }

    /// iCloud Documents / Ubiquity container identifiers (`com.apple.developer.ubiquity-container-identifiers`).
    ///
    /// Default: `iCloud.<host bundle id>`
    public static func iCloudUbiquityContainer(container: Identifier = .default) -> Self {
        .init(.iCloudUbiquityContainer(container: container))
    }

    /// CloudKit container identifiers (`com.apple.developer.icloud-container-identifiers`).
    ///
    /// Default: `iCloud.<host bundle id>`
    public static func iCloudCloudKitContainer(container: Identifier = .default) -> Self {
        .init(.iCloudCloudKitContainer(container: container))
    }

    /// iCloud key-value store identifier (`com.apple.developer.ubiquity-kvstore-identifier`).
    ///
    /// Default: `$(TeamIdentifierPrefix)<host bundle id>`
    public static func iCloudKeyValueStore(id: Identifier = .default) -> Self {
        .init(.iCloudKeyValueStore(id: id))
    }

    /// Apple Pay merchant IDs (`com.apple.developer.in-app-payments`).
    ///
    /// Default: `merchant.<host bundle id>`
    public static func applePayMerchant(id: Identifier = .default) -> Self {
        .init(.applePayMerchant(id: id))
    }

    /// Wallet pass type IDs (`com.apple.developer.pass-type-identifiers`).
    ///
    /// Default: `pass.<host bundle id>`
    public static func walletPassType(id: Identifier = .default) -> Self {
        .init(.walletPassType(id: id))
    }

    /// Associated domains (`com.apple.developer.associated-domains`).
    public static func associatedDomains(_ domains: [String]) -> Self {
        .init(.associatedDomains(domains: domains))
    }

    /// Enables an Apple Developer portal capability by identifier.
    ///
    /// This is a low-level escape hatch for capabilities that map directly to entitlements with
    /// unambiguous values (for example `WeatherKit` or `Sign in with Apple`).
    public static func portal(
        _ capability: PortalCapability,
        environment: DistributionEnvironment = .development
    ) -> Self {
        .init(.portal(capability: capability, environment: environment))
    }

    /// Carrier-Constrained Network Category and Optimized.
    ///
    /// Translates to:
    /// - `com.apple.developer.networking.carrier-constrained.app-optimized` = `true`
    /// - `com.apple.developer.networking.carrier-constrained.appcategory` = `[<selected category>]`
    public static func carrierConstrainedNetwork(
        appCategory: CarrierConstrainedNetworkAppCategory
    ) -> Self {
        .init(.carrierConstrainedNetwork(appCategory: appCategory))
    }

    /// Fonts capability (`com.apple.developer.user-fonts`).
    ///
    /// The array controls whether the app can *use* installed fonts and/or *install* fonts.
    public static func fonts(_ permissions: [FontsPermission]) -> Self {
        .init(.fonts(permissions: permissions))
    }

    /// iCloud Extended Share Access (`com.apple.developer.icloud-extended-share-access`).
    public static func iCloudExtendedShareAccess(_ options: [ICloudExtendedShareAccessOption]) -> Self {
        .init(.iCloudExtendedShareAccess(options: options))
    }

    /// Digital Credentials API - Mobile Document Provider.
    public static func mobileDocumentProvider(_ types: [MobileDocumentType]) -> Self {
        .init(.mobileDocumentProvider(types: types))
    }

    /// Network Extensions (`com.apple.developer.networking.networkextension`).
    public static func networkExtensions(types: [NetworkExtensionType]) -> Self {
        .init(.networkExtensions(types: types))
    }

    /// 5G Network Slicing.
    ///
    /// Translates to:
    /// - `com.apple.developer.networking.slicing.appcategory` = `[<selected category>]`
    /// - `com.apple.developer.networking.slicing.trafficcategory` = `[<selected traffic categories>]`
    public static func fiveGNetworkSlicing(
        appCategory: FiveGNetworkSlicingAppCategory,
        trafficCategories: [FiveGNetworkSlicingTrafficCategory]
    ) -> Self {
        .init(.fiveGNetworkSlicing(appCategory: appCategory, trafficCategories: trafficCategories))
    }

    /// NFC Tag Reading (`com.apple.developer.nfc.readersession.formats`).
    public static func nfcTagReading(formats: [NFCTagReaderSessionFormat]) -> Self {
        .init(.nfcTagReading(formats: formats))
    }

    /// User Management (tvOS) (`com.apple.developer.user-management`).
    public static func userManagement(_ options: [UserManagementOption]) -> Self {
        .init(.userManagement(options: options))
    }

    /// Wi-Fi Aware (`com.apple.developer.wifi-aware`).
    public static func wiFiAware(_ roles: [WiFiAwareRole]) -> Self {
        .init(.wiFiAware(roles: roles))
    }

    /// Adds one or more raw entitlements to the target.
    ///
    /// Prefer using the dedicated `Capability` APIs when available, because they provide
    /// identifier defaults and configuration guardrails.
    public static func customEntitlements(_ entitlements: [CustomEntitlement]) -> Self {
        .init(.customEntitlements(entitlements: entitlements))
    }
}
