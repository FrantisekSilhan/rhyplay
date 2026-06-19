package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type RGB [3]uint8

func (c RGB) ToInt() (r, g, b int) {
	return int(c[0]), int(c[1]), int(c[2])
}

func (c RGB) ToIntAlpha255(alpha int) (r, g, b, a int) {
	return int(c[0]), int(c[1]), int(c[2]), alpha
}

func (c RGB) ToIntAlpha(alpha float64) (r, g, b, a int) {
	return int(c[0]), int(c[1]), int(c[2]), int(alpha * 255)
}

type RGBA [4]uint8

func (c RGBA) ToInt() (r, g, b, a int) {
	return int(c[0]), int(c[1]), int(c[2]), int(c[3])
}

type Settings struct {
	Video struct {
		Width  int `json:"width"`
		Height int `json:"height"`
		FPS    int `json:"fps"`
	} `json:"video"`

	Gameplay struct {
		ApproachDistance float64 `json:"approach_distance"`
		ApproachRate     float64 `json:"approach_rate"`
	} `json:"gameplay"`

	Visuals struct {
		ParallaxAmount float64 `json:"parallax_amount"`
		BackgroundRGB  RGB     `json:"background_rgb"`
		NoteRGB        RGB     `json:"note_rgb"`
		Cursor         struct {
			Size      float64 `json:"size"`
			InnerRGBA RGBA    `json:"inner_rgba"`
			OuterRGBA RGBA    `json:"outer_rgba"`
			OuterSize float64 `json:"outer_size"`
		} `json:"cursor"`
		Background struct {
			CornersRGB RGB `json:"corners_rgb"`
		} `json:"background"`
	} `json:"visuals"`
}

var Current *Settings

func NewDefault() *Settings {
	s := &Settings{}
	s.Video.Width, s.Video.Height, s.Video.FPS = 1920, 1080, 60
	s.Gameplay.ApproachDistance, s.Gameplay.ApproachRate = 20.0, 25.0
	s.Visuals.ParallaxAmount = 40.0
	s.Visuals.BackgroundRGB = RGB{12, 12, 12}
	s.Visuals.NoteRGB = RGB{229, 229, 229}
	s.Visuals.Cursor.Size = 0.15
	s.Visuals.Cursor.InnerRGBA = RGBA{255, 255, 255, 76}
	s.Visuals.Cursor.OuterRGBA = RGBA{255, 255, 255, 255}
	s.Visuals.Cursor.OuterSize = 8.0
	s.Visuals.Background.CornersRGB = RGB{127, 127, 127}
	return s
}

func Save(path string) error {
	data, err := json.MarshalIndent(Current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func Load(path string) (changed bool, err error) {
	Current = NewDefault()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err = Save(path)
		return true, err
	}

	diskData, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(diskData, Current); err != nil {
		return false, fmt.Errorf("corrupted: %w", err)
	}

	newData, _ := json.MarshalIndent(Current, "", "  ")

	if string(diskData) != string(newData) {
		err = Save(path)
		return true, err
	}

	return false, nil
}
