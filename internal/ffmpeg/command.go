package ffmpeg

import (
	"os"
	"path/filepath"
)

func GetExecutablePath() string {
	if exe, err := os.Executable(); err == nil {
		localPath := filepath.Join(filepath.Dir(exe), "ffmpeg.exe")
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}
	return "ffmpeg"
}
