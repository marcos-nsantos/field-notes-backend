package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

const (
	MaxImageWidth  = 2048
	MaxImageHeight = 2048
	JPEGQuality    = 85
)

type ImageProcessorImpl struct {
	maxWidth  int
	maxHeight int
	quality   int
}

func NewImageProcessor() *ImageProcessorImpl {
	return &ImageProcessorImpl{
		maxWidth:  MaxImageWidth,
		maxHeight: MaxImageHeight,
		quality:   JPEGQuality,
	}
}

func (p *ImageProcessorImpl) Process(reader io.Reader) (io.Reader, int64, int, int, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("reading image: %w", err)
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return bytes.NewReader(data), int64(len(data)), 0, 0, nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	needsResize := width > p.maxWidth || height > p.maxHeight

	if !needsResize && (format == "jpeg" || format == "jpg") {
		return bytes.NewReader(data), int64(len(data)), width, height, nil
	}

	if needsResize {
		img = imaging.Fit(img, p.maxWidth, p.maxHeight, imaging.Lanczos)
		bounds = img.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	var buf bytes.Buffer

	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("encoding png: %w", err)
		}
	default:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: p.quality}); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("encoding jpeg: %w", err)
		}
	}

	return bytes.NewReader(buf.Bytes()), int64(buf.Len()), width, height, nil
}
