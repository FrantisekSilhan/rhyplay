package game

type Constants struct {
	NoteUnitToPx       float64
	CursorUnitToPx     float64
	HitboxSize         float64
	BackgroundDrawSize float64
}

const (
	BaseHeight     = 1080.0
	NoteUnitToPx   = 440.0 / 2.00
	CursorUnitToPx = 606.0 / 2.74
	HitboxSize     = 250.0

	// Hardrock
	NoteUnitToPxHR   = 484 / 2.00
	CursorUnitToPxHR = 672 / 2.74
	HitboxSizeHR     = 200.0

	NoteDrawSize         = 182.0
	CursorDrawSize       = 56.0
	BackgroundDrawSize   = 656.0
	BackgroundDrawSizeHR = 722.0

	HitWindowMS = 55.0

	ViewDistance = 3.75

	FadeIn     = 15
	FadeOut    = 25
	MinFadeOut = 0.25

	MissDuration = 400.0
)

func GetEffectiveHitWindow(speed float32) float64 {
	return HitWindowMS * float64(speed)
}

func CalcPerspective(depth float64) float64 {
	return ViewDistance / (depth + ViewDistance)
}
