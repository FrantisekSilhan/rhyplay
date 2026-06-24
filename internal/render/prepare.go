package render

import (
	"fmt"

	"github.com/fogleman/gg"
)

type RenderNote struct {
	NoteIdx    int
	BaseX      float64
	BaseY      float64
	RawX       float64
	RawY       float64
	TargetTime float64
	Status     int
	ResolvedAt float64
}

type RenderFrame struct {
	Progress float64
	X        float64
	Y        float64
	RawX     float64
	RawY     float64
	Hit      bool
}

func (r *Renderer) gameToScreen(gx, gy, perspective float64) (sx, sy float64) {
	scale := r.NoteScale * perspective
	sx = gx * scale
	sy = gy * scale
	return
}

func (r *Renderer) cursorToScreen(cx, cy float32) (sx, sy float64) {
	scale := r.CursorScale
	sx = float64(cx) * scale
	sy = float64(cy) * scale
	return
}

func (r *Renderer) prepareData() {
	r.RenderNotes = make([]RenderNote, len(r.Beatmap.Notes))
	for i, n := range r.Beatmap.Notes {
		//sx, sy := r.gameToScreen(n.X, n.Y, 1.0)
		r.RenderNotes[i] = RenderNote{
			NoteIdx:    i,
			BaseX:      n.X * r.NoteScale,
			BaseY:      n.Y * r.NoteScale,
			RawX:       n.X,
			RawY:       n.Y,
			TargetTime: float64(n.Time),
			Status:     StatusPending,
		}
	}

	r.RenderFrames = make([]RenderFrame, len(r.Replay.Frames))
	for i, f := range r.Replay.Frames {
		//sx, sy := r.cursorToScreen(f.X, f.Y)
		r.RenderFrames[i] = RenderFrame{
			Progress: float64(f.Progress),
			X:        float64(f.X) * r.CursorScale,
			Y:        float64(-f.Y) * r.CursorScale,
			RawX:     float64(f.X),
			RawY:     float64(-f.Y),
			Hit:      f.Hit,
		}
	}
}

func (r *Renderer) loadFonts() error {
	baseSize := 24.0 * r.ResScale
	largeSize := 48.0 * r.ResScale

	fReg, err := gg.LoadFontFace("assets/fonts/Nunito-Regular.ttf", baseSize)
	if err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}
	r.Font.Regular = fReg

	fSemi, err := gg.LoadFontFace("assets/fonts/Nunito-SemiBold.ttf", baseSize)
	if err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}
	r.Font.SemiBold = fSemi

	fLarge, err := gg.LoadFontFace("assets/fonts/Nunito-ExtraBold.ttf", largeSize)
	if err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}
	r.Font.Large = fLarge

	fExtraBold, err := gg.LoadFontFace("assets/fonts/Nunito-ExtraBold.ttf", baseSize)
	if err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}
	r.Font.ExtraBold = fExtraBold

	return nil
}
