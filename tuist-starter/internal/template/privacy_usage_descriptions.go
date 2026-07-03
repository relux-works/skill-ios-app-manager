package template

import (
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func privacyUsageDescriptionInfoPlistLines(privacy config.PrivacyUsageDescriptionsConfig) []string {
	lines := make([]string, 0, 5)

	if value := strings.TrimSpace(privacy.BluetoothAlways); value != "" {
		lines = append(lines, strconv.Quote("NSBluetoothAlwaysUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}
	if value := strings.TrimSpace(privacy.BluetoothPeripheral); value != "" {
		lines = append(lines, strconv.Quote("NSBluetoothPeripheralUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}
	if value := strings.TrimSpace(privacy.Camera); value != "" {
		lines = append(lines, strconv.Quote("NSCameraUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}
	if value := strings.TrimSpace(privacy.Microphone); value != "" {
		lines = append(lines, strconv.Quote("NSMicrophoneUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}
	if value := strings.TrimSpace(privacy.LocalNetwork); value != "" {
		lines = append(lines, strconv.Quote("NSLocalNetworkUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}

	return lines
}
