package converter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "embed"
	_ "image/jpeg"
)

//go:embed JetBrainsMono-VariableFont_wght.ttf
var fontData []byte

type ConvertResult struct {
	Ascii     string
	PngBytes  []byte
	PngBase64 string
}

func loadImg(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func downScale(img *image.Gray, newW, newH int) *image.Gray {
	result := image.NewGray(image.Rect(0, 0, newW, newH))

	origWidth := img.Bounds().Dx()
	origHeight := img.Bounds().Dy()

	scaleX := float64(origWidth) / float64(newW)
	scaleY := float64(origHeight) / float64(newH)

	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			startX := int(float64(x) * scaleX)
			endX := int(float64(x+1) * scaleX)

			startY := int(float64(y) * scaleY)
			endY := int(float64(y+1) * scaleY)

			var sum int
			var count int

			for yy := startY; yy < endY; yy++ {
				for xx := startX; xx < endX; xx++ {
					sum += int(img.GrayAt(xx, yy).Y)
					count++
				}
			}

			avg := uint8(sum / count)
			result.SetGray(x, y, color.Gray{Y: avg})
		}
	}
	return result
}

func asciiToPNG(ascii string) ([]byte, error) {
	lines := strings.Split(strings.TrimRight(ascii, "\n"), "\n")

	tt, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create font face: %w", err)
	}
	defer face.Close()

	measure := &font.Drawer{Face: face}

	lineHeight := face.Metrics().Height.Ceil()
	ascent := face.Metrics().Ascent.Ceil()
	charWidth := measure.MeasureString("M").Ceil()

	maxCols := 0
	for _, line := range lines {
		if len(line) > maxCols {
			maxCols = len(line)
		}
	}

	imgWidth := maxCols * charWidth
	imgHeight := len(lines) * lineHeight

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	draw.Draw(
		img,
		img.Bounds(),
		&image.Uniform{C: color.RGBA{40, 40, 40, 255}},
		image.Point{},
		draw.Src,
	)

	d := &font.Drawer{
		Dst: img,
		Src: &image.Uniform{C: color.RGBA{220, 220, 220, 255}},
		Face: face,
	}

	y := ascent
	for _, line := range lines {
		d.Dot = fixed.P(0, y)
		d.DrawString(line)
		y += lineHeight
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func ConvertImage(r io.Reader, targetW int) (*ConvertResult, error) {
	img, err := loadImg(r)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %w", err)
	}

	grayImg := image.NewGray(img.Bounds())
	draw.Draw(grayImg, img.Bounds(), img, image.Point{}, draw.Src)

	ratio := float64(grayImg.Bounds().Dy()) / float64(grayImg.Bounds().Dx())
	targetH := int(float64(targetW) * ratio * 0.5)

	scaledGrayImg := downScale(grayImg, targetW, targetH)

	var brightness int
	imgStr := []string{}
	count := 0

	for y := scaledGrayImg.Bounds().Min.Y; y < scaledGrayImg.Bounds().Max.Y; y++ {
		for x := scaledGrayImg.Bounds().Min.X; x < scaledGrayImg.Bounds().Max.X; x++ {
			brightness = int(scaledGrayImg.GrayAt(x, y).Y)
			count++
			if scaledGrayImg.Stride == count {
				imgStr = append(imgStr, "\n")
				count = 0
			}
			if brightness <= 25 {
				imgStr = append(imgStr, ".")
			} else if brightness <= 50 {
				imgStr = append(imgStr, ",")
			} else if brightness <= 75 {
				imgStr = append(imgStr, ":")
			} else if brightness <= 100 {
				imgStr = append(imgStr, ";")
			} else if brightness <= 125 {
				imgStr = append(imgStr, "+")
			} else if brightness < 150 {
				imgStr = append(imgStr, "x")
			} else if brightness <= 175 {
				imgStr = append(imgStr, "%")
			} else if brightness <= 200 {
				imgStr = append(imgStr, "$")
			} else if brightness <= 225 {
				imgStr = append(imgStr, "@")
			} else {
				imgStr = append(imgStr, "#")
			}
		}
	}

	ascii := strings.Join(imgStr, "")
	pngBytes, err := asciiToPNG(ascii)
	if err != nil {
		return nil, err
	}

	return &ConvertResult{
		Ascii:     ascii,
		PngBytes:  pngBytes,
		PngBase64: base64.StdEncoding.EncodeToString(pngBytes),
	}, nil
}
