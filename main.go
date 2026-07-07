package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "image/jpeg"
	_ "image/png"
)

func loadImg(path string) (image.Image, error) {
	imgData, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer imgData.Close()

	img, _, err := image.Decode(imgData)
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

func ASCIIToPNG(ascii, outputFile, fontPath string) error {
	lines := strings.Split(strings.TrimRight(ascii, "\n"), "\n")

	fontBytes, err := os.ReadFile(fontPath)
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

func convertImage(inputPath string, targetW int) (string, string, error) {
	img, err := loadImg(inputPath)
	if err != nil {
		return "", "", err
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
	outputFile := fmt.Sprintf("output_%d.png", time.Now().UnixNano())
	if err := ASCIIToPNG(ascii, outputFile, "JetBrainsMono-VariableFont_wght.ttf"); err != nil {
		return "", "", err
	}

	return ascii, outputFile, nil
}

type convertResponse struct {
	Ascii       string `json:"ascii"`
	DownloadURL string `json:"downloadUrl"`
	Error       string `json:"error,omitempty"`
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(32 << 20)

	file, _, err := r.FormFile("image")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(convertResponse{Error: "No image file provided"})
		return
	}
	defer file.Close()

	tmpFile, err := os.CreateTemp("", "upload-*.png")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(convertResponse{Error: "Failed to process upload"})
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, file); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(convertResponse{Error: "Failed to save upload"})
		return
	}
	tmpFile.Close()

	ascii, outputFile, err := convertImage(tmpFile.Name(), 120)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(convertResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertResponse{
		Ascii:       ascii,
		DownloadURL: "/output/" + outputFile,
	})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/output/")
	safe := filepath.Base(filename)
	w.Header().Set("Content-Disposition", "attachment; filename=output.png")
	http.ServeFile(w, r, safe)
	os.Remove(safe)
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch {
		case r.URL.Path == "/convert":
			handleConvert(w, r)
		case strings.HasPrefix(r.URL.Path, "/output/"):
			handleDownload(w, r)
		case r.URL.Path == "/":
			http.ServeFile(w, r, "index.html")
		default:
			http.FileServer(http.Dir(".")).ServeHTTP(w, r)
		}
	})

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("• ASCII Art Converter •\n")
	fmt.Printf("  Server running at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

