package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/chai2010/webp"
	xwebp "golang.org/x/image/webp"
)

const (
	maxDimension = 2048
	webpQuality  = 82
)

// ProcessImage decodes an image from any supported format, optionally resizes it
// so neither dimension exceeds maxDimension, and re-encodes it as WebP.
// Returns the WebP bytes and the actual pixel dimensions.
func ProcessImage(r io.Reader, mimeType string) (data []byte, width, height int, err error) {
	var img image.Image

	switch mimeType {
	case "image/jpeg":
		img, err = jpeg.Decode(r)
	case "image/png":
		img, err = png.Decode(r)
	case "image/webp":
		img, err = xwebp.Decode(r)
	default:
		img, _, err = image.Decode(r)
	}
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decoding image: %w", err)
	}

	img = resizeIfNeeded(img, maxDimension)

	bounds := img.Bounds()
	width = bounds.Dx()
	height = bounds.Dy()

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: webpQuality}); err != nil {
		return nil, 0, 0, fmt.Errorf("encoding to webp: %w", err)
	}

	return buf.Bytes(), width, height, nil
}

// resizeIfNeeded scales down img so neither dimension exceeds maxDim, preserving aspect ratio.
// Uses nearest-neighbour for speed; quality is sufficient given we're encoding to lossy WebP.
func resizeIfNeeded(img image.Image, maxDim int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}

	scaleW := float64(maxDim) / float64(w)
	scaleH := float64(maxDim) / float64(h)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}

	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	dst := image.NewNRGBA(image.Rect(0, 0, newW, newH))
	for y := range newH {
		for x := range newW {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			dst.Set(x, y, img.At(srcX+b.Min.X, srcY+b.Min.Y))
		}
	}
	return dst
}
