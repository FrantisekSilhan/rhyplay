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
	Progress uint32
	X        float32
	Y        float32
	Health   float32
	Hit      bool
}

type ReplayData struct {
	ModState ModState
	Frames   []ReplayFrame
}

type ModState struct {
	SpeedMultiplier float32
	HardrockEnabled bool
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
		ModState: GetMods(data),
		Frames:   frames,
	}, nil
}

func GetMods(data []byte) ModState {
	state := ModState{
		SpeedMultiplier: 1.0,
		HardrockEnabled: false,
	}
	anchor := []byte("online_profile")
	index := bytes.Index(data, anchor)
	if index == -1 {
		return state
	}

	searchStart := index + len(anchor)
	bracketIndex := bytes.IndexByte(data[searchStart:], ']')
	if bracketIndex == -1 {
		return state
	}

	absBracketIndex := searchStart + bracketIndex

	modRange := data[index:absBracketIndex]
	if bytes.Contains(modRange, []byte("mod_hardrock")) {
		state.HardrockEnabled = true
	}

	speedOffset := absBracketIndex + 2

	if speedOffset+4 > len(data) {
		return state
	}

	bits := binary.LittleEndian.Uint32(data[speedOffset : speedOffset+4])
	state.SpeedMultiplier = math.Float32frombits(bits)

	return state
}
