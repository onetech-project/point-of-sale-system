package services

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

// ImageProcessor handles image validation and optimization
type ImageProcessor struct {
	maxSizeBytes int64 // Maximum file size in bytes
	maxWidth     int   // Maximum width in pixels
	maxHeight    int   // Maximum height in pixels
}

// NewImageProcessor creates a new ImageProcessor
func NewImageProcessor(maxSizeBytes int64, maxWidth, maxHeight int) *ImageProcessor {
	return &ImageProcessor{
		maxSizeBytes: maxSizeBytes,
		maxWidth:     maxWidth,
		maxHeight:    maxHeight,
	}
}

// ImageMetadata contains image metadata
type ImageMetadata struct {
	Width    int
	Height   int
	MimeType string
	Size     int64
}

// ValidateImage validates image file size, type, and dimensions
func (p *ImageProcessor) ValidateImage(reader io.Reader) (*ImageMetadata, []byte, error) {
	// Read file content into memory
	buf := new(bytes.Buffer)
	size, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Validate file size
	if size > p.maxSizeBytes {
		return nil, nil, fmt.Errorf("file size %d bytes exceeds maximum %d bytes", size, p.maxSizeBytes)
	}

	// Decode image to get dimensions and format
	img, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		// Try webp manually since it might not be registered
		if strings.Contains(err.Error(), "unknown format") {
			img, err = webp.Decode(bytes.NewReader(buf.Bytes()))
			if err == nil {
				format = "webp"
			} else {
				return nil, nil, fmt.Errorf("unsupported image format: %w", err)
			}
		} else {
			return nil, nil, fmt.Errorf("failed to decode image: %w", err)
		}
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Validate dimensions
	if width > p.maxWidth || height > p.maxHeight {
		return nil, nil, fmt.Errorf("image dimensions %dx%d exceed maximum %dx%d", width, height, p.maxWidth, p.maxHeight)
	}

	if width == 0 || height == 0 {
		return nil, nil, fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	// Convert format to MIME type
	mimeType := formatToMimeType(format)
	if mimeType == "" {
		return nil, nil, fmt.Errorf("unsupported image format: %s", format)
	}

	metadata := &ImageMetadata{
		Width:    width,
		Height:   height,
		MimeType: mimeType,
		Size:     size,
	}

	return metadata, buf.Bytes(), nil
}

// OptimizeImage performs image optimization to reduce file size while maintaining quality
// Optimizations applied:
// - JPEG: Re-encode at 85% quality
// - PNG: Re-encode with default compression
// - GIF: Return original (optimization would lose animation)
// - WebP: Return original (already optimized)
// - Resize if dimensions exceed reasonable display sizes (max 2048x2048)
func (p *ImageProcessor) OptimizeImage(imageData []byte, mimeType string) ([]byte, error) {
	// For GIF and WebP, return original data
	// GIF optimization would require complex frame-by-frame processing
	// WebP is already an optimized format
	if mimeType == "image/gif" || mimeType == "image/webp" {
		return imageData, nil
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		// If decode fails, return original data (better than failing the upload)
		return imageData, nil
	}

	// Check if resizing is needed (images larger than 2048px on either dimension)
	const maxDisplaySize = 2048
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width > maxDisplaySize || height > maxDisplaySize {
		// Calculate new dimensions maintaining aspect ratio
		if width > height {
			height = height * maxDisplaySize / width
			width = maxDisplaySize
		} else {
			width = width * maxDisplaySize / height
			height = maxDisplaySize
		}
		// Resize using Lanczos filter (high quality)
		img = imaging.Resize(img, width, height, imaging.Lanczos)
	}

	// Re-encode based on format
	buf := new(bytes.Buffer)

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		// Re-encode JPEG at 85% quality (good balance of quality/size)
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
		if err != nil {
			return imageData, nil // Return original on error
		}

	case "png":
		// Re-encode PNG with default compression (level 6)
		err = png.Encode(buf, img)
		if err != nil {
			return imageData, nil // Return original on error
		}

	default:
		// Unknown format, return original
		return imageData, nil
	}

	optimizedData := buf.Bytes()

	// Only use optimized version if it's actually smaller
	// In some cases, re-encoding might increase file size
	if len(optimizedData) < len(imageData) {
		return optimizedData, nil
	}

	return imageData, nil
}

// formatToMimeType converts image format string to MIME type
func formatToMimeType(format string) string {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return ""
	}
}

// IsSupportedMimeType checks if a MIME type is supported
func IsSupportedMimeType(mimeType string) bool {
	supported := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	return supported[strings.ToLower(mimeType)]
}
