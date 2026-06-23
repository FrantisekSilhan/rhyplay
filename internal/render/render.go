package render

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"rhyplay/internal/config"
	"rhyplay/internal/ffmpeg"
	"rhyplay/internal/game"
	"rhyplay/internal/parser"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type NoteConstants struct {
	at        float64
	horizon   float64
	depthStep float64
}

type Score struct {
	HitCount  int
	MissCount int
	Combo     int
	Score     int

	NextPendingNoteIdx int
}

type Font struct {
	Regular   font.Face
	SemiBold  font.Face
	Large     font.Face
	ExtraBold font.Face
}

type Renderer struct {
	s       *config.Settings
	c       *game.Constants
	nc      NoteConstants
	Beatmap *parser.MapData
	Replay  *parser.ReplayData

	Width, Height int
	FPS           int

	ResScale         float64
	OffsetX, OffsetY float64

	CursorScale    float64
	NoteScale      float64
	ParallaxFactor float64

	RenderNotes  []RenderNote
	RenderFrames []RenderFrame

	LastProcessedTime float64

	Score Score
	Font  Font
}

func NewRenderer(b *parser.MapData, r *parser.ReplayData) (*Renderer, error) {
	s := config.Current
	w, h := s.Video.Width, s.Video.Height

	resScale := float64(h) / game.BaseHeight
	if float64(h) > float64(w) {
		resScale = float64(w) / game.BaseHeight
	}

	c := &game.Constants{
		NoteUnitToPx:       game.NoteUnitToPx,
		CursorUnitToPx:     game.CursorUnitToPx,
		HitboxSize:         game.HitboxSize,
		BackgroundDrawSize: game.BackgroundDrawSize,
	}
	cursorScale := game.CursorUnitToPx * resScale
	noteScale := game.NoteUnitToPx * resScale

	if r.ModState.HardrockEnabled {
		c.NoteUnitToPx = game.NoteUnitToPxHR
		c.CursorUnitToPx = game.CursorUnitToPxHR
		c.HitboxSize = game.HitboxSizeHR
		c.BackgroundDrawSize = game.BackgroundDrawSizeHR
		cursorScale = game.CursorUnitToPxHR * resScale
		noteScale = game.NoteUnitToPxHR * resScale
	}

	at := s.Gameplay.ApproachDistance / s.Gameplay.ApproachRate
	horizon := (1000 * at) * float64(r.ModState.SpeedMultiplier)
	depthStep := s.Gameplay.ApproachDistance / horizon

	noteConstants := NoteConstants{
		at:        at,
		horizon:   horizon,
		depthStep: depthStep,
	}

	renderer := &Renderer{
		s:              s,
		c:              c,
		nc:             noteConstants,
		Beatmap:        b,
		Replay:         r,
		Width:          w,
		Height:         h,
		FPS:            s.Video.FPS,
		ResScale:       resScale,
		CursorScale:    cursorScale,
		NoteScale:      noteScale,
		ParallaxFactor: s.Visuals.Parallax / cursorScale,
		OffsetX:        float64(w) / 2.0,
		OffsetY:        float64(h) / 2.0,

		RenderNotes:  make([]RenderNote, len(b.Notes)),
		RenderFrames: make([]RenderFrame, len(r.Frames)),

		LastProcessedTime: -1,

		Score: Score{
			HitCount:           0,
			MissCount:          0,
			Combo:              0,
			Score:              0,
			NextPendingNoteIdx: 0,
		},
		Font: Font{},
	}

	renderer.prepareData()
	err := renderer.loadFonts()
	if err != nil {
		return nil, err
	}

	return renderer, nil
}

func (r *Renderer) writeFrames(stdin io.WriteCloser, videoDuration float64) {
	defer stdin.Close()
	msPerFrame := 1000.0 / float64(r.FPS)
	dc := gg.NewContext(r.Width, r.Height)
	replayIdx := 0
	noteWindowIdx := 0

	for currentTime := 0.0; currentTime <= videoDuration; currentTime += msPerFrame {
		engineTime := currentTime * float64(r.Replay.ModState.SpeedMultiplier)
		var replayWindow []RenderFrame
		for replayIdx < len(r.RenderFrames) {
			f := r.RenderFrames[replayIdx]

			if f.Progress > engineTime {
				break
			}

			if f.Progress > r.LastProcessedTime {
				replayWindow = append(replayWindow, f)
			}
			replayIdx++
		}

		idx := replayIdx
		if idx >= len(r.RenderFrames) {
			idx = len(r.RenderFrames) - 1
		}
		if idx < 1 {
			idx = 1
		}

		f1, f2 := r.RenderFrames[idx-1], r.RenderFrames[idx]
		alpha := calculateAlpha(float64(f1.Progress), float64(f2.Progress), engineTime)
		curX, curY := lerp64(f1.X, f2.X, alpha), lerp64(f1.Y, f2.Y, alpha)

		for noteWindowIdx < len(r.RenderNotes) && float64(r.RenderNotes[noteWindowIdx].TargetTime) < engineTime-(game.MissDuration+100) {
			noteWindowIdx++
		}

		shiftX, shiftY := r.OffsetX-curX*r.ParallaxFactor, r.OffsetY-curY*r.ParallaxFactor

		r.DrawBackground(dc)

		if r.s.Debug.ShowCollision {
			for _, n := range r.RenderNotes {
				if n.Status == StatusPending && n.TargetTime < engineTime+500 {
					r.DrawCollision(dc, n, curX, curY, shiftX, shiftY)
					break
				}
			}
		}

		r.updateScoreLogic(dc, engineTime, replayWindow, r.OffsetX-curX*r.ParallaxFactor, r.OffsetY-curY*r.ParallaxFactor)
		r.LastProcessedTime = engineTime

		r.SetupNotes(dc, engineTime, noteWindowIdx, shiftX, shiftY)
		r.DrawCursor(dc, curX, curY, shiftX, shiftY)

		r.DrawUI(dc, shiftX, shiftY)
		stdin.Write(dc.Image().(*image.RGBA).Pix)
	}
}

func (r *Renderer) Render(outputPath string, audioPath string) error {
	if len(r.Replay.Frames) < 2 {
		return fmt.Errorf("replay contains too few frames")
	}

	replayEndTime := float64(r.Replay.Frames[len(r.Replay.Frames)-1].Progress)
	videoDuration := replayEndTime / float64(r.Replay.ModState.SpeedMultiplier)

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
