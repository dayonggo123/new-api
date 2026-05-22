package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"strings"
)

const DefaultMaxImageBytes = 10 << 20 // 10MB

// CompressBase64Image compresses a base64 data URI image if it exceeds maxBytes.
// Returns the compressed data URI (or original if no compression needed).
func CompressBase64Image(dataURI string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxImageBytes
	}

	// Parse data URI: data:image/png;base64,xxxx
	prefix := "data:"
	if !strings.HasPrefix(dataURI, prefix) {
		// Not a data URI, return as-is
		return dataURI, nil
	}

	rest := strings.TrimPrefix(dataURI, prefix)
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return dataURI, nil
	}

	meta := rest[:commaIdx]
	b64data := rest[commaIdx+1:]

	mimeType := "application/octet-stream"
	if semiIdx := strings.Index(meta, ";"); semiIdx >= 0 {
		mimeType = meta[:semiIdx]
	} else {
		mimeType = meta
	}

	if !strings.HasPrefix(mimeType, "image/") {
		return dataURI, nil
	}

	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		// Try URL-safe base64
		data, err = base64.URLEncoding.DecodeString(b64data)
		if err != nil {
			return dataURI, fmt.Errorf("decode base64 failed: %w", err)
		}
	}

	if len(data) <= maxBytes {
		return dataURI, nil
	}

	compressed, newMime, err := compressImageBytes(data, mimeType, maxBytes)
	if err != nil {
		return dataURI, err
	}

	newURI := fmt.Sprintf("data:%s;base64,%s", newMime, base64.StdEncoding.EncodeToString(compressed))
	return newURI, nil
}

// compressImageBytes compresses image data to fit within maxBytes.
// Returns compressed bytes and the MIME type of the output.
func compressImageBytes(data []byte, mimeType string, maxBytes int) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image failed: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Strategy:
	// 1. Try JPEG with quality 85
	// 2. If still too large, scale down by 0.7 each iteration
	// 3. Max 5 iterations to avoid infinite loop
	scale := 1.0
	for i := 0; i < 5; i++ {
		if scale < 1.0 {
			newW := int(float64(width) * scale)
			newH := int(float64(height) * scale)
			if newW < 64 || newH < 64 {
				break
			}
			img = resizeImage(img, newW, newH)
		}

		var buf bytes.Buffer
		quality := 85
		if i == 0 && (mimeType == "image/jpeg" || mimeType == "image/jpg") {
			// First iteration: try lower quality for original JPEG
			quality = 75
		}

		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, "", fmt.Errorf("jpeg encode failed: %w", err)
		}

		if buf.Len() <= maxBytes {
			return buf.Bytes(), "image/jpeg", nil
		}

		scale *= 0.7
	}

	// Fallback: last attempt with quality 60
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 60})
	if err != nil {
		return nil, "", err
	}
	if buf.Len() <= maxBytes {
		return buf.Bytes(), "image/jpeg", nil
	}

	return nil, "", fmt.Errorf("unable to compress image below %d bytes", maxBytes)
}

// resizeImage scales an image to the specified dimensions using nearest neighbor.
func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * srcW / width
			srcY := y * srcH / height
			dst.Set(x, y, src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}
	return dst
}

// DetectImageSizeFromBase64 returns the decoded byte size of a base64 data URI.
func DetectImageSizeFromBase64(dataURI string) int {
	commaIdx := strings.Index(dataURI, ",")
	if commaIdx < 0 {
		return 0
	}
	b64data := dataURI[commaIdx+1:]
	// base64 encoded length * 3/4 is approximate decoded size
	return len(b64data) * 3 / 4
}

// IsDataURI checks if a string is a data URI.
func IsDataURI(s string) bool {
	return strings.HasPrefix(s, "data:")
}

// CompressImageInBodyMap compresses all base64 image fields in a body map.
// It looks for "images", "image" fields and compresses any base64 data URIs.
func CompressImageInBodyMap(bodyMap map[string]interface{}, maxBytes int) map[string]interface{} {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxImageBytes
	}

	// Process "images" array
	if imgs, ok := bodyMap["images"]; ok {
		switch v := imgs.(type) {
		case []interface{}:
			for i, img := range v {
				if imgStr, ok := img.(string); ok && IsDataURI(imgStr) {
					compressed, err := CompressBase64Image(imgStr, maxBytes)
					if err == nil {
						v[i] = compressed
					}
				}
			}
		case []string:
			for i, imgStr := range v {
				if IsDataURI(imgStr) {
					compressed, err := CompressBase64Image(imgStr, maxBytes)
					if err == nil {
						v[i] = compressed
					}
				}
			}
		}
	}

	// Process single "image" field
	if img, ok := bodyMap["image"].(string); ok && IsDataURI(img) {
		compressed, err := CompressBase64Image(img, maxBytes)
		if err == nil {
			bodyMap["image"] = compressed
		}
	}

	return bodyMap
}
