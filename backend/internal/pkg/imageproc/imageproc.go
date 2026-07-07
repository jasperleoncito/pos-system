// Package imageproc implements the mandatory image optimization pipeline:
// validate → decode → resize → re-encode as lossy WebP (+ thumbnail).
// Re-encoding from decoded pixels inherently strips all metadata (EXIF,
// GPS, ICC). Originals are never stored.
package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"

	_ "image/jpeg" // register JPEG decoder

	"github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

const (
	// MaxUploadBytes is the hard upload cap from the product spec.
	MaxUploadBytes = 10 << 20 // 10 MB

	maxMainDimension  = 1600
	thumbnailMaxSide  = 300
	lossyWebPQuality  = 80
	thumbWebPQuality  = 75
	faviconPNGMaxSide = 512
)

// FaviconSizes are the square PNG favicon variants generated for logos.
var FaviconSizes = []int{16, 32, 180, 512}

// Result is the output of the optimization pipeline.
type Result struct {
	// WebP is the optimized main image (max 1600px, quality 80).
	WebP []byte
	// ThumbWebP is a small preview (max 300px).
	ThumbWebP []byte
	Width     int
	Height    int
}

// allowedFormats maps Go image-registry format names accepted for upload.
var allowedFormats = map[string]bool{"png": true, "jpeg": true, "webp": true}

// Decode validates size/format and decodes the upload into pixels.
func Decode(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, apperror.Validation("image file is empty")
	}
	if len(data) > MaxUploadBytes {
		return nil, apperror.Validation("image exceeds the 10MB upload limit")
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, apperror.Validation("unsupported image — use PNG, JPG, or WEBP")
	}
	if !allowedFormats[strings.ToLower(format)] {
		return nil, apperror.Validation("unsupported image format — use PNG, JPG, or WEBP")
	}
	return img, nil
}

// Optimize runs the full pipeline on an uploaded file.
func Optimize(data []byte) (*Result, error) {
	img, err := Decode(data)
	if err != nil {
		return nil, err
	}

	main := resizeToFit(img, maxMainDimension)
	mainBytes, err := encodeWebP(main, lossyWebPQuality)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "failed to encode image", err)
	}

	thumb := resizeToFit(img, thumbnailMaxSide)
	thumbBytes, err := encodeWebP(thumb, thumbWebPQuality)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "failed to encode thumbnail", err)
	}

	bounds := main.Bounds()
	return &Result{
		WebP:      mainBytes,
		ThumbWebP: thumbBytes,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
	}, nil
}

// Favicons renders square PNG favicons (transparent letterboxing) from a
// decoded logo, keyed by pixel size.
func Favicons(img image.Image) (map[int][]byte, error) {
	base := resizeToFit(img, faviconPNGMaxSide)
	out := make(map[int][]byte, len(FaviconSizes))
	for _, size := range FaviconSizes {
		square := squareFit(base, size)
		var buf bytes.Buffer
		if err := png.Encode(&buf, square); err != nil {
			return nil, fmt.Errorf("failed to encode %dpx favicon: %w", size, err)
		}
		out[size] = buf.Bytes()
	}
	return out, nil
}

// resizeToFit scales down so the longest side is maxSide; never upscales.
func resizeToFit(img image.Image, maxSide int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return img
	}
	scale := float64(maxSide) / float64(max(w, h))
	nw, nh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

// squareFit centers the image inside a transparent size×size canvas.
func squareFit(img image.Image, size int) image.Image {
	fitted := resizeToFit(img, size)
	fb := fitted.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	offset := image.Pt((size-fb.Dx())/2, (size-fb.Dy())/2)
	draw.Draw(dst, fb.Add(offset).Sub(fb.Min), fitted, fb.Min, draw.Over)
	return dst
}

func encodeWebP(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: quality, Method: 4}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
