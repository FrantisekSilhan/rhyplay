package parser

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Note struct {
	Time int     `json:"Time"`
	X    float64 `json:"X"`
	Y    float64 `json:"Y"`
}

type MapData struct {
	SongName   string  `json:"SongName"`
	Title      string  `json:"Title"`
	Difficulty int     `json:"Difficulty"`
	StarRating float64 `json:"StarRating"`
	Notes      []Note  `json:"Notes"`
}

func ParseMap(path string) (*MapData, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open map: %w", err)
	}
	defer r.Close()

	var mapData MapData

	found := false
	for _, f := range r.File {
		if f.Name == "map" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(content, &mapData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse map JSON: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("file 'map' not found inside rhm file")
	}

	return &mapData, nil
}

func ExtractAudio(path string, outputPath string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "audio" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			dst, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, rc)
			return err
		}
	}
	return fmt.Errorf("file 'audio' not found inside rhm file")
}
