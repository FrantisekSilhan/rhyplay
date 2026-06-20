package render

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"rhyplay/internal/config"
	"rhyplay/internal/game"
	"rhyplay/internal/parser"

	"github.com/fogleman/gg"
)

type Renderer struct {
	s       *config.Settings
	Beatmap *parser.MapData
	Replay  *parser.ReplayData

	Width, Height int
	FPS           int

	PlayAreaSize     float64
	OffsetX, OffsetY float64
}

func NewRenderer(b *parser.MapData, r *parser.ReplayData) *Renderer {
	s := config.Current
	w, h := s.Video.Width, s.Video.Height
	size := float64(h) * (1.0 - (game.Padding * 2))

	return &Renderer{
		s:            s,
		Beatmap:      b,
		Replay:       r,
		Width:        w,
		Height:       h,
		FPS:          s.Video.FPS,
		PlayAreaSize: size,
		OffsetX:      (float64(w) - size) / 2.0,
		OffsetY:      (float64(h) - size) / 2.0,
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
	if len(r.Replay.Frames) < 2 {
		return fmt.Errorf("replay contains too few frames")
	}
	replayEndTime := float64(r.Replay.Frames[len(r.Replay.Frames)-1].Progress)
	videoDuration := replayEndTime / float64(r.Replay.SpeedMultiplier)

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

		filterComplex += fmt.Sprintf("[%d:a]asetrate=%d*%.6f,aresample=44100[bg];", musicIdx, sampleRate, r.Replay.SpeedMultiplier)
		audioMapLabel = "[bg]"
		currentInputIdx++
	}

	hitIdx := currentInputIdx
	args = append(args, "-i", hitSoundPath)
	currentInputIdx++

	var hitLabels []string
	for i, frame := range r.Replay.Frames {
		if frame.Hit {
			timestamp := (float64(frame.Progress) / 1000.0) / float64(r.Replay.SpeedMultiplier)
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
		engineTime := currentTime * float64(r.Replay.SpeedMultiplier)
		for replayIdx < len(r.Replay.Frames)-2 && float64(r.Replay.Frames[replayIdx+1].Progress) < engineTime {
			replayIdx++
		}
		f1, f2 := r.Replay.Frames[replayIdx], r.Replay.Frames[replayIdx+1]
		alpha := calculateAlpha(float64(f1.Progress), float64(f2.Progress), engineTime)
		curX := lerp32(f1.X, f2.X, alpha)
		curY := -lerp32(f1.Y, f2.Y, alpha)

		shiftX := -float64(curX) * r.s.Visuals.Parallax
		shiftY := -float64(curY) * r.s.Visuals.Parallax

		r.DrawBackground(dc)
		r.DrawCorners(dc, r.OffsetX+shiftX, r.OffsetY+shiftY, r.PlayAreaSize, 100, 10)

		for i := len(r.Beatmap.Notes) - 1; i >= 0; i-- {
			note := r.Beatmap.Notes[i]
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
	c := r.s.Visuals.Background.RGB
	dc.SetRGB255(c.ToInt())
	dc.Clear()
}

func (r *Renderer) DrawNote(dc *gg.Context, note parser.Note, currentTime, shiftX, shiftY float64) {
	ad := r.s.Gameplay.ApproachDistance
	ar := r.s.Gameplay.ApproachRate
	at := ad / ar

	depth := (float64(note.Time) - currentTime) / (1000 * at) * ad / float64(r.Replay.SpeedMultiplier)

	if depth > ad || depth < 0 {
		return
	}

	perspective := game.CalcPerspective(depth)
	currentSize := r.PlayAreaSize * (game.NoteSize / game.GridSize) * perspective

	relX, relY := game.GameToScreen(note.X, note.Y, r.PlayAreaSize, perspective)

	centerX, centerY := (float64(r.Width)/2.0)+shiftX, (float64(r.Height)/2.0)+shiftY
	drawX, drawY := centerX+relX, centerY+relY

	fadeIn := game.FadeIn / 100.0
	progress := 1.0 - (depth / ad)
	alpha := 1.0
	if progress < fadeIn {
		alpha = progress / fadeIn
	}

	if r.s.Visuals.Modifiers.Ghost {
		startFade := 0.25
		endFade := 0.9
		if progress > startFade {
			ratio := (progress - startFade) / (endFade - startFade)

			if ratio > 1.0 {
				ratio = 1.0
			}

			alpha -= ratio
		}
	} else if r.s.Visuals.Modifiers.FadeOut {
		fadeOut := game.FadeOut / 100.0
		alpha -= 1 - math.Min(1, (1-progress)/fadeOut)
		if alpha < game.MinFadeOut {
			alpha = game.MinFadeOut
		}
	}

	if !r.s.Visuals.Modifiers.Pushback && float64(note.Time)-currentTime <= 0 {
		alpha = 0
	}

	if alpha > 0 {
		dc.SetRGBA255(r.s.Visuals.NoteRGB[note.NoteIdx%len(r.s.Visuals.NoteRGB)].ToIntAlpha(alpha))
		dc.SetLineWidth(20.0 * perspective)
		dc.DrawRoundedRectangle(drawX-currentSize/2, drawY-currentSize/2, currentSize, currentSize, currentSize*0.2)
		dc.Stroke()
	}
}

func (r *Renderer) DrawCorners(dc *gg.Context, x, y, size, length, lineWidth float64) {
	c := r.s.Visuals.Background.CornersRGB
	dc.SetRGB255(c.ToInt())
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
	visualSize := r.PlayAreaSize * ((game.CursorSize * config.Current.Visuals.Cursor.Size) / game.GridSize)

	relX, relY := game.CursorToScreen(float64(x), float64(y), r.PlayAreaSize)

	centerX, centerY := (float64(r.Width)/2.0)+shiftX, (float64(r.Height)/2.0)+shiftY
	screenX, screenY := centerX+relX, centerY+relY

	dc.SetRGBA255(r.s.Visuals.Cursor.InnerRGBA.ToInt())
	dc.DrawCircle(screenX, screenY, visualSize/2)
	dc.Fill()

	dc.SetRGBA255(r.s.Visuals.Cursor.OuterRGBA.ToInt())
	dc.SetLineWidth(r.s.Visuals.Cursor.OuterSize)
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
