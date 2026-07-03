package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	// "image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

func loadImg(path string) image.Image {
	imgData, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer imgData.Close()

	img, _, err := image.Decode(imgData)
	if err != nil {
		log.Fatal(err)
	}
	return img
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
			// Store avg in the output image
			result.SetGray(x, y, color.Gray{Y: avg})
		}
	}
	return result
}

func ASCIIToPNG(ascii, outputFile string) error {
	lines := strings.Split(strings.TrimRight(ascii, "\n"), "\n")

	fontBytes, err := os.ReadFile("JetBrainsMono-VariableFont_wght.ttf")
	if err != nil {
		return err
	}

	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	// Use a Drawer to measure text.
	measure := &font.Drawer{
		Face: face,
	}

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

	// Dark gray background
	draw.Draw(
		img,
		img.Bounds(),
		&image.Uniform{
			C: color.RGBA{40, 40, 40, 255},
		},
		image.Point{},
		draw.Src,
	)

	d := &font.Drawer{
		Dst: img,
		Src: &image.Uniform{
			C: color.RGBA{220, 220, 220, 255},
		},
		Face: face,
	}

	y := ascent

	for _, line := range lines {
		d.Dot = fixed.P(0, y)
		d.DrawString(line)
		y += lineHeight
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func main() {
	img := loadImg("riel2.jpeg")

	grayImg := image.NewGray(img.Bounds())
	draw.Draw(grayImg, img.Bounds(), img, image.Point{}, draw.Src)

	targetW := 120
	ratio := float64(grayImg.Bounds().Dy()) / float64(grayImg.Bounds().Dx())
	targetH := int(float64(targetW) * ratio * 0.5) // *0.5 because terminal chars are ~2x tall

	scaledGrayImg := downScale(grayImg, targetW, targetH)

	var brightness int
	imgStr := []string{}
	count := 0

	for y := scaledGrayImg.Bounds().Min.Y; y < scaledGrayImg.Bounds().Max.Y; y++ {
		for x := scaledGrayImg.Bounds().Min.X; x < scaledGrayImg.Bounds().Max.X; x++ {
			brightness = int(scaledGrayImg.GrayAt(x, y).Y)
			count++
			if scaledGrayImg.Stride == count {
				fmt.Print("\n")
				imgStr = append(imgStr, "\n")
				count = 0
			}
			if brightness <= 25 {
				fmt.Print(".")
				imgStr = append(imgStr, ".")
			} else if brightness <= 50{
				fmt.Print(",")
				imgStr = append(imgStr, ",")
			} else if brightness <= 75 {
				fmt.Print(":")
				imgStr = append(imgStr, ":")
			} else if brightness <= 100 {
				fmt.Print(";")
				imgStr = append(imgStr, ";")
			} else if brightness <= 125 {
				fmt.Print("+")
				imgStr = append(imgStr, "+")
			} else if brightness < 150 {
				fmt.Print("x")
				imgStr = append(imgStr, "x")
			} else if brightness <= 175 {
				fmt.Print("%")
				imgStr = append(imgStr, "%")
			} else if brightness <= 200 {
				fmt.Print("$")
				imgStr = append(imgStr, "$")
			} else if brightness <= 225 {
				fmt.Print("@")
				imgStr = append(imgStr, "@")
			} else {
				fmt.Print("#")
				imgStr = append(imgStr, "#")
			}
		}
	}

	// save the edited image
	// newImg, err := os.Create("Output.txt")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer newImg.Close()

	// // png.Encode(newImg, grayImg)

	// newImg.WriteString(strings.Join(imgStr, ""))

	ASCIIToPNG(strings.Join(imgStr, ""), "output.png")

}
