package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"golang.org/x/image/draw"
)

// Limits guarding against decompression/image bombs: a few KB of input can
// otherwise declare enormous dimensions and force a huge pixel allocation.
const (
	maxImageBytes  = 10 << 20 // 10 MiB of decoded input
	maxImageDim    = 8192     // max width or height in pixels
	maxImagePixels = 1 << 26  // total pixel budget (8192*8192)
)

type Service interface {
	ResizeBase64(dataBase64 string, width, height int) (string, error)
}

type service struct{}

func New() Service {
	return &service{}
}

func (s *service) ResizeBase64(dataBase64 string, width, height int) (string, error) {
	if !strings.HasPrefix(dataBase64, "data:image/") {
		return "", fmt.Errorf("invalid image format: missing data:image/ prefix")
	}

	parts := strings.SplitN(dataBase64, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64 image format")
	}

	meta := parts[0]
	raw := parts[1]

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	if len(decoded) > maxImageBytes {
		return "", fmt.Errorf("image too large: %d bytes (max %d)", len(decoded), maxImageBytes)
	}

	// Inspect dimensions cheaply (no full pixel allocation) before decoding, to
	// reject image bombs that declare enormous sizes.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(decoded))
	if err != nil {
		return "", fmt.Errorf("decode image config: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 ||
		cfg.Width > maxImageDim || cfg.Height > maxImageDim ||
		int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return "", fmt.Errorf("image dimensions too large: %dx%d", cfg.Width, cfg.Height)
	}

	img, fmtName, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	// Validate squareness
	bounds := img.Bounds()
	if bounds.Dx() != bounds.Dy() {
		return "", fmt.Errorf("image must be square (current: %dx%d)", bounds.Dx(), bounds.Dy())
	}

	// Resizing
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	switch fmtName {
	case "png":
		err = png.Encode(&buf, dst)
	case "jpeg":
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 100})
	default:
		// Fallback to PNG if unknown (should not happen with standard image/xxx imports)
		err = png.Encode(&buf, dst)
	}

	if err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	// Keep the same mime type in meta part, or update it to be safe
	return meta + "," + encoded, nil
}
