package capability_files

import "embed"

// CapabilityFiles embeds the Capability DSL Swift files for use by the scaffolding pipeline.
//
// Embedded files:
//   - Capability.swift — full capability type system
//   - EntitlementsFactory.swift — converts [Capability] → Tuist Entitlements
//   - Capability+PortalCapability.swift — 118 Apple portal capabilities enum
//   - AppleSupportedCapabilities.swift — per-platform capability catalog
//
//go:embed Capability.swift EntitlementsFactory.swift Capability+PortalCapability.swift AppleSupportedCapabilities.swift
var CapabilityFiles embed.FS
