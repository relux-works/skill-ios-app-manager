package scaffold

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// assetCatalogContentsJSON returns the root Contents.json for an Xcode asset catalog.
func assetCatalogContentsJSON() string {
	return `{
  "info" : {
    "author" : "xcode",
    "version" : 1
  }
}
`
}

// appIconsetContentsJSON returns the Contents.json for an AppIcon.appiconset
// with a single 1024x1024 universal iOS entry.
func appIconsetContentsJSON() string {
	return `{
  "images" : [
    {
      "filename" : "AppIcon.png",
      "idiom" : "universal",
      "platform" : "ios",
      "size" : "1024x1024"
    }
  ],
  "info" : {
    "author" : "xcode",
    "version" : 1
  }
}
`
}

// generatePlaceholderIcon creates a 1024x1024 solid-color PNG suitable
// as a default app icon placeholder.
func generatePlaceholderIcon() ([]byte, error) {
	const size = 1024
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fill := color.RGBA{R: 88, G: 86, B: 214, A: 255} // Indigo
	draw.Draw(img, img.Bounds(), &image.Uniform{C: fill}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode placeholder icon: %w", err)
	}
	return buf.Bytes(), nil
}
