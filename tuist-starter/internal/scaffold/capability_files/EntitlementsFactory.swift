import Foundation
import ProjectDescription

/// Builds `.entitlements` values from a set of `Capability` declarations.
///
/// This is the source of truth for what ends up in the code signature for host apps and app extensions.
///
/// - Important: When `destinations` contains multiple platforms, the generated entitlements must be
///   identical across those platforms (Tuist only supports a single entitlements file per target).
///   If they differ, this factory aborts with a message telling you to split the target by platform.
public enum EntitlementsFactory {
    /// Builds an `Entitlements` value for the given host bundle ID, destinations, and capabilities.
    ///
    /// Returns `nil` when `capabilities` don't produce any entitlements.
    public static func make(hostBundleId: String, destinations: Destinations, capabilities: [Capability]) -> Entitlements? {
        let platforms = destinations.platforms
        guard !platforms.isEmpty else { return nil }

        validatePortalBackedCapabilitiesSupportedOnAllPlatforms(capabilities: capabilities, platforms: platforms)

        let entitlementsByPlatform: [(platform: Platform, entitlements: NormalizedEntitlements)] =
            platforms.sorted(by: { String(describing: $0) < String(describing: $1) }).map { platform in
                (platform: platform, entitlements: makeDictionary(hostBundleId: hostBundleId, capabilities: capabilities, platform: platform))
            }

        let hasAnyEntitlements = entitlementsByPlatform.contains { !$0.entitlements.isEmpty }
        guard hasAnyEntitlements else { return nil }

        guard let reference = entitlementsByPlatform.first else { return nil }
        for (platform, entitlements) in entitlementsByPlatform where entitlements != reference.entitlements {
            failMultiplatformEntitlementsMismatch(
                referencePlatform: reference.platform,
                reference: reference.entitlements,
                platform: platform,
                entitlements: entitlements
            )
        }

        return .dictionary(plistDictionary(from: reference.entitlements))
    }

    /// A normalized entitlement value that is easy to compare across platforms.
    private enum NormalizedEntitlementValue: Hashable, Sendable {
        case boolean(Bool)
        case string(String)
        case strings([String])
    }

    /// A normalized entitlements dictionary.
    private typealias NormalizedEntitlements = [String: NormalizedEntitlementValue]

    /// Builds a normalized entitlement dictionary for the given platform.
    private static func makeDictionary(hostBundleId: String, capabilities: [Capability], platform: Platform) -> NormalizedEntitlements {
        var keychainAccessGroups: Set<String> = []
        var applicationGroups: Set<String> = []
        var ubiquityContainerIdentifiers: [String] = []
        var cloudKitContainerIdentifiers: [String] = []
        var iCloudServices: Set<String> = []
        var ubiquityKVStoreIdentifier: String?
        var merchantIdentifiers: Set<String> = []
        var passTypeIdentifiers: Set<String> = []
        var associatedDomains: Set<String> = []
        var extraEntitlements: NormalizedEntitlements = [:]

        for capability in capabilities {
            switch capability.kind {
            case let .iCloud(services, containers, keyValueStoreIdentifier):
                let resolvedContainers = containers.isEmpty ? [.default] : containers

                if services.contains(.documents) {
                    iCloudServices.insert("CloudDocuments")
                    ubiquityContainerIdentifiers.append(contentsOf: resolvedContainers.map { resolveICloudContainer($0, hostBundleId: hostBundleId) })
                }

                if services.contains(.cloudKit) {
                    iCloudServices.insert("CloudKit")
                    cloudKitContainerIdentifiers.append(contentsOf: resolvedContainers.map { resolveICloudContainer($0, hostBundleId: hostBundleId) })
                }

                if services.contains(.keyValueStorage) {
                    ubiquityKVStoreIdentifier = resolveIdentifierPrefixed(
                        keyValueStoreIdentifier,
                        defaultUnprefixed: hostBundleId,
                        prefixMacro: "$(TeamIdentifierPrefix)"
                    )
                }

            case let .healthKit(options):
                guard platformSupportsPortalCapability(.healthKit, platform: platform) else { continue }
                mergeEntitlementValue(
                    key: "com.apple.developer.healthkit",
                    value: .boolean(true),
                    into: &extraEntitlements
                )

                if options.contains(.clinicalHealthRecords) {
                    mergeEntitlementValue(
                        key: "com.apple.developer.healthkit.access",
                        value: .strings(["health-records"]),
                        into: &extraEntitlements
                    )
                }

                if options.contains(.backgroundDelivery) {
                    mergeEntitlementValue(
                        key: "com.apple.developer.healthkit.background-delivery",
                        value: .boolean(true),
                        into: &extraEntitlements
                    )
                }

            case let .keychainSharing(group):
                keychainAccessGroups.insert(resolveKeychainAccessGroup(group, hostBundleId: hostBundleId))

            case let .appGroups(group):
                applicationGroups.insert(resolveAppGroup(group, hostBundleId: hostBundleId))

            case let .iCloudUbiquityContainer(container):
                iCloudServices.insert("CloudDocuments")
                ubiquityContainerIdentifiers.append(resolveICloudContainer(container, hostBundleId: hostBundleId))

            case let .iCloudCloudKitContainer(container):
                iCloudServices.insert("CloudKit")
                cloudKitContainerIdentifiers.append(resolveICloudContainer(container, hostBundleId: hostBundleId))

            case let .iCloudKeyValueStore(id):
                ubiquityKVStoreIdentifier = resolveIdentifierPrefixed(
                    id,
                    defaultUnprefixed: hostBundleId,
                    prefixMacro: "$(TeamIdentifierPrefix)"
                )

            case let .applePayMerchant(id):
                merchantIdentifiers.insert(resolveMerchantIdentifier(id, hostBundleId: hostBundleId))

            case let .walletPassType(id):
                passTypeIdentifiers.insert(resolvePassTypeIdentifier(id, hostBundleId: hostBundleId))

            case let .associatedDomains(domains):
                associatedDomains.formUnion(domains)

            case let .portal(portalCapability, environment):
                mergePortalCapabilityEntitlements(
                    portalCapability,
                    environment: environment,
                    platform: platform,
                    into: &extraEntitlements
                )

            case let .carrierConstrainedNetwork(appCategory):
                guard platformSupportsPortalCapability(.carrierConstrainedNetworkCategoryAndOptimized, platform: platform) else { continue }
                mergeEntitlementValue(
                    key: "com.apple.developer.networking.carrier-constrained.app-optimized",
                    value: .boolean(true),
                    into: &extraEntitlements
                )
                mergeEntitlementValue(
                    key: "com.apple.developer.networking.carrier-constrained.appcategory",
                    value: .strings([appCategory.rawValue]),
                    into: &extraEntitlements
                )

            case let .fonts(permissions):
                guard platformSupportsPortalCapability(.fonts, platform: platform) else { continue }
                guard !permissions.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: Fonts
                        Rule: Provide at least one `FontsPermission`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.user-fonts",
                    value: .strings(uniquePreservingOrder(permissions.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .iCloudExtendedShareAccess(options):
                guard platformSupportsPortalCapability(.iCloudExtendedShareAccess, platform: platform) else { continue }
                guard !options.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: iCloud Extended Share Access
                        Rule: Provide at least one `ICloudExtendedShareAccessOption`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.icloud-extended-share-access",
                    value: .strings(uniquePreservingOrder(options.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .mobileDocumentProvider(types):
                guard platformSupportsPortalCapability(.digitalCredentialsApiMobileDocumentProvider, platform: platform) else { continue }
                guard !types.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: Digital Credentials API - Mobile Document Provider
                        Rule: Provide at least one `MobileDocumentType`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.identity-document-services.document-provider.mobile-document-types",
                    value: .strings(uniquePreservingOrder(types.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .networkExtensions(types):
                guard platformSupportsPortalCapability(.networkExtensions, platform: platform) else { continue }
                guard !types.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: Network Extensions
                        Rule: Provide at least one `NetworkExtensionType`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.networking.networkextension",
                    value: .strings(uniquePreservingOrder(types.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .fiveGNetworkSlicing(appCategory, trafficCategories):
                guard platformSupportsPortalCapability(.fiveGNetworkSlicing, platform: platform) else { continue }
                guard !trafficCategories.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: 5G Network Slicing
                        Rule: Provide at least one `FiveGNetworkSlicingTrafficCategory`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.networking.slicing.appcategory",
                    value: .strings([appCategory.rawValue]),
                    into: &extraEntitlements
                )
                mergeEntitlementValue(
                    key: "com.apple.developer.networking.slicing.trafficcategory",
                    value: .strings(uniquePreservingOrder(trafficCategories.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .nfcTagReading(formats):
                guard platformSupportsPortalCapability(.nfcTagReading, platform: platform) else { continue }
                guard !formats.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: NFC Tag Reading
                        Rule: Provide at least one `NFCTagReaderSessionFormat`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.nfc.readersession.formats",
                    value: .strings(uniquePreservingOrder(formats.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .userManagement(options):
                guard platformSupportsPortalCapability(.userManagement, platform: platform) else { continue }
                guard !options.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: User Management
                        Rule: Provide at least one `UserManagementOption`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.user-management",
                    value: .strings(uniquePreservingOrder(options.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .wiFiAware(roles):
                guard platformSupportsPortalCapability(.wiFiAware, platform: platform) else { continue }
                guard !roles.isEmpty else {
                    fatalError(
                        """
                        INVALID CAPABILITY CONFIGURATION
                        ---------------------------------------------------
                        Capability: Wi-Fi Aware
                        Rule: Provide at least one `WiFiAwareRole`.
                        ---------------------------------------------------
                        """
                    )
                }
                mergeEntitlementValue(
                    key: "com.apple.developer.wifi-aware",
                    value: .strings(uniquePreservingOrder(roles.map(\.rawValue))),
                    into: &extraEntitlements
                )

            case let .customEntitlements(entitlements: customEntitlements):
                for entitlement in customEntitlements {
                    guard !entitlement.key.isEmpty else { continue }

                    let value: NormalizedEntitlementValue
                    switch entitlement.value {
                    case let .boolean(flag):
                        value = .boolean(flag)
                    case let .string(string):
                        value = .string(string)
                    case let .strings(strings):
                        value = .strings(uniquePreservingOrder(strings))
                    }

                    mergeEntitlementValue(key: entitlement.key, value: value, into: &extraEntitlements)
                }
            }
        }

        var entitlements: NormalizedEntitlements = [:]

        if !keychainAccessGroups.isEmpty {
            entitlements["keychain-access-groups"] = .strings(keychainAccessGroups.sorted())
        }

        if !applicationGroups.isEmpty {
            entitlements["com.apple.security.application-groups"] = .strings(applicationGroups.sorted())
        }

        let uniqueUbiquityContainers = uniquePreservingOrder(ubiquityContainerIdentifiers)
        if !uniqueUbiquityContainers.isEmpty {
            entitlements["com.apple.developer.ubiquity-container-identifiers"] = .strings(uniqueUbiquityContainers)
        }

        let uniqueCloudKitContainers = uniquePreservingOrder(cloudKitContainerIdentifiers)
        if !uniqueCloudKitContainers.isEmpty {
            entitlements["com.apple.developer.icloud-container-identifiers"] = .strings(uniqueCloudKitContainers)
        }

        if !iCloudServices.isEmpty {
            entitlements["com.apple.developer.icloud-services"] = .strings(iCloudServices.sorted())
        }

        if let ubiquityKVStoreIdentifier {
            entitlements["com.apple.developer.ubiquity-kvstore-identifier"] = .string(ubiquityKVStoreIdentifier)
        }

        if !merchantIdentifiers.isEmpty {
            entitlements["com.apple.developer.in-app-payments"] = .strings(merchantIdentifiers.sorted())
        }

        if !passTypeIdentifiers.isEmpty {
            entitlements["com.apple.developer.pass-type-identifiers"] = .strings(passTypeIdentifiers.sorted())
        }

        if !associatedDomains.isEmpty {
            entitlements["com.apple.developer.associated-domains"] = .strings(associatedDomains.sorted())
        }

        for (key, value) in extraEntitlements {
            mergeEntitlementValue(key: key, value: value, into: &entitlements)
        }

        return entitlements
    }

    /// Produces a `ProjectDescription` entitlements dictionary.
    private static func plistDictionary(from entitlements: NormalizedEntitlements) -> [String: Plist.Value] {
        var dictionary: [String: Plist.Value] = [:]
        dictionary.reserveCapacity(entitlements.count)

        for (key, value) in entitlements {
            switch value {
            case let .boolean(flag):
                dictionary[key] = .boolean(flag)
            case let .string(string):
                dictionary[key] = .string(string)
            case let .strings(strings):
                dictionary[key] = .array(strings.map { .string($0) })
            }
        }

        return dictionary
    }

    /// Aborts manifest evaluation when entitlements differ between platforms in a multiplatform target.
    private static func failMultiplatformEntitlementsMismatch(
        referencePlatform: Platform,
        reference: NormalizedEntitlements,
        platform: Platform,
        entitlements: NormalizedEntitlements
    ) -> Never {
        let referenceKeys = Set(reference.keys)
        let keys = Set(entitlements.keys)

        let missing = referenceKeys.subtracting(keys).sorted()
        let extra = keys.subtracting(referenceKeys).sorted()
        let common = referenceKeys.intersection(keys)
        let differing = common.filter { reference[$0] != entitlements[$0] }.sorted()

        fatalError(
            """
            MULTIPLATFORM ENTITLEMENTS MISMATCH
            ---------------------------------------------------
            Target destinations contain multiple platforms, but generated entitlements differ:

            Reference platform: \(String(describing: referencePlatform))
            Other platform: \(String(describing: platform))

            Missing keys on \(String(describing: platform)): \(missing)
            Extra keys on \(String(describing: platform)): \(extra)
            Differing keys: \(differing)

            Fix: Split the target by platform (separate iOS/macOS apps) or use only cross-platform capabilities.
            ---------------------------------------------------
            """
        )
    }

    /// Validates that portal-backed capabilities are supported on all destination platforms.
    private static func validatePortalBackedCapabilitiesSupportedOnAllPlatforms(capabilities: [Capability], platforms: Set<Platform>) {
        for capability in capabilities {
            for portalCapability in portalCapabilitiesReferenced(by: capability.kind) {
                guard let info = portalCapabilities[portalCapability] else {
                    fatalError(
                        """
                        MISSING PORTAL CAPABILITY METADATA
                        ---------------------------------------------------
                        Capability: \(portalCapability.rawValue)
                        Issue: The active Xcode portal capability catalog does not contain this entry.
                        ---------------------------------------------------
                        """
                    )
                }

                if !platforms.isSubset(of: info.supportedPlatforms) {
                    fatalError(
                        """
                        UNSUPPORTED CAPABILITY FOR DESTINATIONS
                        ---------------------------------------------------
                        Capability: \(info.name) (\(portalCapability.rawValue))
                        Target destinations: \(platforms.map { String(describing: $0) }.sorted().joined(separator: ", "))
                        Supported platforms: \(info.supportedPlatforms.map { String(describing: $0) }.sorted().joined(separator: ", "))
                        ---------------------------------------------------
                        """
                    )
                }
            }
        }
    }

    /// Returns the portal capability identifiers referenced by a capability kind (when applicable).
    private static func portalCapabilitiesReferenced(by kind: Capability.Kind) -> [Capability.PortalCapability] {
        switch kind {
        case let .portal(portalCapability, _):
            return [portalCapability]
        case .appGroups:
            return [.appGroups]
        case .associatedDomains:
            return [.associatedDomains]
        case .applePayMerchant:
            return [.applePayPaymentProcessing]
        case .walletPassType:
            return [.wallet]
        case .healthKit:
            return [.healthKit]
        case .iCloud, .iCloudUbiquityContainer, .iCloudCloudKitContainer, .iCloudKeyValueStore:
            return [.iCloud]
        case .carrierConstrainedNetwork:
            return [.carrierConstrainedNetworkCategoryAndOptimized]
        case .fonts:
            return [.fonts]
        case .iCloudExtendedShareAccess:
            return [.iCloudExtendedShareAccess]
        case .mobileDocumentProvider:
            return [.digitalCredentialsApiMobileDocumentProvider]
        case .networkExtensions:
            return [.networkExtensions]
        case .fiveGNetworkSlicing:
            return [.fiveGNetworkSlicing]
        case .nfcTagReading:
            return [.nfcTagReading]
        case .userManagement:
            return [.userManagement]
        case .wiFiAware:
            return [.wiFiAware]
        case .keychainSharing, .customEntitlements:
            return []
        }
    }

    /// Returns `true` when the portal capability supports the given platform according to Xcode.
    private static func platformSupportsPortalCapability(_ portalCapability: Capability.PortalCapability, platform: Platform) -> Bool {
        guard let info = portalCapabilities[portalCapability] else {
            fatalError(
                """
                MISSING PORTAL CAPABILITY METADATA
                ---------------------------------------------------
                Capability: \(portalCapability.rawValue)
                Issue: The active Xcode portal capability catalog does not contain this entry.
                ---------------------------------------------------
                """
            )
        }
        return info.supportedPlatforms.contains(platform)
    }

    /// Adds a portal capability's entitlements into an entitlement dictionary, filtered by platform.
    private static func mergePortalCapabilityEntitlements(
        _ portalCapability: Capability.PortalCapability,
        environment: Capability.DistributionEnvironment,
        platform: Platform,
        into entitlements: inout NormalizedEntitlements
    ) {
        guard portalCapability != .healthKit else {
            fatalError(
                """
                UNSUPPORTED PORTAL CAPABILITY
                ---------------------------------------------------
                Capability: HealthKit (HEALTHKIT)
                Issue: HealthKit has optional sub-features (Clinical Health Records, Background Delivery) that `.portal(...)` can't express safely.
                ---------------------------------------------------
                Suggestion: Use `.healthKit(_:)` instead of `.portal(.healthKit)`.
                """
            )
        }

        guard let info = portalCapabilities[portalCapability] else { return }
        guard info.supportedPlatforms.contains(platform) else { return }

        for entitlement in info.entitlements {
            let supportedPlatforms = entitlement.supportedPlatforms ?? info.supportedPlatforms
            guard supportedPlatforms.contains(platform) else { continue }

            let value = resolvePortalEntitlementValue(entitlement, environment: environment, portalCapability: info)
            mergeEntitlementValue(key: entitlement.key, value: value, into: &entitlements)
        }
    }

    /// Resolves a portal entitlement value, enforcing that the value is unambiguous.
    private static func resolvePortalEntitlementValue(
        _ entitlement: PortalEntitlementInfo,
        environment: Capability.DistributionEnvironment,
        portalCapability: PortalCapabilityInfo
    ) -> NormalizedEntitlementValue {
        switch entitlement.valueType {
        case .boolean:
            return .boolean(true)
        case .distributionEnvironmentSpecific:
            return .string(environment.rawValue)
        case .string:
            guard let constant = entitlement.singleConstantValue else {
                failPortalCapabilityRequiresConfiguration(portalCapability: portalCapability, entitlement: entitlement)
            }
            return .string(constant)
        case .array:
            guard let constant = entitlement.singleConstantValue else {
                failPortalCapabilityRequiresConfiguration(portalCapability: portalCapability, entitlement: entitlement)
            }
            return .strings([constant])
        }
    }

    /// Aborts manifest evaluation for portal capabilities that require additional configuration.
    private static func failPortalCapabilityRequiresConfiguration(
        portalCapability: PortalCapabilityInfo,
        entitlement: PortalEntitlementInfo
    ) -> Never {
        let suggestion = portalCapabilitySuggestion(for: portalCapability.id).map { "\nSuggestion: \($0)" } ?? ""
        fatalError(
            """
            PORTAL CAPABILITY REQUIRES CONFIGURATION
            ---------------------------------------------------
            Capability: \(portalCapability.name) (\(portalCapability.id))
            Entitlement: \(entitlement.key) (\(entitlement.valueType.rawValue))
            Issue: This portal capability can't be expressed without additional values.
            ---------------------------------------------------\(suggestion)
            """
        )
    }

    /// Provides a best-effort suggestion for a dedicated DSL helper to use instead of `.portal(...)`.
    private static func portalCapabilitySuggestion(for portalId: String) -> String? {
        switch portalId {
        case "APP_GROUPS":
            return ".appGroups(group:)"
        case "ASSOCIATED_DOMAINS":
            return ".associatedDomains(_:)"
        case "APPLE_PAY":
            return ".applePayMerchant(id:)"
        case "WALLET":
            return ".walletPassType(id:)"
        case "ICLOUD":
            return ".iCloud(services:containers:keyValueStoreIdentifier:)"
        case "HEALTHKIT":
            return ".healthKit(_:)"
        case "NETWORK_EXTENSIONS":
            return ".networkExtensions(types:)"
        case "NETWORK_SLICING":
            return ".fiveGNetworkSlicing(appCategory:trafficCategories:)"
        case "NFC_TAG_READING":
            return ".nfcTagReading(formats:)"
        case "FONT_INSTALLATION":
            return ".fonts(_:)"
        case "ICLOUD_EXTENDED_SHARE_ACCESS":
            return ".iCloudExtendedShareAccess(_:)"
        case "MOBILE_DOCUMENT_PROVIDER":
            return ".mobileDocumentProvider(_:)"
        case "USER_MANAGEMENT":
            return ".userManagement(_:)"
        case "WIFI_AWARE":
            return ".wiFiAware(_:)"
        case "CARRIER_CONSTRAINED_NETWORK_CAT_OPT":
            return ".carrierConstrainedNetwork(appCategory:)"
        case "ENABLED_FOR_MAC", "ENHANCED_SECURITY":
            return ".customEntitlements(_:)"
        default:
            return nil
        }
    }

    // MARK: - Identifier Resolution

    /// Resolves a Keychain Access Group entry (`keychain-access-groups`).
    ///
    /// Default: `$(AppIdentifierPrefix)<host bundle id>`
    private static func resolveKeychainAccessGroup(_ id: Capability.Identifier, hostBundleId: String) -> String {
        resolveIdentifierPrefixed(
            id,
            defaultUnprefixed: hostBundleId,
            prefixMacro: "$(AppIdentifierPrefix)"
        )
    }

    /// Resolves an App Group identifier entry (`com.apple.security.application-groups`).
    ///
    /// Default: `group.<host bundle id>`
    private static func resolveAppGroup(_ id: Capability.Identifier, hostBundleId: String) -> String {
        let resolved: String
        switch id {
        case .default:
            resolved = "group.\(hostBundleId)"
        case let .custom(id: customId):
            resolved = customId.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        validatePrefixedLowercaseReverseDNSIdentifier(
            resolved,
            prefix: "group.",
            title: "INVALID APP GROUP IDENTIFIER",
            capability: "App Groups"
        )
        return resolved
    }

    /// Resolves an iCloud container identifier (used for iCloud Documents and CloudKit containers).
    ///
    /// Default: `iCloud.<host bundle id>`
    private static func resolveICloudContainer(_ id: Capability.Identifier, hostBundleId: String) -> String {
        let resolved: String
        switch id {
        case .default:
            resolved = "iCloud.\(hostBundleId)"
        case let .custom(id: customId):
            resolved = customId.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        validateICloudContainerIdentifier(resolved)
        return resolved
    }

    private static func validateICloudContainerIdentifier(_ identifier: String) {
        validatePrefixedLowercaseReverseDNSIdentifier(
            identifier,
            prefix: "iCloud.",
            title: "INVALID ICLOUD CONTAINER IDENTIFIER",
            capability: "iCloud"
        )
    }

    /// Resolves an Apple Pay merchant identifier entry (`com.apple.developer.in-app-payments`).
    ///
    /// Default: `merchant.<host bundle id>`
    private static func resolveMerchantIdentifier(_ id: Capability.Identifier, hostBundleId: String) -> String {
        let resolved: String
        switch id {
        case .default:
            resolved = "merchant.\(hostBundleId)"
        case let .custom(id: customId):
            resolved = customId.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        validatePrefixedLowercaseReverseDNSIdentifier(
            resolved,
            prefix: "merchant.",
            title: "INVALID MERCHANT IDENTIFIER",
            capability: "Apple Pay"
        )
        return resolved
    }

    /// Resolves a Wallet pass type identifier entry (`com.apple.developer.pass-type-identifiers`).
    ///
    /// Default: `pass.<host bundle id>`
    private static func resolvePassTypeIdentifier(_ id: Capability.Identifier, hostBundleId: String) -> String {
        let resolved: String
        switch id {
        case .default:
            resolved = "pass.\(hostBundleId)"
        case let .custom(id: customId):
            resolved = customId.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        validatePrefixedLowercaseReverseDNSIdentifier(
            resolved,
            prefix: "pass.",
            title: "INVALID PASS TYPE IDENTIFIER",
            capability: "Wallet"
        )
        return resolved
    }

    /// Resolves an identifier that must be prefixed with an Xcode-provided identifier prefix macro.
    private static func resolveIdentifierPrefixed(
        _ id: Capability.Identifier,
        defaultUnprefixed: String,
        prefixMacro: String
    ) -> String {
        let unprefixed: String
        switch id {
        case .default:
            unprefixed = defaultUnprefixed
        case let .custom(id: customId):
            unprefixed = customId.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        validateLowercaseReverseDNSIdentifier(
            unprefixed,
            title: "INVALID IDENTIFIER",
            capability: "Entitlements",
            prefixDescription: prefixMacro
        )

        return "\(prefixMacro)\(unprefixed)"
    }

    // MARK: - Validation

    private static func validatePrefixedLowercaseReverseDNSIdentifier(_ identifier: String, prefix: String, title: String, capability: String) {
        guard identifier.hasPrefix(prefix) else {
            fatalError(
                """
                \(title)
                ---------------------------------------------------
                Capability: \(capability)
                Rule: Identifiers must start with '\(prefix)'.
                Value: \(identifier)
                ---------------------------------------------------
                """
            )
        }

        guard !identifier.contains("*") else {
            fatalError(
                """
                \(title)
                ---------------------------------------------------
                Capability: \(capability)
                Rule: Identifiers must not contain wildcard ('*') characters.
                Value: \(identifier)
                ---------------------------------------------------
                """
            )
        }

        let remainder = String(identifier.dropFirst(prefix.count))
        guard !remainder.isEmpty else {
            fatalError(
                """
                \(title)
                ---------------------------------------------------
                Capability: \(capability)
                Rule: Identifier must include a reverse-DNS name after '\(prefix)'.
                Value: \(identifier)
                ---------------------------------------------------
                """
            )
        }

        validateLowercaseReverseDNSIdentifier(
            remainder,
            title: title,
            capability: capability,
            prefixDescription: prefix,
            valueForDisplay: identifier
        )
    }

    private static func validateLowercaseReverseDNSIdentifier(
        _ identifier: String,
        title: String,
        capability: String,
        prefixDescription: String,
        valueForDisplay: String? = nil
    ) {
        let displayValue = valueForDisplay ?? identifier
        guard identifier == identifier.lowercased() else {
            fatalError(
                """
                \(title)
                ---------------------------------------------------
                Capability: \(capability)
                Rule: Identifier components must be lowercase.
                Prefix: \(prefixDescription)
                Value: \(displayValue)
                ---------------------------------------------------
                """
            )
        }

        guard identifier.range(of: #"^[a-z0-9.-]+$"#, options: .regularExpression) != nil else {
            fatalError(
                """
                \(title)
                ---------------------------------------------------
                Capability: \(capability)
                Rule: Identifier may contain only [a-z0-9.-].
                Prefix: \(prefixDescription)
                Value: \(displayValue)
                ---------------------------------------------------
                """
            )
        }

        let segments = identifier.split(separator: ".").map(String.init)
        guard !segments.isEmpty, segments.allSatisfy({ !$0.isEmpty }) else {
            fatalError(
                """
                \(title)
                ---------------------------------------------------
                Capability: \(capability)
                Rule: Identifier must not contain empty components.
                Prefix: \(prefixDescription)
                Value: \(displayValue)
                ---------------------------------------------------
                """
            )
        }

        for segment in segments {
            guard segment.range(of: #"^[a-z][a-z0-9-]*$"#, options: .regularExpression) != nil else {
                fatalError(
                    """
                    \(title)
                    ---------------------------------------------------
                    Capability: \(capability)
                    Rule: Each component must match ^[a-z][a-z0-9-]*$.
                    Prefix: \(prefixDescription)
                    Invalid component: \(segment)
                    Value: \(displayValue)
                    ---------------------------------------------------
                    """
                )
            }
        }
    }

    // MARK: - Helpers

    /// De-duplicates an array while preserving the first occurrence order.
    ///
    /// This is important for entitlements where order can be semantically meaningful (for example,
    /// the first iCloud container is treated as the "primary" container).
    private static func uniquePreservingOrder(_ values: [String]) -> [String] {
        var seen: Set<String> = []
        var result: [String] = []
        result.reserveCapacity(values.count)

        for value in values where seen.insert(value).inserted {
            result.append(value)
        }

        return result
    }

    /// Merges an entitlement value into a dictionary, handling common merge strategies.
    private static func mergeEntitlementValue(key: String, value: NormalizedEntitlementValue, into entitlements: inout NormalizedEntitlements) {
        guard let existing = entitlements[key] else {
            entitlements[key] = value
            return
        }

        switch (existing, value) {
        case let (.strings(existing), .strings(new)):
            entitlements[key] = .strings(uniquePreservingOrder(existing + new))
        case let (.string(existing), .string(new)):
            guard existing == new else { failEntitlementConflict(key: key, existing: existing, new: new) }
        case let (.boolean(existing), .boolean(new)):
            guard existing == new else { failEntitlementConflict(key: key, existing: existing, new: new) }
        default:
            failEntitlementConflict(key: key, existing: existing, new: value)
        }
    }

    /// Aborts manifest evaluation when two capabilities try to set the same entitlement to incompatible values.
    private static func failEntitlementConflict(key: String, existing: Any, new: Any) -> Never {
        fatalError(
            """
            ENTITLEMENT CONFLICT
            ---------------------------------------------------
            Key: \(key)
            Existing value: \(existing)
            New value: \(new)
            ---------------------------------------------------
            """
        )
    }

    // MARK: - Portal Capability Catalog

    /// Representation of an entitlement entry in Xcode's portal capability catalog.
    private struct PortalEntitlementInfo: Hashable, Sendable {
        /// Entitlement key (for example `com.apple.developer.weatherkit`).
        let key: String

        /// Value type declared by Xcode.
        let valueType: PortalEntitlementValueType

        /// Allowed values declared by Xcode.
        let values: [PortalEntitlementValue]

        /// Optional per-entitlement supported platforms.
        let supportedPlatforms: Set<Platform>?

        /// Returns a single constant value when the entitlement doesn't require configuration.
        var singleConstantValue: String? {
            guard values.count == 1, let value = values.first else { return nil }
            guard value.inference == .constant else { return nil }
            guard !value.value.contains("${") else { return nil }
            return value.value
        }
    }

    /// Representation of a portal capability entry in Xcode's catalog.
    private struct PortalCapabilityInfo: Hashable, Sendable {
        /// Portal capability identifier (for example `WEATHERKIT`).
        let id: String

        /// Human-readable capability name.
        let name: String

        /// Platforms supported by the capability.
        let supportedPlatforms: Set<Platform>

        /// Entitlement entries associated with the capability.
        let entitlements: [PortalEntitlementInfo]
    }

    /// Supported value types for portal entitlements.
    private enum PortalEntitlementValueType: String, Hashable, Sendable {
        case boolean = "BOOLEAN"
        case array = "ARRAY"
        case string = "STRING"
        case distributionEnvironmentSpecific = "DISTRIBUTION_ENV_SPECIFIC"
    }

    /// Supported value inference kinds in portal entitlements.
    private enum PortalEntitlementValueInference: String, Hashable, Sendable {
        case constant
        case wildcard
    }

    /// A single value entry in a portal entitlement definition.
    private struct PortalEntitlementValue: Hashable, Sendable {
        /// Inference strategy (`constant` or `wildcard`).
        let inference: PortalEntitlementValueInference

        /// Raw value string.
        let value: String
    }

    /// Cached portal capability catalog loaded from Xcode.
    private static let portalCapabilities: [Capability.PortalCapability: PortalCapabilityInfo] = loadPortalCapabilities()

    /// Loads portal capability definitions from the active Xcode installation.
    private static func loadPortalCapabilities() -> [Capability.PortalCapability: PortalCapabilityInfo] {
        let url = portalCapabilitiesURL()

        do {
            let data = try Data(contentsOf: url)
            let response = try JSONDecoder().decode(PortalCapabilitiesResponse.self, from: data)

            var result: [Capability.PortalCapability: PortalCapabilityInfo] = [:]
            result.reserveCapacity(response.data.count)

            for item in response.data {
                guard let capability = Capability.PortalCapability(rawValue: item.id) else { continue }

                let supportedPlatforms = Set((item.attributes.supportedSDKs ?? []).compactMap { platform(fromPortalSDK: $0.name) })
                let entitlements: [PortalEntitlementInfo] = (item.attributes.entitlements ?? []).map { record in
                    let valueType = PortalEntitlementValueType(rawValue: record.valueType ?? "") ?? .boolean
                    let values = (record.values ?? []).compactMap { valueRecord -> PortalEntitlementValue? in
                        guard let rawValue = valueRecord.value else { return nil }
                        let inference = PortalEntitlementValueInference(rawValue: valueRecord.inference ?? "constant") ?? .constant
                        return PortalEntitlementValue(inference: inference, value: rawValue)
                    }
                    let entitlementSupportedPlatforms = record.supportedSDKs.map { Set($0.compactMap { platform(fromPortalSDK: $0.name) }) }

                    return PortalEntitlementInfo(
                        key: record.profileKey,
                        valueType: valueType,
                        values: values,
                        supportedPlatforms: entitlementSupportedPlatforms
                    )
                }

                result[capability] = PortalCapabilityInfo(
                    id: item.id,
                    name: item.attributes.name,
                    supportedPlatforms: supportedPlatforms,
                    entitlements: entitlements
                )
            }

            return result
        } catch {
            fatalError("Failed to load portal capabilities from Xcode: \(error)")
        }
    }

    /// Returns the URL to Xcode's cached portal capabilities JSON file.
    private static func portalCapabilitiesURL() -> URL {
        let candidates: [URL] = [
            resolveXcodeDeveloperDirectoryFromEnvironment(),
            resolveXcodeDeveloperDirectoryFromXcodeSelect(),
            URL(fileURLWithPath: "/Applications/Xcode.app/Contents/Developer"),
        ].compactMap { $0 }

        for developerDir in candidates {
            // Typical: /Applications/Xcode.app/Contents/Developer -> /Applications/Xcode.app/Contents
            let contentsDir = developerDir.lastPathComponent == "Developer" ? developerDir.deletingLastPathComponent() : developerDir
            let url = contentsDir
                .appendingPathComponent("SharedFrameworks/DVTPortal.framework/Versions/A/Resources/DVTPortalCachedPortalCapabilities.json")

            if FileManager.default.fileExists(atPath: url.path) {
                return url
            }
        }

        fatalError(
            """
            MISSING XCODE PORTAL CAPABILITIES CATALOG
            ---------------------------------------------------
            Could not locate `DVTPortalCachedPortalCapabilities.json`.
            Make sure Xcode is installed and `xcode-select -p` points to it.
            ---------------------------------------------------
            """
        )
    }

    /// Returns the Xcode developer directory from `DEVELOPER_DIR` when set.
    private static func resolveXcodeDeveloperDirectoryFromEnvironment() -> URL? {
        guard let value = ProcessInfo.processInfo.environment["DEVELOPER_DIR"], !value.isEmpty else { return nil }
        return URL(fileURLWithPath: value)
    }

    /// Returns the Xcode developer directory using `xcode-select -p`.
    private static func resolveXcodeDeveloperDirectoryFromXcodeSelect() -> URL? {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/xcode-select")
        process.arguments = ["-p"]

        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = Pipe()

        do {
            try process.run()
            process.waitUntilExit()
            guard process.terminationStatus == 0 else { return nil }

            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            let output = String(decoding: data, as: UTF8.self).trimmingCharacters(in: .whitespacesAndNewlines)
            guard !output.isEmpty else { return nil }
            return URL(fileURLWithPath: output)
        } catch {
            return nil
        }
    }

    /// Maps Xcode portal SDK identifiers to `Platform`.
    private static func platform(fromPortalSDK name: String) -> Platform? {
        switch name {
        case "IOS":
            return .iOS
        case "MAC_OS":
            return .macOS
        case "TV_OS":
            return .tvOS
        case "WATCH_OS":
            return .watchOS
        case "VISION_OS":
            return .visionOS
        default:
            return nil
        }
    }

    // MARK: - Codable Models for Xcode Portal JSON

    /// Mirror of Xcode's portal capabilities JSON format.
    private struct PortalCapabilitiesResponse: Decodable {
        /// Portal capabilities payload list.
        let data: [PortalCapabilityRecord]
    }

    /// Portal capability record.
    private struct PortalCapabilityRecord: Decodable {
        /// Capability identifier (for example `WEATHERKIT`).
        let id: String

        /// Capability attributes.
        let attributes: PortalCapabilityAttributes
    }

    /// Portal capability attributes.
    private struct PortalCapabilityAttributes: Decodable {
        /// Display name shown by Xcode.
        let name: String

        /// Supported SDKs (platforms).
        let supportedSDKs: [PortalSDKRecord]?

        /// Entitlement entries.
        let entitlements: [PortalEntitlementRecord]?
    }

    /// Portal SDK record.
    private struct PortalSDKRecord: Decodable {
        /// SDK identifier string (for example `IOS` or `MAC_OS`).
        let name: String
    }

    /// Portal entitlement record.
    private struct PortalEntitlementRecord: Decodable {
        /// Entitlement key.
        let profileKey: String

        /// Value type name.
        let valueType: String?

        /// Allowed values.
        let values: [PortalEntitlementValueRecord]?

        /// Optional supported SDKs for the entitlement key.
        let supportedSDKs: [PortalSDKRecord]?
    }

    /// Portal entitlement value record.
    private struct PortalEntitlementValueRecord: Decodable {
        /// Inference kind (`constant` or `wildcard`).
        let inference: String?

        /// Value string.
        let value: String?
    }
}
