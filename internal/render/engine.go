package render

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"rhyplay/internal/config"
	"rhyplay/internal/ffmpeg"
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
	size := (float64(h) / game.BaseHeight) * game.BasePlayAreaSize

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

		filterComplex += fmt.Sprintf("[%d:a]asetrate=%d*%.6f,aresample=44100[bg];", musicIdx, sampleRate, r.Replay.SpeedMultiplier)
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

func (r *Renderer) writeFrames(stdin io.WriteCloser, videoDuration float64) {
	defer stdin.Close()
	msPerFrame := 1000.0 / float64(r.FPS)
	dc := gg.NewContext(r.Width, r.Height)
	replayIdx := 0

	for currentTime := 0.0; currentTime <= videoDuration; currentTime += msPerFrame {
		engineTime := currentTime * float64(r.Replay.SpeedMultiplier)
		for replayIdx < len(r.Replay.Frames)-2 && float64(r.Replay.Frames[replayIdx+1].Progress) < engineTime {
			replayIdx++
		}

		f1, f2 := r.Replay.Frames[replayIdx], r.Replay.Frames[replayIdx+1]
		alpha := calculateAlpha(float64(f1.Progress), float64(f2.Progress), engineTime)
		curX, curY := lerp32(f1.X, f2.X, alpha), -lerp32(f1.Y, f2.Y, alpha)
		shiftX, shiftY := -curX*r.s.Visuals.Parallax, -curY*r.s.Visuals.Parallax

		r.DrawBackground(dc)
		r.DrawCorners(dc, r.OffsetX+shiftX, r.OffsetY+shiftY, r.PlayAreaSize)

		for i := len(r.Beatmap.Notes) - 1; i >= 0; i-- {
			r.SetupNote(dc, r.Beatmap.Notes[i], engineTime, shiftX, shiftY)
		}

		r.DrawCursor(dc, curX, curY, shiftX, shiftY)
		stdin.Write(dc.Image().(*image.RGBA).Pix)
	}
}

func (r *Renderer) Render(outputPath string, audioPath string) error {
	if len(r.Replay.Frames) < 2 {
		return fmt.Errorf("replay contains too few frames")
	}

	replayEndTime := float64(r.Replay.Frames[len(r.Replay.Frames)-1].Progress)
	videoDuration := replayEndTime / float64(r.Replay.SpeedMultiplier)

	port, progressChan, err := ffmpeg.StartProgressServer()
	if err != nil {
		return fmt.Errorf("failed to start progress server: %w", err)
	}

	args, filterFile, err := r.prepareArgs(outputPath, audioPath, port)
	if err != nil {
		return fmt.Errorf("failed to prepare ffmpeg arguments: %w", err)
	}

	if filterFile != "" {
		defer os.Remove(filterFile)
	}

	cmd := exec.Command(ffmpeg.GetExecutablePath(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w, stderr: %s", err, stderr.String())
	}

	progressDone := make(chan bool)
	go func() {
		for p := range progressChan {
			if p.Done {
				fmt.Printf("\r       > Progress: 100.00%%          ")
				break
			}
			percent := (p.Percent / videoDuration) * 100.0
			if percent > 100.0 {
				percent = 100.0
			}
			fmt.Printf("\r       > Progress: %.2f%%", percent)
		}
		progressDone <- true
	}()

	r.writeFrames(stdin, videoDuration)

	stdin.Close()
	err = cmd.Wait()

	<-progressDone
	if err != nil {
		return fmt.Errorf("ffmpeg exited with error: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (r *Renderer) DrawBackground(dc *gg.Context) {
	c := r.s.Visuals.Background.RGB
	dc.SetRGB255(c.ToInt())
	dc.Clear()
}

func (r *Renderer) SetupNote(dc *gg.Context, note parser.Note, currentTime, shiftX, shiftY float64) {
	ad := r.s.Gameplay.ApproachDistance
	ar := r.s.Gameplay.ApproachRate
	at := ad / ar

	depth := (float64(note.Time) - currentTime) / (1000 * at) * ad / float64(r.Replay.SpeedMultiplier)

	if depth > ad || depth < 0 {
		return
	}

	perspective := game.CalcPerspective(depth)
	currentNoteSize := r.PlayAreaSize * (game.NoteSize / game.GridSize) * perspective

	hbSizeUnit := game.HitboxSizeNormal // TODO: change size based on replay mods
	currentHitboxSize := r.PlayAreaSize * (hbSizeUnit / game.GridSize) * perspective

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
		r.DrawNote(dc, alpha, note.NoteIdx, drawX-currentNoteSize/2, drawY-currentNoteSize/2, currentNoteSize, perspective)
		r.DrawHitbox(dc, drawX-currentHitboxSize/2, drawY-currentHitboxSize/2, currentHitboxSize) // TODO: add toggle to settings
	}
}

type ShapeDrawer func(s config.Shape, dc *gg.Context, x, y, size float64)

var noteShapes = map[string]ShapeDrawer{
	"square": func(s config.Shape, dc *gg.Context, x, y, size float64) {
		dc.DrawRoundedRectangle(x, y, size, size, s.RoundCorners*size)
	},
	"circle": func(s config.Shape, dc *gg.Context, x, y, size float64) {
		radius := size / 2
		dc.DrawCircle(x+radius, y+radius, radius)
	},
	"ngon": func(s config.Shape, dc *gg.Context, x, y, size float64) {
		drawNgon(dc, x, y, size, s.Ngon.Sides, s.Ngon.Angle)
	},
	"weirdo": func(s config.Shape, dc *gg.Context, x, y, size float64) {
		radius := size * s.RoundCorners
		rc := s.Weirdo.RoundedCorners
		drawCustomRounded(dc, x, y, size, radius, rc[0], rc[1], rc[2], rc[3])
	},
}

func drawNgon(dc *gg.Context, x, y, size float64, n int, a float64) {
	if n < 3 {
		n = 3
	}
	radius := size / 2
	cx, cy := x+radius, y+radius

	rotation := a * (math.Pi / 180.0)

	for i := 0; i < n; i++ {
		angle := float64(i)*2*math.Pi/float64(n) - math.Pi/2 + rotation

		px := cx + radius*math.Cos(angle)
		py := cy + radius*math.Sin(angle)

		if i == 0 {
			dc.MoveTo(px, py)
		} else {
			dc.LineTo(px, py)
		}
	}
	dc.ClosePath()
}

func drawCustomRounded(dc *gg.Context, x, y, s, r float64, tl, tr, br, bl bool) {
	if tl {
		dc.MoveTo(x+r, y)
	} else {
		dc.MoveTo(x, y)
	}

	if tr {
		dc.LineTo(x+s-r, y)
		dc.DrawArc(x+s-r, y+r, r, 1.5*math.Pi, 2*math.Pi)
	} else {
		dc.LineTo(x+s, y)
	}

	if br {
		dc.LineTo(x+s, y+s-r)
		dc.DrawArc(x+s-r, y+s-r, r, 0, 0.5*math.Pi)
	} else {
		dc.LineTo(x+s, y+s)
	}

	if bl {
		dc.LineTo(x+r, y+s)
		dc.DrawArc(x+r, y+s-r, r, 0.5*math.Pi, math.Pi)
	} else {
		dc.LineTo(x, y+s)
	}

	if tl {
		dc.LineTo(x, y+r)
		dc.DrawArc(x+r, y+r, r, math.Pi, 1.5*math.Pi)
	} else {
		dc.LineTo(x, y)
	}
	dc.ClosePath()
}

func (r *Renderer) DrawNote(dc *gg.Context, alpha float64, noteIdx int, x, y, size, perspective float64) {
	s := r.s.Visuals.Note.Shape
	f := r.s.Visuals.Note.Fill

	if drawer, ok := noteShapes[s.NoteShape]; ok {
		drawer(s, dc, x, y, size)
	} else {
		dc.DrawRoundedRectangle(x, y, size, size, s.RoundCorners*size)
	}

	if f.Enabled {
		if f.Mode == "custom" && len(f.Custom.RGBA) > 0 {
			fillColor := f.Custom.RGBA[noteIdx%len(f.Custom.RGBA)]
			dc.SetRGBA255(fillColor.ToInt())
		} else {
			strokeColor := r.s.Visuals.Note.RGB[noteIdx%len(r.s.Visuals.Note.RGB)]
			dc.SetRGBA255(strokeColor.ToIntAlpha(alpha * (float64(f.Solid.Alpha) / 255.0)))
		}
		dc.FillPreserve()
	}

	color := r.s.Visuals.Note.RGB[noteIdx%len(r.s.Visuals.Note.RGB)]
	dc.SetRGBA255(color.ToIntAlpha(alpha))
	dc.SetLineWidth(s.LineWidth * perspective)
	dc.Stroke()
}

func (r *Renderer) DrawCorners(dc *gg.Context, x, y, size float64) {
	c := r.s.Visuals.Background.Corners
	dc.SetRGBA255(c.RGBA.ToInt())
	dc.SetLineWidth(c.LineWidth)

	actualLength := (size / 2.0) * c.Length

	radius := (size / 2.0) * c.RoundCorners
	if radius > actualLength {
		radius = actualLength
	}

	if radius > 0 {
		dc.MoveTo(x, y+actualLength)
		dc.LineTo(x, y+radius)
		dc.DrawArc(x+radius, y+radius, radius, math.Pi, 1.5*math.Pi)
		dc.LineTo(x+actualLength, y)
	} else {
		dc.MoveTo(x, y+actualLength)
		dc.LineTo(x, y)
		dc.LineTo(x+actualLength, y)
	}
	dc.Stroke()

	if radius > 0 {
		dc.MoveTo(x+size-actualLength, y)
		dc.LineTo(x+size-radius, y)
		dc.DrawArc(x+size-radius, y+radius, radius, 1.5*math.Pi, 2*math.Pi)
		dc.LineTo(x+size, y+actualLength)
	} else {
		dc.MoveTo(x+size-actualLength, y)
		dc.LineTo(x+size, y)
		dc.LineTo(x+size, y+actualLength)
	}
	dc.Stroke()

	if radius > 0 {
		dc.MoveTo(x+size, y+size-actualLength)
		dc.LineTo(x+size, y+size-radius)
		dc.DrawArc(x+size-radius, y+size-radius, radius, 0, 0.5*math.Pi)
		dc.LineTo(x+size-actualLength, y+size)
	} else {
		dc.MoveTo(x+size, y+size-actualLength)
		dc.LineTo(x+size, y+size)
		dc.LineTo(x+size-actualLength, y+size)
	}
	dc.Stroke()

	if radius > 0 {
		dc.MoveTo(x+actualLength, y+size)
		dc.LineTo(x+radius, y+size)
		dc.DrawArc(x+radius, y+size-radius, radius, 0.5*math.Pi, math.Pi)
		dc.LineTo(x, y+size-actualLength)
	} else {
		dc.MoveTo(x+actualLength, y+size)
		dc.LineTo(x, y+size)
		dc.LineTo(x, y+size-actualLength)
	}
	dc.Stroke()
}

func (r *Renderer) DrawCursor(dc *gg.Context, x, y, shiftX, shiftY float64) {
	size := r.PlayAreaSize * ((game.CursorSize * config.Current.Visuals.Cursor.Size) / game.GridSize)

	relX, relY := game.CursorToScreen(float64(x), float64(y), r.PlayAreaSize)

	centerX, centerY := (float64(r.Width)/2.0)+shiftX, (float64(r.Height)/2.0)+shiftY
	screenX, screenY := centerX+relX, centerY+relY

	s := r.s.Visuals.Cursor.Shape
	f := r.s.Visuals.Cursor.Fill

	drawX := screenX - size/2
	drawY := screenY - size/2

	if drawer, ok := noteShapes[s.NoteShape]; ok {
		drawer(s, dc, drawX, drawY, size)
	} else {
		dc.DrawCircle(screenX, screenY, size/2)
	}

	if f.Enabled {
		if f.Mode == "custom" {
			dc.SetRGBA255(f.Custom.RGBA.ToInt())
		} else {
			strokeColor := r.s.Visuals.Cursor.RGBA
			strokeColor[3] = f.Solid.Alpha
			dc.SetRGBA255(strokeColor.ToInt())
		}
		dc.FillPreserve()
	}

	color := r.s.Visuals.Cursor.RGBA
	dc.SetRGBA255(color.ToInt())
	dc.SetLineWidth(s.LineWidth)
	dc.Stroke()
}

func (r *Renderer) DrawHitbox(dc *gg.Context, x, y, size float64) {
	dc.SetRGBA255(255, 0, 0, 255)
	dc.SetLineWidth(1.0)
	dc.DrawRectangle(x, y, size, size)
	dc.Stroke()
}

func lerp32(a, b float32, t float64) float64 { return float64(a + float32(t)*(b-a)) }
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
