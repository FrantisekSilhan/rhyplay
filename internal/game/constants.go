package game

const (
	Padding = 0.2

	HitWindowMS       = 55.0
	InternalGridSize  = 2.74
	CursorSensitivity = 1.37
	HitboxScale       = 1.2

	NoteSizeMultiplier = 0.3
	BaseLineWidth      = 20.0
	ViewDistance       = 2.0
)

func GetEffectiveHitWindows(speed float32) float64 {
	return HitWindowMS * float64(speed)
}

func CalcPerspective(depth float64) float64 {
	return ViewDistance / (depth + ViewDistance)
}

func GameToScreen(gx, gy, playAreaSize, perspective float64) (sx, sy float64) {
	relX := (gx - 1.0) * 0.5
	relY := (gy - 1.0) * 0.5

	currentSize := playAreaSize * NoteSizeMultiplier * perspective
	currentLineWidth := BaseLineWidth * perspective
	visualTotalSize := currentSize + (currentLineWidth * 2)
	usableArea := playAreaSize - visualTotalSize

	sx = relX * usableArea * perspective
	sy = relY * usableArea * perspective
	return
}

func CursorToScreen(cx, cy, playAreaSize float64) (sx, sy float64) {
	hitboxSize := playAreaSize * 0.06
	usableArea := playAreaSize - hitboxSize

	sx = (cx / InternalGridSize) * usableArea
	sy = (cy / InternalGridSize) * usableArea
	return
}
