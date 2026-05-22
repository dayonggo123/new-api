package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"golang.org/x/image/webp"
)

const DefaultMaxImageBytes = 10 << 20 // 10MB

// CompressBase64Image compresses a base64 data URI image if it exceeds maxBytes.
// Returns the compressed data URI. If the original exceeds maxBytes and compression
// fails, returns an error instead of the oversized original.
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

	// Remove whitespace/newlines that some clients embed in base64
	b64data = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, b64data)

	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		// Try URL-safe base64
		data, err = base64.URLEncoding.DecodeString(b64data)
		if err != nil {
			return dataURI, fmt.Errorf("decode base64 failed: %w", err)
		}
	}

	if len(data) <= maxBytes {
		common.SysLog(fmt.Sprintf("[image_compress] no compression needed: size=%d bytes <= max=%d bytes", len(data), maxBytes))
		return dataURI, nil
	}

	compressed, newMime, err := CompressImageBytes(data, mimeType, maxBytes)
	if err != nil {
		return "", fmt.Errorf("compress image failed (original=%d bytes): %w", len(data), err)
	}

	newURI := fmt.Sprintf("data:%s;base64,%s", newMime, base64.StdEncoding.EncodeToString(compressed))
	ratio := float64(len(compressed)) / float64(len(data)) * 100
	common.SysLog(fmt.Sprintf("[image_compress] compressed: before=%d bytes, after=%d bytes, ratio=%.1f%%", len(data), len(compressed), ratio))
	return newURI, nil
}

// CompressImageBytes compresses image data to fit within maxBytes.
// Supports PNG, JPEG, GIF, WebP. Returns compressed bytes and the MIME type of the output.
func CompressImageBytes(data []byte, mimeType string, maxBytes int) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// Try webp decoder explicitly (some webp files may not be auto-detected)
		img, err = webp.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("decode image failed: %w", err)
		}
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Strategy:
	// 1. Try JPEG with decreasing quality and increasing scale reduction
	// 2. Max 12 iterations (scale down to ~0.7^12 ≈ 1.4%)
	// 3. Quality steps: 85, 80, 75, 70, 65, 60, 55, 50, 45, 40, 35, 30
	scale := 1.0
	qualities := []int{85, 80, 75, 70, 65, 60, 55, 50, 45, 40, 35, 30}

	for i, quality := range qualities {
		if i > 0 && scale < 1.0 {
			scale *= 0.7
			newW := int(float64(width) * scale)
			newH := int(float64(height) * scale)
			if newW < 32 || newH < 32 {
				break
			}
			img = resizeImage(img, newW, newH)
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, "", fmt.Errorf("jpeg encode failed: %w", err)
		}

		if buf.Len() <= maxBytes {
			common.SysLog(fmt.Sprintf("[image_compress] success at iteration %d (quality=%d, scale=%.3f): %d bytes", i, quality, scale, buf.Len()))
			return buf.Bytes(), "image/jpeg", nil
		}
	}

	return nil, "", fmt.Errorf("unable to compress image below %d bytes after %d attempts (original size %d, dims %dx%d)", maxBytes, len(qualities), len(data), width, height)
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
// If a single image exceeds maxBytes and compression fails, the error is logged
// and the original is kept (callers should validate before sending upstream).
// Logs compression results via common.SysLog for observability.
func CompressImageInBodyMap(bodyMap map[string]interface{}, maxBytes int) map[string]interface{} {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxImageBytes
	}

	// Debug: log entry and bodyMap keys
	var keys []string
	for k := range bodyMap {
		keys = append(keys, k)
	}
	common.SysLog(fmt.Sprintf("[image_compress] CompressImageInBodyMap called, keys=%v", keys))

	// Process "images" array
	if imgs, ok := bodyMap["images"]; ok {
		switch v := imgs.(type) {
		case []interface{}:
			for i, img := range v {
				if imgStr, ok := img.(string); ok && IsDataURI(imgStr) {
					compressed, err := CompressBase64Image(imgStr, maxBytes)
					if err == nil {
						v[i] = compressed
					} else {
						common.SysLog(fmt.Sprintf("[image_compress] compress failed for images[%d]: %v", i, err))
					}
				}
			}
		case []string:
			for i, imgStr := range v {
				if IsDataURI(imgStr) {
					compressed, err := CompressBase64Image(imgStr, maxBytes)
					if err == nil {
						v[i] = compressed
					} else {
						common.SysLog(fmt.Sprintf("[image_compress] compress failed for images[%d]: %v", i, err))
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
		} else {
			common.SysLog(fmt.Sprintf("[image_compress] compress failed for image field: %v", err))
		}
	}

	return bodyMap
}
