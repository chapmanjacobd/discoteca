package utils_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func createTestImage(brightness uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := range 100 {
		for x := range 100 {
			img.Set(x, y, color.RGBA{brightness, brightness, brightness, 255})
		}
	}
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, nil)
	return buf.Bytes()
}

func TestGetImageBrightness(t *testing.T) {
	// Black image
	black := createTestImage(0)
	b1, _ := utils.GetImageBrightness(black)
	if b1 > 0.01 {
		t.Errorf("Expected black image brightness near 0, got %f", b1)
	}

	// White image
	white := createTestImage(255)
	b2, _ := utils.GetImageBrightness(white)
	if b2 < 0.99 {
		t.Errorf("Expected white image brightness near 1, got %f", b2)
	}

	// Mid-gray image
	gray := createTestImage(128)
	b3, _ := utils.GetImageBrightness(gray)
	if b3 < 0.45 || b3 > 0.55 {
		t.Errorf("Expected gray image brightness near 0.5, got %f", b3)
	}
}

func TestIsImageTooDark(t *testing.T) {
	dark := createTestImage(10) // ~4% brightness
	if !utils.IsImageTooDark(dark, 0.05) {
		t.Errorf("Expected image with 4%% brightness to be too dark for 5%% threshold")
	}

	bright := createTestImage(30) // ~12% brightness
	if utils.IsImageTooDark(bright, 0.05) {
		t.Errorf("Expected image with 12%% brightness to not be too dark for 5%% threshold")
	}
}
