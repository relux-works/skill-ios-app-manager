package scaffold

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

var firebasePlistRequiredKeys = []string{
	"PROJECT_ID",
	"GOOGLE_APP_ID",
	"BUNDLE_ID",
	"API_KEY",
}

func init() {
	RegisterRuntimeProfilePlugin(&RuntimeProfilePlugin{
		Name:         "firebase-client-inputs",
		Short:        "Validate environment-keyed Firebase public client inputs without retaining them",
		Dependencies: []string{"runtime-profile-schema"},
		Sync:         syncRuntimeProfileFirebaseInputs,
	})
}

func syncRuntimeProfileFirebaseInputs(input RuntimeProfileInput) (RuntimeProfilePluginResult, error) {
	result := RuntimeProfilePluginResult{
		Name:    "firebase-client-inputs",
		Enabled: input.Config.HasRuntimeProfiles(),
	}
	if !result.Enabled {
		result.Message = "runtime profiles disabled"
		return result, nil
	}

	if err := ValidateFirebaseClientConfigurationInputs(input.ProjectRoot, input.Config); err != nil {
		return result, err
	}
	result.Message = "validated Firebase public metadata and local inputs without retention"
	return result, nil
}

// ValidateFirebaseClientConfigurationInputs reads operator-supplied plist paths
// only from the configured environment-variable hooks. It compares public
// metadata in memory and never writes, copies, or reports the path or API key.
func ValidateFirebaseClientConfigurationInputs(projectRoot string, cfg config.ProjectConfig) error {
	if !cfg.HasRuntimeProfiles() {
		return nil
	}

	for _, environment := range cfg.OrderedBackendEnvironments() {
		descriptor := cfg.RuntimeProfiles.BackendEnvironments[environment]
		if descriptor.Firebase == nil {
			continue
		}
		firebase := descriptor.Firebase
		hook := firebase.ValidationInputEnvironmentVar
		inputPath, ok := os.LookupEnv(hook)
		if !ok || strings.TrimSpace(inputPath) == "" {
			return fmt.Errorf("%s Firebase validation hook %q is not set", environment, hook)
		}
		if !filepath.IsAbs(inputPath) {
			inputPath = filepath.Join(projectRoot, inputPath)
		}

		payload, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("%s Firebase validation input from %q cannot be read", environment, hook)
		}
		metadata, err := decodeFirebasePlistMetadata(payload)
		if err != nil {
			return fmt.Errorf("%s Firebase validation input from %q is not a supported XML plist", environment, hook)
		}
		for _, key := range firebasePlistRequiredKeys {
			if strings.TrimSpace(metadata[key]) == "" {
				return fmt.Errorf("%s Firebase validation input from %q is missing %s", environment, hook, key)
			}
		}
		if metadata["PROJECT_ID"] != firebase.ProjectID {
			return fmt.Errorf("%s Firebase PROJECT_ID does not match configured public metadata", environment)
		}
		if metadata["GOOGLE_APP_ID"] != firebase.GoogleAppID {
			return fmt.Errorf("%s Firebase GOOGLE_APP_ID does not match configured public metadata", environment)
		}
		if metadata["BUNDLE_ID"] != firebase.BundleID {
			return fmt.Errorf("%s Firebase BUNDLE_ID does not match configured public metadata", environment)
		}
	}

	return nil
}

func decodeFirebasePlistMetadata(payload []byte) (map[string]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(payload))
	values := make(map[string]string, len(firebasePlistRequiredKeys))
	wanted := make(map[string]struct{}, len(firebasePlistRequiredKeys))
	for _, key := range firebasePlistRequiredKeys {
		wanted[key] = struct{}{}
	}

	var pendingKey string
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "key":
			var key string
			if err := decoder.DecodeElement(&key, &start); err != nil {
				return nil, err
			}
			if _, ok := wanted[key]; ok {
				pendingKey = key
			} else {
				pendingKey = ""
			}
		case "string":
			var value string
			if err := decoder.DecodeElement(&value, &start); err != nil {
				return nil, err
			}
			if pendingKey != "" {
				values[pendingKey] = value
				pendingKey = ""
			}
		}
	}

	return values, nil
}
