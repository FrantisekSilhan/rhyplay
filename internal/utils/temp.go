package utils

import "os"

type TempFileResult struct {
	Path    string
	Cleanup func()
}

// Example pattern: "audio-*.mp3" "video-*.mp4"
func SaveTempFile(data []byte, pattern string) (*TempFileResult, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, err
	}

	_, err = f.Write(data)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return nil, err
	}

	return &TempFileResult{
		Path: f.Name(),
		Cleanup: func() {
			os.Remove(f.Name())
		},
	}, nil
}
