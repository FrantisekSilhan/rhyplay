package render

import (
	"fmt"
	"os"
	"os/exec"
)

func getSampleRate(path string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=sample_rate", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 44100, err
	}
	var sr int
	fmt.Sscanf(string(out), "%d", &sr)
	return sr, nil
}

func (r *Renderer) prepareArgs(outputPath, audioPath string, progressPort int) ([]string, string, error) {
	args := []string{
		"-y", "-f", "rawvideo", "-vcodec", "rawvideo", "-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", r.Width, r.Height),
		"-r", fmt.Sprintf("%d", r.FPS),
		"-i", "-",
	}

	if progressPort > 0 {
		args = append([]string{"-progress", fmt.Sprintf("tcp://127.0.0.1:%d", progressPort)}, args...)
	}

	currentInputIdx := 1
	var audioMapLabel string
	var filterComplex string

	musicIdx := -1
	if audioPath != "" {
		args = append(args, "-i", audioPath)
		musicIdx = currentInputIdx

		sampleRate, err := getSampleRate(audioPath)
		if err != nil {
			sampleRate = 44100
		}

		filterComplex += fmt.Sprintf("[%d:a]asetrate=%d*%.6f,aresample=44100[bg];", musicIdx, sampleRate, r.Replay.ModState.SpeedMultiplier)
		audioMapLabel = "[bg]"
		currentInputIdx++
	}

	hitSoundPath := "sounds/hit.mp3"
	hitIdx := currentInputIdx
	args = append(args, "-i", hitSoundPath)
	currentInputIdx++

	var hitLabels []string
	for i, frame := range r.Replay.Frames {
		if frame.Hit {
			timestamp := (float64(frame.Progress) / 1000.0) / float64(r.Replay.ModState.SpeedMultiplier)
			label := fmt.Sprintf("h%d", i)
			filterComplex += fmt.Sprintf("[%d:a]adelay=%.0f|%.0f[%s];", hitIdx, timestamp*1000, timestamp*1000, label)
			hitLabels = append(hitLabels, label)
		}
	}

	if len(hitLabels) > 0 {
		var mixInputs string
		for _, lbl := range hitLabels {
			mixInputs += fmt.Sprintf("[%s]", lbl)
		}
		filterComplex += fmt.Sprintf("%samix=inputs=%d:normalize=0[allhits];", mixInputs, len(hitLabels))

		if audioMapLabel == "[bg]" {
			filterComplex += "[bg][allhits]amix=inputs=2:duration=first[combined]"
			audioMapLabel = "[combined]"
		} else {
			audioMapLabel = "[allhits]"
		}
	}

	// TODO: add hitsounds in more efficient way
	var filterPath string
	if filterComplex != "" {
		f, err := os.CreateTemp("", "ffmpeg-filter-*.txt")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create filter script: %w", err)
		}

		if _, err := f.WriteString(filterComplex); err != nil {
			return nil, "", fmt.Errorf("failed to write filter script: %w", err)
		}
		filterPath = f.Name()
		f.Close()

		args = append(args, "-filter_complex_script", f.Name())
	}

	args = append(args, "-map", "0:v")

	if audioMapLabel != "" {
		args = append(args, "-map", audioMapLabel)
	}

	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-preset", "ultrafast", outputPath)

	return args, filterPath, nil
}
