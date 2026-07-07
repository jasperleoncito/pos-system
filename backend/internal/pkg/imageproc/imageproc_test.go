package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makePNG renders deterministic photo-like noise. Noise defeats PNG's
// lossless compression, so the lossy-WebP-is-smaller assertion holds the
// way it does for real photos.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	seed := uint32(2463534242)
	next := func() uint32 { // xorshift32 — deterministic pseudo-noise
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return seed
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := next()
			img.Set(x, y, color.RGBA{R: uint8(v), G: uint8(v >> 8), B: uint8(v >> 16), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestOptimizeResizesLargeImage(t *testing.T) {
	// Arrange: 2400x1200 — larger than the 1600px cap.
	data := makePNG(t, 2400, 1200)

	// Act
	result, err := Optimize(data)

	// Assert
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if result.Width != 1600 || result.Height != 800 {
		t.Errorf("expected 1600x800, got %dx%d", result.Width, result.Height)
	}
	if len(result.WebP) == 0 || len(result.ThumbWebP) == 0 {
		t.Fatal("expected non-empty webp outputs")
	}
	if len(result.WebP) >= len(data) {
		t.Errorf("optimized (%d bytes) should be smaller than original PNG (%d bytes)", len(result.WebP), len(data))
	}
	// WebP magic: RIFF....WEBP
	if string(result.WebP[:4]) != "RIFF" || string(result.WebP[8:12]) != "WEBP" {
		t.Error("main output is not a WebP file")
	}
}

func TestOptimizeKeepsSmallImageDimensions(t *testing.T) {
	data := makePNG(t, 400, 300)

	result, err := Optimize(data)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if result.Width != 400 || result.Height != 300 {
		t.Errorf("small image must not be upscaled: got %dx%d", result.Width, result.Height)
	}
}

func TestDecodeRejectsOversizedUpload(t *testing.T) {
	huge := make([]byte, MaxUploadBytes+1)
	if _, err := Decode(huge); err == nil {
		t.Error("expected error for >10MB upload")
	}
}

func TestDecodeRejectsNonImage(t *testing.T) {
	if _, err := Decode([]byte("definitely not an image")); err == nil {
		t.Error("expected error for non-image data")
	}
}

func TestDecodeRejectsEmpty(t *testing.T) {
	if _, err := Decode(nil); err == nil {
		t.Error("expected error for empty data")
	}
}

func TestFaviconsGeneratesAllSizes(t *testing.T) {
	img, err := Decode(makePNG(t, 640, 480))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	favicons, err := Favicons(img)
	if err != nil {
		t.Fatalf("Favicons: %v", err)
	}

	for _, size := range FaviconSizes {
		data, ok := favicons[size]
		if !ok || len(data) == 0 {
			t.Errorf("missing favicon %dpx", size)
			continue
		}
		decoded, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Errorf("favicon %dpx is not valid PNG: %v", size, err)
			continue
		}
		if decoded.Bounds().Dx() != size || decoded.Bounds().Dy() != size {
			t.Errorf("favicon %dpx has wrong dimensions %v", size, decoded.Bounds())
		}
	}
}
