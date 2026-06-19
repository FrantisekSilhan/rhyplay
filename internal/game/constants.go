package game

const (
	Padding = 0.2

	HitWindowMS = 55.0
	GridSize    = 3.0
	CursorSize  = 0.2625
	HitBoxSize  = 0.07
	NoteSize    = 0.875

	BaseLineWidth = 20.0
	ViewDistance  = 3.75

	FadeIn     = 15
	FadeOut    = 25
	MinFadeOut = 0.25
)

func GetEffectiveHitWindows(speed float32) float64 {
	return HitWindowMS * float64(speed)
}

func CalcPerspective(depth float64) float64 {
	return ViewDistance / (depth + ViewDistance)
}

func GameToScreen(gx, gy, playAreaSize, perspective float64) (sx, sy float64) {
	scale := playAreaSize / GridSize
	sx = gx * scale * perspective
	sy = gy * scale * perspective
	return
}

func CursorToScreen(cx, cy, playAreaSize float64) (sx, sy float64) {
	scale := playAreaSize / GridSize
	sx = cx * scale
	sy = cy * scale
	return
}
