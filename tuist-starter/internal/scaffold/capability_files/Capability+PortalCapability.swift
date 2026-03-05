/// Portal capability identifiers derived from Xcode.
///
/// This file is generated from Xcode's bundled portal capability definitions.
/// Do not edit by hand; regenerate from the current Xcode installation.
///
/// Source: `DVTPortalCachedPortalCapabilities.json` inside the active Xcode installation.
extension Capability {
    /// Apple Developer portal "App ID capabilities" (a.k.a. managed capabilities).
    ///
    /// Use these when you need to enable an Apple service via entitlements but don't want
    /// to model a dedicated high-level API yet. Some capabilities require extra configuration
    /// (for example, selecting values or providing identifiers) and will be rejected by
    /// `EntitlementsFactory` unless a dedicated helper exists.
    public enum PortalCapability: String, CaseIterable, Hashable, Sendable {
        case accessWiFiInformation = "ACCESS_WIFI_INFORMATION" // Access Wi-Fi Information
        case accessibilityMerchantApiControl = "MERCHANT_ACCESSIBILITY_CONTROL" // Accessibility Merchant API Control
        case accessorySetupExtension = "ACCESSORY_SETUP_KIT" // Accessory Setup Extension
        case appAttest = "APP_ATTEST" // App Attest
        case appAttestOptIn = "APP_ATTEST_OPT_IN" // App Attest Opt-In
        case appGroups = "APP_GROUPS" // App Groups
        case applePayLaterMerchandising = "APPLE_PAY_LATER_MERCHANDISING" // Apple Pay Later Merchandising
        case applePayPaymentProcessing = "APPLE_PAY" // Apple Pay Payment Processing
        case associatedDomains = "ASSOCIATED_DOMAINS" // Associated Domains
        case autoFillCredentialProvider = "AUTOFILL_CREDENTIAL_PROVIDER" // AutoFill Credential Provider
        case backgroundGpuAccess = "BACKGROUND_GPU_ACCESS" // Background GPU Access
        case carrierConstrainedNetworkCategoryAndOptimized = "CARRIER_CONSTRAINED_NETWORK_CAT_OPT" // Carrier-Constrained Network Category and Optimized
        case classKit = "CLASSKIT" // ClassKit
        case communicationNotifications = "USERNOTIFICATIONS_COMMUNICATION" // Communication Notifications
        case criticalMessaging = "CRITICAL_MESSAGING" // Critical Messaging
        case customNetworkProtocol = "NETWORK_CUSTOM_PROTOCOL" // Custom Network Protocol
        case dataProtection = "DATA_PROTECTION" // Data Protection
        case declaredAge = "DECLARED_AGE_RANGE" // Declared Age
        case defaultCallingApp = "DEFAULT_CALLING_APP" // Default Calling App
        case defaultCarrierMessagingApp = "DEFAULT_CARRIER_MESSAGING_APP" // Default Carrier Messaging App
        case defaultDialerApp = "DIALING_APP" // Default Dialer App
        case defaultMessagingApp = "DEFAULT_MESSAGING_APP" // Default Messaging App
        case defaultNavigationApp = "DEFAULT_NAVIGATION_APP" // Default Navigation App
        case defaultTranslationApp = "DEFAULT_TRANSLATION_APP" // Default Translation App
        case deviceDiscoveryPairingAccess = "DEVICE_DISCOVERY_PAIRING" // Device Discovery Pairing Access
        case digitalCredentialsApiMobileDocumentProvider = "MOBILE_DOCUMENT_PROVIDER" // Digital Credentials API - Mobile Document Provider
        case driverKit = "DRIVERKIT_PUBLIC" // DriverKit (development)
        case driverKitAllowThirdPartyUserClients = "DRIVERKIT_ALLOWTHIRDPARTY_USERCLIENTS" // DriverKit Allow Third Party UserClients
        case driverKitCommunicatesWithDrivers = "DRIVERKIT_COMMUNICATESWITHDRIVERS" // DriverKit Communicates with Drivers
        case driverKitFamilyAudio = "DRIVERKIT_FAMILY_AUDIO_PUB" // DriverKit Family Audio (development)
        case driverKitFamilyHidDevice = "DRIVERKIT_FAMILY_HIDDEVICE_PUB" // DriverKit Family HID Device (development)
        case driverKitFamilyHidEventService = "DRIVERKIT_FAMILY_HIDEVENTSERVICE_PUB" // DriverKit Family HID EventService (development)
        case driverKitFamilyMidi = "DRIVERKIT_FAMILY_MIDI_DEV" // DriverKit Family MIDI (development)
        case driverKitFamilyNetworking = "DRIVERKIT_FAMILY_NETWORKING_PUB" // DriverKit Family Networking (development)
        case driverKitFamilySCSIController = "DRIVERKIT_FAMILY_SCSICONTROLLER_PUB" // DriverKit Family SCSIController (development)
        case driverKitFamilySerial = "DRIVERKIT_FAMILY_SERIAL_PUB" // DriverKit Family Serial (development)
        case driverKitTransportHid = "DRIVERKIT_TRANSPORT_HID_PUB" // DriverKit Transport HID (development)
        case driverKitUsbTransport = "DRIVERKIT_USBTRANSPORT_PUB" // DriverKit USB Transport (development)
        case energyKit = "ENERGYKIT" // EnergyKit (Development Only)
        case extendedVirtualAddressing = "EXTENDED_VIRTUAL_ADDRESSING" // Extended Virtual Addressing
        case familyControls = "FAMILY_CONTROLS" // Family Controls (Development)
        case fileProviderTestingMode = "FILEPROVIDER_TESTINGMODE" // FileProvider Testing Mode
        case financeKitTransactionPickerUi = "FINANCEKIT_TRANSACTION_PICKER" // FinanceKit Transaction Picker UI
        case fiveGNetworkSlicing = "NETWORK_SLICING" // 5G Network Slicing
        case fonts = "FONT_INSTALLATION" // Fonts
        case fsKitModule = "FSKIT_MODULE" // FSKit Module
        case gameCenter = "GAME_CENTER" // Game Center
        case groupActivities = "GROUP_ACTIVITIES" // Group Activities
        case hardenedProcess = "ENHANCED_SECURITY" // Hardened Process
        case headPose = "COREMOTION_HEAD_POSE" // Head Pose
        case healthKit = "HEALTHKIT" // HealthKit
        case healthKitEstimateRecalibration = "HEALTHKIT_RECALIBRATE_ESTIMATES" // HealthKit Estimate Recalibration
        case hlsInterstitialPreviews = "HLS_INTERSTITIAL_PREVIEW" // HLS Interstitial Previews
        case homeKit = "HOMEKIT" // HomeKit
        case hotspot = "HOT_SPOT" // Hotspot
        case iCloud = "ICLOUD" // iCloud
        case iCloudExtendedShareAccess = "ICLOUD_EXTENDED_SHARE_ACCESS" // iCloud Extended Share Access
        case idVerifierDisplayOnly = "TAP_TO_DISPLAY_ID" // ID Verifier - Display Only
        case inAppPurchase = "IN_APP_PURCHASE" // In-App Purchase
        case increasedDebuggingMemoryLimit = "INCREASED_MEMORY_LIMIT_DEBUGGING" // Increased Debugging Memory Limit
        case increasedMemoryLimit = "INCREASED_MEMORY_LIMIT" // Increased Memory Limit
        case interAppAudio = "INTER_APP_AUDIO" // Inter-App Audio
        case journalingSuggestions = "JOURNALING_SUGGESTIONS" // Journaling Suggestions
        case locationPushServiceExtension = "LOCATION_PUSH_SERVICE_EXT" // Location Push Service Extension
        case lowLatencyHLS = "COREMEDIA_HLS_LOW_LATENCY" // Low Latency HLS
        case lowLatencyStreaming = "LOW_LATENCY_STREAMING" // Low-Latency Streaming
        case macCatalyst = "ENABLED_FOR_MAC" // Mac Catalyst
        case manageThreadNetworkCredentials = "DEV_MANAGE_THREAD_NETWORK_CREDENTIALS" // Manage Thread Network Credentials (development)
        case managedAppInstallationUi = "MANAGED_APP_INSTALLATION_UI" // Managed App Installation UI
        case maps = "MAPS" // Maps
        case matterAllowSetupPayload = "MATTER_ALLOW_SETUP_PAYLOAD" // Matter Allow Setup Payload
        case mdmManagedAssociatedDomains = "MDM_MANAGED_ASSOCIATED_DOMAINS" // MDM Managed Associated Domains
        case mediaDeviceDiscovery = "MEDIA_DEVICE_DISCOVERY" // Media Device Discovery
        case mediaExtensionFormatReader = "MEDIAEXTENSION_FORMATREADER" // Media Extension Format Reader
        case mediaExtensionVideoDecoder = "MEDIAEXTENSION_VIDEODECODER" // Media Extension Video Decoder
        case messagesCollaboration = "MESSAGES_COLLABORATION" // Messages Collaboration
        case multipath = "MULTIPATH" // Multipath
        case multitaskingCameraAccess = "IPAD_CAMERA_MULTITASKING" // Multitasking Camera Access
        case networkExtensions = "NETWORK_EXTENSIONS" // Network Extensions
        case nfcTagReading = "NFC_TAG_READING" // NFC Tag Reading
        case onDemandInstallCapable = "ON_DEMAND_INSTALL_CAPABLE" // On Demand Install Capable
        case onDemandInstallCapableForAppClipExtensions = "ONDEMANDINSTALL_EXTENSIONS" // On Demand Install Capable for App Clip Extensions
        case personalVPN = "PERSONAL_VPN" // Personal VPN
        case pushNotifications = "PUSH_NOTIFICATIONS" // Push Notifications
        case pushToTalk = "PUSH_TO_TALK" // Push to Talk
        case sensitiveContentAnalysis = "SENSITIVE_CONTENT_ANALYSIS" // Sensitive Content Analysis
        case shallowDepthAndPressure = "SHALLOW_DEPTH_PRESSURE" // Shallow Depth and Pressure
        case sharedWithYou = "SHARED_WITH_YOU" // Shared with You
        case signInWithApple = "APPLE_ID_AUTH" // Sign In with Apple
        case simInsertedForWirelessCarriers = "CORETELEPONY_SIMINSERTED" // SIM Inserted for Wireless Carriers
        case siri = "SIRIKIT" // Siri
        case spatialAudioProfile = "SPATIAL_AUDIO_PROFILE" // Spatial Audio Profile
        case sustainedExecution = "SUSTAINED_EXECUTION" // Sustained Execution
        case systemExtension = "SYSTEM_EXTENSION_INSTALL" // System Extension
        case timeSensitiveNotifications = "USERNOTIFICATIONS_TIMESENSITIVE" // Time Sensitive Notifications
        case userManagement = "USER_MANAGEMENT" // User Management
        case vmNet = "VMNET" // VMNet
        case wallet = "WALLET" // Wallet
        case weatherKit = "WEATHERKIT" // WeatherKit
        case wiFiAware = "WIFI_AWARE" // Wi-Fi Aware
        case wirelessAccessoryConfiguration = "WIRELESS_ACCESSORY_CONFIGURATION" // Wireless Accessory Configuration
        case wirelessInsightsServicePredictions = "WIRELESS_INSIGHTS_SERVICE_PREDICTIONS" // Wireless Insights Service Predictions
    }
}
