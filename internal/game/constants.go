package game

const (
	BaseHeight     = 1080.0
	NoteUnitToPx   = 440.0 / 2.00
	CursorUnitToPx = 606.0 / 2.74
	HitboxSize     = 250.0

	NoteDrawSize       = 182.0
	CursorDrawSize     = 56.0
	BackgroundDrawSize = 656.0

	HitWindowMS = 55.0

	ViewDistance = 3.75

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

func GameToScreen(gx, gy, resScale, perspective float64) (sx, sy float64) {
	scale := NoteUnitToPx * resScale
	sx = gx * scale * perspective
	sy = gy * scale * perspective
	return
}

func CursorToScreen(cx, cy, resScale float64) (sx, sy float64) {
	scale := CursorUnitToPx * resScale
	sx = cx * scale
	sy = cy * scale
	return
}
