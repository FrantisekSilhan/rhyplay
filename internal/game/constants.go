package game

type Constants struct {
	NoteDrawSize        float64
	NoteHitboxDrawSize  float64
	CursorDrawSize      float64
	BackgroundDrawBound float64
	HitThreshold        float64
}

const (
	BaseHeight = 1080.0

	HitboxRadiusNormal = 0.5
	CursorRadiusNormal = 0.07

	HitboxRadiusHR = 0.4
	CursorRadiusHR = 0.056

	NoteUnitNormal    = 219.0
	CursorBoundNormal = 1.36874997615814 // (3.0 / 2.0) - (0.2625 / 2.0)

	NoteUnitHR    = 241.0
	CursorBoundHR = 1.51875007152557

	NoteDrawSize         = 182.0
	CursorDrawSize       = 56.0
	NoteHitboxDrawSize   = 250.0
	NoteHitboxDrawSizeHR = 200.0

	CursorDrawBound       = 606.0
	BackgroundDrawBound   = 654.0
	BackgroundDrawBoundHR = 720.0

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
