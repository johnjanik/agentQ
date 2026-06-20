package image

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// forgePNGWithDimensions builds a PNG containing only a signature and an IHDR
// chunk declaring the given dimensions. image.DecodeConfig reads the IHDR
// without allocating pixels, so this cheaply simulates an image bomb.
func forgePNGWithDimensions(w, h uint32) string {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) // signature

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type: RGBA

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(ihdr)))
	buf.Write(length[:])
	buf.WriteString("IHDR")
	buf.Write(ihdr)

	crc := crc32.NewIEEE()
	crc.Write([]byte("IHDR"))
	crc.Write(ihdr)
	var crcb [4]byte
	binary.BigEndian.PutUint32(crcb[:], crc.Sum32())
	buf.Write(crcb[:])

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func createTestImageBase64(w, h int) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestImageService(t *testing.T) {
	s := New()

	t.Run("ResizeValid", func(t *testing.T) {
		input := createTestImageBase64(100, 100)
		output, err := s.ResizeBase64(input, 50, 50)
		if err != nil {
			t.Fatalf("failed to resize image: %v", err)
		}
		if output == "" {
			t.Fatal("got empty output")
		}
	})

	t.Run("NotABase64Image", func(t *testing.T) {
		input := "not a base64 image"
		_, err := s.ResizeBase64(input, 50, 50)
		if err == nil {
			t.Error("expected error for non-base64 image, got nil")
		}
	})

	t.Run("NotSquare", func(t *testing.T) {
		input := createTestImageBase64(100, 50)
		_, err := s.ResizeBase64(input, 50, 50)
		if err == nil {
			t.Error("expected error for non-square image, got nil")
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		input := "data:image/png;base64,invalid-base64"
		_, err := s.ResizeBase64(input, 50, 50)
		if err == nil {
			t.Error("expected error for invalid base64, got nil")
		}
	})
	
	t.Run("DecodeError", func(t *testing.T) {
		input := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not an image"))
		_, err := s.ResizeBase64(input, 50, 50)
		if err == nil {
			t.Error("expected error for invalid image data, got nil")
		}
	})

	// SECURITY-REVIEW.md #5: reject images whose declared dimensions exceed the
	// pixel budget before allocating the full image (image-bomb guard). A
	// 20000x20000 PNG is small when run-length compressed but would allocate
	// ~1.6GB if decoded.
	t.Run("DimensionsTooLarge", func(t *testing.T) {
		input := forgePNGWithDimensions(maxImageDim+1, maxImageDim+1)
		_, err := s.ResizeBase64(input, 50, 50)
		if err == nil {
			t.Error("expected error for oversized image dimensions, got nil")
		}
	})
}
