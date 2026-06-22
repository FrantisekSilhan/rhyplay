package render

type RenderNote struct {
	NoteIdx    int
	BaseX      float64
	BaseY      float64
	TargetTime float64
	Status     int
	ResolvedAt float64
}

type RenderFrame struct {
	Progress float64
	X        float64
	Y        float64
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
		sx, sy := r.gameToScreen(n.X, n.Y, 1.0)
		r.RenderNotes[i] = RenderNote{
			NoteIdx:    i,
			BaseX:      sx,
			BaseY:      sy,
			TargetTime: float64(n.Time),
			Status:     StatusPending,
		}
	}

	r.RenderFrames = make([]RenderFrame, len(r.Replay.Frames))
	for i, f := range r.Replay.Frames {
		sx, sy := r.cursorToScreen(f.X, f.Y)
		r.RenderFrames[i] = RenderFrame{
			Progress: float64(f.Progress),
			X:        sx,
			Y:        -sy,
			Hit:      f.Hit,
		}
	}
}
