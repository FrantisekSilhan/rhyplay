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

type Shape struct {
	RoundCorners float64 `json:"round_corners"`
	LineWidth    float64 `json:"line_width"`
	NoteShape    string  `json:"note_shape"`
	Ngon         struct {
		Sides int     `json:"sides"`
		Angle float64 `json:"angle"`
	} `json:"ngon"`
	Weirdo struct {
		RoundedCorners [4]bool `json:"rounded_corners"`
	} `json:"weirdo"`
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
		Modifiers struct {
			Ghost    bool `json:"ghost"`
			FadeOut  bool `json:"fade_out"`
			Pushback bool `json:"pushback"`
		} `json:"modifiers"`
		Parallax float64 `json:"parallax"`
		Note     struct {
			RGB  []RGB `json:"rgb"`
			Fill struct {
				Enabled bool   `json:"enabled"`
				Mode    string `json:"mode"` // "solid" or "custom"
				Solid   struct {
					Alpha uint8 `json:"alpha"`
				} `json:"solid"`
				Custom struct {
					RGBA []RGBA `json:"rgba"`
				} `json:"custom"`
			} `json:"fill"`
			Shape Shape `json:"shape"`
		} `json:"note"`
		Cursor struct {
			Size float64 `json:"size"`
			RGBA RGBA    `json:"rgba"`
			Fill struct {
				Enabled bool   `json:"enabled"`
				Mode    string `json:"mode"` // "solid" or "custom"
				Solid   struct {
					Alpha uint8 `json:"alpha"`
				} `json:"solid"`
				Custom struct {
					RGBA RGBA `json:"rgba"`
				} `json:"custom"`
			} `json:"fill"`
			Shape Shape `json:"shape"`
		} `json:"cursor"`
		Background struct {
			RGB     RGB `json:"rgb"`
			Corners struct {
				RGBA         RGBA    `json:"rgba"`
				RoundCorners float64 `json:"round_corners"`
				LineWidth    float64 `json:"line_width"`
				Length       float64 `json:"length"`
			} `json:"corners"`
		} `json:"background"`
	} `json:"visuals"`
}

var Current *Settings

func NewDefault() *Settings {
	s := &Settings{}
	s.Video.Width, s.Video.Height, s.Video.FPS = 1920, 1080, 60
	s.Gameplay.ApproachDistance, s.Gameplay.ApproachRate = 40.0, 40.0
	s.Visuals.Modifiers.Ghost = false
	s.Visuals.Modifiers.FadeOut = false
	s.Visuals.Modifiers.Pushback = false
	s.Visuals.Parallax = 5.0
	s.Visuals.Note.RGB = []RGB{{229, 229, 229}}
	s.Visuals.Note.Fill.Enabled = false
	s.Visuals.Note.Fill.Mode = "solid"
	s.Visuals.Note.Fill.Solid.Alpha = 64
	s.Visuals.Note.Fill.Custom.RGBA = []RGBA{{229, 229, 229, 64}}
	s.Visuals.Note.Shape.RoundCorners = 0.25
	s.Visuals.Note.Shape.NoteShape = "square"
	s.Visuals.Note.Shape.LineWidth = 20.0
	s.Visuals.Note.Shape.Ngon.Sides = 6
	s.Visuals.Note.Shape.Ngon.Angle = 0.0
	s.Visuals.Note.Shape.Weirdo.RoundedCorners = [4]bool{false, true, false, true}
	s.Visuals.Cursor.Size = 1.0
	s.Visuals.Cursor.RGBA = RGBA{255, 255, 255, 255}

	s.Visuals.Cursor.Fill.Enabled = true
	s.Visuals.Cursor.Fill.Mode = "solid"
	s.Visuals.Cursor.Fill.Solid.Alpha = 64
	s.Visuals.Cursor.Fill.Custom.RGBA = RGBA{255, 255, 255, 64}

	s.Visuals.Cursor.Shape.LineWidth = 8.0
	s.Visuals.Cursor.Shape.RoundCorners = 0.25
	s.Visuals.Cursor.Shape.NoteShape = "circle"
	s.Visuals.Cursor.Shape.Ngon.Sides = 6
	s.Visuals.Cursor.Shape.Ngon.Angle = 0.0
	s.Visuals.Cursor.Shape.Weirdo.RoundedCorners = [4]bool{false, true, false, true}

	s.Visuals.Background.RGB = RGB{12, 12, 12}
	s.Visuals.Background.Corners.RGBA = RGBA{127, 127, 127, 255}
	s.Visuals.Background.Corners.RoundCorners = 0
	s.Visuals.Background.Corners.LineWidth = 10.0
	s.Visuals.Background.Corners.Length = 0.5
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
