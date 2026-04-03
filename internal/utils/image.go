package utils

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

// GetImageBrightness calculates the average brightness of an image.
// Returns a value between 0.0 (black) and 1.0 (white).
func GetImageBrightness(data []byte) (float64, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Sample pixels to save CPU
	const maxSamples = 1000
	step := 1
	if width*height > maxSamples {
		step = max(int(math.Sqrt(float64(width*height)/float64(maxSamples))), 1)
	}

	var totalBrightness float64
	var count float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()

			// Use standard luminance formula: 0.299*R + 0.587*G + 0.114*B
			// RGBA() returns values in range [0, 65535]
			brightness := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
			totalBrightness += brightness
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	return totalBrightness / count, nil
}

// IsImageTooDark returns true if the image brightness is below the threshold.
func IsImageTooDark(data []byte, threshold float64) bool {
	brightness, err := GetImageBrightness(data)
	if err != nil {
		return false
	}
	return brightness < threshold
}
