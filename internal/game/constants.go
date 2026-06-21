package game

const (
	BaseHeight       = 1080.0
	BasePlayAreaSize = 644.0
	GridSize         = 3.0

	HitWindowMS = 55.0
	CursorSize  = 0.2625
	NoteSize    = 0.875

	ViewDistance = 3.75

	HitboxSizeNormal   = (250.0 / BasePlayAreaSize) * GridSize
	HitboxSizeHardrock = (200.0 / BasePlayAreaSize) * GridSize

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
