package render

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"os/exec"
	"rhyplay/internal/parser"

	"github.com/fogleman/gg"
)

type Renderer struct {
	Beatmap         *parser.MapData
	Replay          []parser.ReplayFrame
	SpeedMultiplier float32

	Width, Height int
	FPS           int

	PlayAreaSize     float64
	OffsetX, OffsetY float64

	ParallaxAmount float64
}

func NewRenderer(b *parser.MapData, r *parser.ReplayData) *Renderer {
	// TODO: Make everything configurable
	w, h := 1920, 1080
	padding := 0.2
	size := float64(h) * (1.0 - (padding * 2))

	return &Renderer{
		Beatmap:         b,
		Replay:          r.Frames,
		SpeedMultiplier: r.SpeedMultiplier,
		Width:           w,
		Height:          h,
		FPS:             60,
		PlayAreaSize:    size,
		OffsetX:         (float64(w) - size) / 2.0,
		OffsetY:         (float64(h) - size) / 2.0,
		ParallaxAmount:  40.0,
	}
}

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

func (r *Renderer) Render(outputPath string, audioPath string) error {
	msPerFrame := 1000.0 / float64(r.FPS)
	replayEndTime := float64(r.Replay[len(r.Replay)-1].Counter)
	videoDuration := replayEndTime / float64(r.SpeedMultiplier)

	hitSoundPath := "sounds/hit.mp3"

	args := []string{
		"-y", "-f", "rawvideo", "-vcodec", "rawvideo", "-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", r.Width, r.Height),
		"-r", fmt.Sprintf("%d", r.FPS),
		"-i", "-",
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

		filterComplex += fmt.Sprintf("[%d:a]asetrate=%d*%.6f,aresample=44100[bg];", musicIdx, sampleRate, r.SpeedMultiplier)
		audioMapLabel = "[bg]"
		currentInputIdx++
	}

	hitIdx := currentInputIdx
	args = append(args, "-i", hitSoundPath)
	currentInputIdx++

	var hitLabels []string
	for i, frame := range r.Replay {
		if frame.Hit {
			timestamp := (float64(frame.Counter) / 1000.0) / float64(r.SpeedMultiplier)
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
	if filterComplex != "" {
		f, err := os.CreateTemp("", "ffmpeg-filter-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create filter script: %w", err)
		}
		defer os.Remove(f.Name())

		if _, err := f.WriteString(filterComplex); err != nil {
			return fmt.Errorf("failed to write filter script: %w", err)
		}
		f.Close()

		args = append(args, "-filter_complex_script", f.Name())
	}

	args = append(args, "-map", "0:v")

	if audioMapLabel != "" {
		args = append(args, "-map", audioMapLabel)
	}

	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-preset", "ultrafast", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, _ := cmd.StdinPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg failed to start: %w", err)
	}

	dc := gg.NewContext(r.Width, r.Height)

	replayIdx := 0
	for currentTime := 0.0; currentTime <= videoDuration; currentTime += msPerFrame {
		engineTime := currentTime * float64(r.SpeedMultiplier)
		for replayIdx < len(r.Replay)-2 && float64(r.Replay[replayIdx+1].Counter) < engineTime {
			replayIdx++
		}
		f1, f2 := r.Replay[replayIdx], r.Replay[replayIdx+1]
		alpha := calculateAlpha(float64(f1.Counter), float64(f2.Counter), engineTime)
		curX := lerp32(f1.X, f2.X, alpha)
		curY := lerp32(f1.Y, f2.Y, alpha)

		normX := float64(curX) / 1.37
		normY := float64(curY) / 1.37

		shiftX := normX * -r.ParallaxAmount
		shiftY := normY * -r.ParallaxAmount

		r.DrawBackground(dc)
		r.DrawCorners(dc, r.OffsetX+shiftX, r.OffsetY+shiftY, r.PlayAreaSize, 100, 10)

		for _, note := range r.Beatmap.Notes {
			r.DrawNote(dc, note, engineTime, shiftX, shiftY)
		}

		r.DrawCursor(dc, curX, curY, shiftX, shiftY)

		img := dc.Image().(*image.RGBA)
		stdin.Write(img.Pix)
	}

	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg error: %v\n-- LOG --\n%s", err, stderr.String())
	}

	return nil
}

func (r *Renderer) DrawBackground(dc *gg.Context) {
	dc.SetRGB(0.05, 0.05, 0.05)
	dc.Clear()
}

func (r *Renderer) DrawNote(dc *gg.Context, note parser.Note, currentTime, shiftX, shiftY float64) {
	ad := 20.0
	ar := 25.0
	at := ad / ar

	fadeIn := 0.2
	noteSizeMultiplier := 0.3
	baseLineWidth := 20.0

	depth := (float64(note.Time) - currentTime) / (1000 * at) * ad / float64(r.SpeedMultiplier)

	if depth > ad || depth < 0 {
		return
	}

	viewDist := 2.0
	perspective := viewDist / (depth + viewDist)

	currentSize := r.PlayAreaSize * noteSizeMultiplier * perspective
	currentLineWidth := baseLineWidth * perspective

	visualTotalSize := currentSize + (currentLineWidth * 2)
	usableArea := r.PlayAreaSize - visualTotalSize

	relX := (note.X - 1.0) * 0.5
	relY := (1.0 - note.Y) * 0.5

	centerX, centerY := (float64(r.Width)/2.0)+shiftX, (float64(r.Height)/2.0)+shiftY

	drawX := centerX + (relX * usableArea * perspective)
	drawY := centerY + (relY * usableArea * perspective)

	progress := 1.0 - (depth / ad)
	alpha := 1.0
	if progress < fadeIn {
		alpha = progress / fadeIn
	}

	if alpha > 0 {
		radius := currentSize * 0.2
		dc.SetRGBA(1, 1, 1, alpha)
		dc.SetLineWidth(currentLineWidth)

		dc.DrawRoundedRectangle(
			drawX-currentSize/2,
			drawY-currentSize/2,
			currentSize,
			currentSize,
			radius,
		)
		dc.Stroke()
	}
}

func (r *Renderer) DrawCorners(dc *gg.Context, x, y, size, length, lineWidth float64) {
	dc.SetRGB(0.5, 0.5, 0.5)
	dc.SetLineWidth(lineWidth)

	dc.MoveTo(x, y+length)
	dc.LineTo(x, y)
	dc.LineTo(x+length, y)
	dc.Stroke()
	dc.MoveTo(x+size-length, y)
	dc.LineTo(x+size, y)
	dc.LineTo(x+size, y+length)
	dc.Stroke()
	dc.MoveTo(x, y+size-length)
	dc.LineTo(x, y+size)
	dc.LineTo(x+length, y+size)
	dc.Stroke()
	dc.MoveTo(x+size-length, y+size)
	dc.LineTo(x+size, y+size)
	dc.LineTo(x+size, y+size-length)
	dc.Stroke()
}

func (r *Renderer) DrawCursor(dc *gg.Context, x, y float32, shiftX, shiftY float64) {
	hitboxSize := r.PlayAreaSize * 0.06

	visualSize := r.PlayAreaSize * 0.06

	usableArea := r.PlayAreaSize - hitboxSize

	normX := float64(x) / 2.74
	normY := float64(y) / 2.74

	centerX, centerY := (float64(r.Width)/2.0)+shiftX, (float64(r.Height)/2.0)+shiftY

	screenX := centerX + (normX * usableArea)
	screenY := centerY + (normY * usableArea)

	dc.SetRGBA(1, 1, 1, 0.3)
	dc.DrawCircle(screenX, screenY, visualSize/2)
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 1.0)
	dc.SetLineWidth(8)
	dc.DrawCircle(screenX, screenY, visualSize/2)
	dc.Stroke()
}

func lerp32(a, b float32, t float64) float32 { return a + float32(t)*(b-a) }
func lerp64(a, b, t float64) float64         { return a + t*(b-a) }
func calculateAlpha(t1, t2, current float64) float64 {
	if current <= t1 {
		return 0
	}
	if current >= t2 {
		return 1
	}
	return (current - t1) / (t2 - t1)
}
