package parser

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type Note struct {
	Time    int     `json:"Time"`
	X       float64 `json:"X"`
	Y       float64 `json:"Y"`
	NoteIdx int     `json:"-"`
}

func toGameSpace(x, y float64) (float64, float64) {
	return x - 1, y - 1
}
func toNoteSpace(x, y float64) Note {
	gx, gy := toGameSpace(x, y)
	return Note{X: gx, Y: gy}
}

type MapData struct {
	SongName   string  `json:"SongName"`
	Title      string  `json:"Title"`
	Difficulty int     `json:"Difficulty"`
	StarRating float64 `json:"StarRating"`
	Notes      []Note  `json:"Notes"`
}

func ParseMap(path string) (*MapData, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	p := &FileParser{r: f, endian: binary.LittleEndian}

	header := p.ReadBytes(4)
	if header[0] == 0x50 && header[1] == 0x4b && header[2] == 0x03 && header[3] == 0x04 {
		return ParseMapZip(path)
	}

	if string(header) != "SS+m" {
		return nil, nil, fmt.Errorf("invalid map file: missing 'SS+m' header")
	}

	version := p.ReadUInt16()

	switch version {
	case 1:
		return ParseSSPMV1(p)
	case 2:
		return ParseSSPMV2(p)
	default:
		return nil, nil, fmt.Errorf("unsupported map version: %d", version)
	}
}

func ParseMapZip(path string) (*MapData, []byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open map: %w", err)
	}
	defer r.Close()

	var mapData MapData

	found := false
	for _, f := range r.File {
		if f.Name == "map" {
			rc, err := f.Open()
			if err != nil {
				return nil, nil, err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, nil, err
			}

			err = json.Unmarshal(content, &mapData)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse map JSON: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return nil, nil, fmt.Errorf("file 'map' not found inside rhm file")
	}

	audioBytes, err := ExtractAudioBytes(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract audio: %w", err)
	}

	for i := range mapData.Notes {
		note := mapData.Notes[i]
		mapData.Notes[i] = toNoteSpace(note.X, note.Y)
		mapData.Notes[i].Time = note.Time
		mapData.Notes[i].NoteIdx = i
	}

	return &mapData, audioBytes, nil
}

func ParseSSPMV1(p *FileParser) (*MapData, []byte, error) {
	p.Skip(2)
	p.ReadLine() // id

	mapName := strings.Split(p.ReadLine(), " - ")

	var artist string
	var song string

	if len(mapName) == 1 {
		song = strings.TrimSpace(mapName[0])
	} else {
		artist = strings.TrimSpace(mapName[0])
		song = strings.TrimSpace(mapName[1])
	}

	p.ReadLine() // mappers

	p.Skip(4) // uint32 mapLength
	noteCount := p.ReadUInt32()

	difficulty := p.ReadUInt8()

	hasCover := p.ReadUInt8() == 2
	if hasCover {
		coverByteLength := int(p.ReadUInt64())
		p.Skip(coverByteLength)
	}

	hasAudio := p.ReadBool()
	var audioBuffer []byte
	if hasAudio {
		audioByteLength := int(p.ReadUInt64())
		audioBuffer = p.ReadBytes(audioByteLength)
	}

	var notes []Note
	for i := uint32(0); i < noteCount; i++ {
		time := int(p.ReadUInt32())

		isQuantum := p.ReadBool()
		var x, y float64
		if isQuantum {
			x = float64(p.ReadFloat32())
			y = float64(p.ReadFloat32())
		} else {
			x = float64(p.ReadUInt8())
			y = float64(p.ReadUInt8())
		}

		note := toNoteSpace(x, y)
		note.Time = time
		note.NoteIdx = int(i)
		notes = append(notes, note)
	}

	return &MapData{
		SongName:   song,
		Title:      fmt.Sprintf("%s - %s", artist, song),
		Difficulty: int(difficulty),
		Notes:      notes,
	}, audioBuffer, nil
}
func ParseSSPMV2(p *FileParser) (*MapData, []byte, error) {
	p.Skip(4)  // reserved
	p.Skip(20) // hash

	p.Skip(4) // mapLength
	noteCount := p.ReadUInt32()

	p.Skip(4) // marker count

	difficulty := p.ReadBytes(1)[0]

	p.Skip(2) // map rating

	hasCover := p.ReadBool()
	hasAudio := p.ReadBool()

	p.Skip(1) // 1mod

	customDataOffset := p.ReadUInt64()
	p.Skip(8) // customDataLength

	audioByteOffset := p.ReadUInt64()
	audioByteLength := p.ReadUInt64()

	p.Skip(16) // cover meta

	p.Skip(16) // marker meta

	markerByteOffset := p.ReadUInt64()

	p.Skip(8) // marker byte length

	mapIdLength := p.ReadUInt16()
	//id := string(p.ReadBytes(int(mapIdLength)))
	p.Skip(int(mapIdLength)) // unused

	mapNameLength := p.ReadUInt16()
	mapName := strings.Split(string(p.ReadBytes(int(mapNameLength))), " - ")

	var artist string
	var song string

	if len(mapName) == 1 {
		song = strings.TrimSpace(mapName[0])
	} else {
		artist = strings.TrimSpace(mapName[0])
		song = strings.TrimSpace(mapName[1])
	}

	songNameLength := p.ReadUInt16()
	p.Skip(int(songNameLength)) // song name

	mapperCount := p.ReadUInt16()
	for i := uint16(0); i < mapperCount; i++ {
		lineLength := p.ReadUInt16()
		p.Skip(int(lineLength)) // mapper name
	}

	var audioBuffer []byte
	//var coverBuffer []byte
	//var difficultyName string

	p.Seek(customDataOffset)
	p.Skip(2)

	if string(p.ReadBytes(int(p.ReadUInt16()))) == "difficulty_name" {
		var length int

		switch p.ReadBytes(1)[0] {
		case 9:
			{
				length = int(p.ReadUInt16())
				break
			}
		case 11:
			{
				length = int(p.ReadUInt32())
				break
			}
		}

		//difficultyName = string(p.ReadBytes(length))
		p.Skip(length) // unused
	}

	if hasAudio {
		p.Seek(audioByteOffset)
		audioBuffer = p.ReadBytes(int(audioByteLength))
	}
	if hasCover {
		// unused
		//p.Seek(int64(coverByteOffset))
		//coverBuffer = p.ReadBytes(int(coverByteLength))
	}

	p.Seek(markerByteOffset)
	var notes []Note

	for i := uint32(0); i < noteCount; i++ {
		time := int(p.ReadUInt32())

		p.Skip(1) // marker type, always note

		isQuantum := p.ReadBool()
		var x, y float64
		if isQuantum {
			x = float64(p.ReadFloat32())
			y = float64(p.ReadFloat32())
		} else {
			x = float64(p.ReadBytes(1)[0])
			y = float64(p.ReadBytes(1)[0])
		}

		note := toNoteSpace(x, y)
		note.Time = time
		note.NoteIdx = int(i)
		notes = append(notes, note)
	}

	return &MapData{
		SongName:   song,
		Title:      fmt.Sprintf("%s - %s", artist, song),
		Difficulty: int(difficulty),
		Notes:      notes,
	}, audioBuffer, nil
}

func ExtractAudioBytes(path string) ([]byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "audio" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file 'audio' not found inside rhm file")
}
