package parser

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	KVersionNegateY        = 20260118
	KVersionExtendedFields = 20260125
	KVersionFailTime       = 20260222
	KVersionInt32Time      = 20260510
	KVersionBeatmapHash    = 20260517
)

type ReplayFrame struct {
	Progress int32
	X        float32
	Y        float32
	Health   float32
	Hit      bool
}

type ScoreData struct {
	Timestamp   int64
	DatePlayed  time.Time
	PlayerName  string
	LegacyMapID string
	MapID       int32
	StartFrom   int32
	Mode        string
	Passed      bool
	Mods        string
	Spin        bool
	Speed       float32
	TotalScore  int64
	Accuracy    float32
	Hits        int32
	Misses      int32
	Points      int32
	FailTime    int32
	Failed      bool
	BeatmapHash string
}

type ReplayData struct {
	Version   int32
	ScoreData ScoreData
	Frames    []ReplayFrame
}

type replayReader struct {
	reader io.Reader
	err    error
}

func (r *replayReader) read(data interface{}) {
	if r.err != nil {
		return
	}
	r.err = binary.Read(r.reader, binary.LittleEndian, data)
}

func (r *replayReader) readString() string {
	if r.err != nil {
		return ""
	}

	var length uint32
	var shift uint
	for {
		var b uint8
		r.err = binary.Read(r.reader, binary.LittleEndian, &b)
		if r.err != nil {
			return ""
		}

		length |= uint32(b&0x7F) << shift
		if (b & 0x80) == 0 {
			break
		}
		shift += 7
		if shift >= 35 {
			r.err = fmt.Errorf("string length varint overflow")
			return ""
		}
	}

	if length == 0 {
		return ""
	}

	buf := make([]byte, length)
	_, r.err = io.ReadFull(r.reader, buf)
	return string(buf)
}

func ParseReplay(path string) (*ReplayData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer file.Close()

	rr := &replayReader{reader: file}
	res := &ReplayData{}

	rr.read(&res.Version)
	rr.read(&res.ScoreData.Timestamp)

	unixSecs := (res.ScoreData.Timestamp - 621355968000000000) / 10000000
	res.ScoreData.DatePlayed = time.Unix(unixSecs, 0)

	res.ScoreData.PlayerName = rr.readString()
	res.ScoreData.LegacyMapID = rr.readString()
	rr.read(&res.ScoreData.MapID)
	rr.read(&res.ScoreData.StartFrom)
	res.ScoreData.Mode = rr.readString()

	if res.Version >= KVersionExtendedFields {
		rr.read(&res.ScoreData.Passed)
		res.ScoreData.Mods = rr.readString()
		rr.read(&res.ScoreData.Spin)
		rr.read(&res.ScoreData.Speed)
		rr.read(&res.ScoreData.TotalScore)
	} else {
		res.ScoreData.Passed = true
		res.ScoreData.Mods = "[]"
		res.ScoreData.Speed = 1.0
	}

	rr.read(&res.ScoreData.Accuracy)
	rr.read(&res.ScoreData.Hits)
	rr.read(&res.ScoreData.Misses)
	rr.read(&res.ScoreData.Points)

	res.ScoreData.FailTime = -1
	if res.Version >= KVersionFailTime {
		rr.read(&res.ScoreData.FailTime)
		res.ScoreData.Failed = res.ScoreData.FailTime >= 0
	}

	if res.Version >= KVersionBeatmapHash {
		res.ScoreData.BeatmapHash = rr.readString()
	}

	var frameCount int32
	rr.read(&frameCount)

	if rr.err != nil {
		return nil, fmt.Errorf("failed to read frame count: %w", rr.err)
	}

	res.Frames = make([]ReplayFrame, frameCount)
	for i := 0; i < int(frameCount); i++ {
		var frame ReplayFrame
		if res.Version >= KVersionInt32Time {
			rr.read(&frame.Progress)
		} else {
			var floatTime float32
			rr.read(&floatTime)
			frame.Progress = int32(floatTime)
		}

		rr.read(&frame.X)
		rr.read(&frame.Y)
		rr.read(&frame.Health)

		var important uint8
		rr.read(&important)
		frame.Hit = important != 0

		if res.Version < KVersionNegateY {
			frame.Y = -frame.Y
		}

		if rr.err != nil {
			return nil, fmt.Errorf("error parsing frame at index %d: %w", i, rr.err)
		}
		res.Frames[i] = frame
	}

	return res, nil
}
