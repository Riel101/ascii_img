package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"ascii-converter/internal/converter"
)

type convertResponse struct {
	Ascii      string `json:"ascii"`
	PngDataURL string `json:"pngDataUrl"`
	Error      string `json:"error,omitempty"`
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

	result, err := converter.ConvertImage(file, 120)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(convertResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertResponse{
		Ascii:      result.Ascii,
		PngDataURL: "data:image/png;base64," + result.PngBase64,
	})
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
