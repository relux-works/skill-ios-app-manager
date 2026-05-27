package template

import (
	"strconv"
)

func exportComplianceInfoPlistLines(usesNonExemptEncryption *bool) []string {
	if usesNonExemptEncryption == nil {
		return nil
	}

	return []string{
		strconv.Quote("ITSAppUsesNonExemptEncryption") + ": .boolean(" + strconv.FormatBool(*usesNonExemptEncryption) + "),",
	}
}
