package scaffold

import (
	"bytes"
	"encoding/json"
	"image/png"
	"testing"
)

func TestAssetCatalogContentsJSON(t *testing.T) {
	t.Parallel()

	content := assetCatalogContentsJSON()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("assetCatalogContentsJSON() produced invalid JSON: %v", err)
	}

	info, ok := parsed["info"].(map[string]interface{})
	if !ok {
		t.Fatal("missing info object in Contents.json")
	}
	if info["author"] != "xcode" {
		t.Fatalf("info.author = %v, want xcode", info["author"])
	}
	if info["version"] != float64(1) {
		t.Fatalf("info.version = %v, want 1", info["version"])
	}
}

func TestAppIconsetContentsJSON(t *testing.T) {
	t.Parallel()

	content := appIconsetContentsJSON()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("appIconsetContentsJSON() produced invalid JSON: %v", err)
	}

	images, ok := parsed["images"].([]interface{})
	if !ok || len(images) == 0 {
		t.Fatal("missing images array in AppIcon Contents.json")
	}

	entry, ok := images[0].(map[string]interface{})
	if !ok {
		t.Fatal("images[0] is not an object")
	}
	if entry["filename"] != "AppIcon.png" {
		t.Fatalf("images[0].filename = %v, want AppIcon.png", entry["filename"])
	}
	if entry["size"] != "1024x1024" {
		t.Fatalf("images[0].size = %v, want 1024x1024", entry["size"])
	}
	if entry["platform"] != "ios" {
		t.Fatalf("images[0].platform = %v, want ios", entry["platform"])
	}
}

func TestGeneratePlaceholderIcon(t *testing.T) {
	t.Parallel()

	data, err := generatePlaceholderIcon()
	if err != nil {
		t.Fatalf("generatePlaceholderIcon() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("generatePlaceholderIcon() returned empty data")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("PNG decode error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 1024 || bounds.Dy() != 1024 {
		t.Fatalf("icon size = %dx%d, want 1024x1024", bounds.Dx(), bounds.Dy())
	}
}
