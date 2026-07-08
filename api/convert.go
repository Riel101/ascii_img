package main

import (
	"encoding/json"
	"net/http"

	"ascii-converter/internal/converter"
)

type convertResponse struct {
	Ascii      string `json:"ascii"`
	PngDataURL string `json:"pngDataUrl"`
	Error      string `json:"error,omitempty"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
