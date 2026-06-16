package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
)

type ReplayFrame struct {
	Counter uint32
	X       float32
	Y       float32
	Val     float32
	Hit     bool
}

type ReplayData struct {
	SpeedMultiplier float32
	Frames          []ReplayFrame
}

var hashMarkerRegex = regexp.MustCompile(`@[0-9a-fA-F]{64}`)

func ParseReplay(path string) (*ReplayData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	loc := hashMarkerRegex.FindIndex(data)
	if loc == nil {
		return nil, fmt.Errorf("marker not found in file")
	}

	markerIndex := loc[0]

	dataStart := markerIndex + 69
	if dataStart >= len(data) {
		return nil, fmt.Errorf("file too short after marker")
	}

	reader := bytes.NewReader(data[dataStart:])
	var frames []ReplayFrame

	for {
		var frame ReplayFrame
		err := binary.Read(reader, binary.LittleEndian, &frame)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing frame at index %d: %w", len(frames), err)
		}

		if frame.X > 10.0 || frame.X < -10.0 {
			break
		}

		frames = append(frames, frame)
	}

	return &ReplayData{
		SpeedMultiplier: GetSpeedMultiplier(data),
		Frames:          frames,
	}, nil
}

func GetSpeedMultiplier(data []byte) float32 {
	anchor := []byte("online_profile")
	index := bytes.Index(data, anchor)
	if index == -1 {
		return 1.0
	}

	floatOffset := index + 19

	if floatOffset+4 > len(data) {
		return 1.0
	}

	bits := binary.LittleEndian.Uint32(data[floatOffset : floatOffset+4])
	return math.Float32frombits(bits)
}
