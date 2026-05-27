package template

import (
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func privacyUsageDescriptionInfoPlistLines(privacy config.PrivacyUsageDescriptionsConfig) []string {
	lines := make([]string, 0, 2)

	if value := strings.TrimSpace(privacy.BluetoothAlways); value != "" {
		lines = append(lines, strconv.Quote("NSBluetoothAlwaysUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}
	if value := strings.TrimSpace(privacy.BluetoothPeripheral); value != "" {
		lines = append(lines, strconv.Quote("NSBluetoothPeripheralUsageDescription")+": .string("+strconv.Quote(value)+"),")
	}

	return lines
}
