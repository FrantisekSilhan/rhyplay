package render

import (
	"math"
	"rhyplay/internal/config"
	"rhyplay/internal/game"

	"github.com/fogleman/gg"
)

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

func (r *Renderer) DrawNote(dc *gg.Context, alpha float64, noteIdx int, cx, cy, size, perspective float64) {
	x, y := cx-size/2, cy-size/2

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
	dc.SetLineWidth(s.LineWidth * r.ResScale * perspective)
	dc.Stroke()
}

func (r *Renderer) DrawUI(dc *gg.Context, shiftX, shiftY float64) {
	r.DrawCorners(dc, shiftX, shiftY)
}

func (r *Renderer) DrawCorners(dc *gg.Context, shiftX, shiftY float64) {
	c := r.s.Visuals.Background.Corners

	pathSize := r.c.BackgroundDrawSize * r.ResScale
	lineWidth := r.s.Visuals.Background.Corners.LineWidth * r.ResScale

	x, y := shiftX-pathSize/2, shiftY-pathSize/2

	dc.SetRGBA255(c.RGBA.ToInt())
	dc.SetLineWidth(lineWidth)

	actualLength := (pathSize / 2.0) * c.Length

	radius := (pathSize / 2.0) * c.RoundCorners
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
		dc.MoveTo(x+pathSize-actualLength, y)
		dc.LineTo(x+pathSize-radius, y)
		dc.DrawArc(x+pathSize-radius, y+radius, radius, 1.5*math.Pi, 2*math.Pi)
		dc.LineTo(x+pathSize, y+actualLength)
	} else {
		dc.MoveTo(x+pathSize-actualLength, y)
		dc.LineTo(x+pathSize, y)
		dc.LineTo(x+pathSize, y+actualLength)
	}
	dc.Stroke()

	if radius > 0 {
		dc.MoveTo(x+pathSize, y+pathSize-actualLength)
		dc.LineTo(x+pathSize, y+pathSize-radius)
		dc.DrawArc(x+pathSize-radius, y+pathSize-radius, radius, 0, 0.5*math.Pi)
		dc.LineTo(x+pathSize-actualLength, y+pathSize)
	} else {
		dc.MoveTo(x+pathSize, y+pathSize-actualLength)
		dc.LineTo(x+pathSize, y+pathSize)
		dc.LineTo(x+pathSize-actualLength, y+pathSize)
	}
	dc.Stroke()

	if radius > 0 {
		dc.MoveTo(x+actualLength, y+pathSize)
		dc.LineTo(x+radius, y+pathSize)
		dc.DrawArc(x+radius, y+pathSize-radius, radius, 0.5*math.Pi, math.Pi)
		dc.LineTo(x, y+pathSize-actualLength)
	} else {
		dc.MoveTo(x+actualLength, y+pathSize)
		dc.LineTo(x, y+pathSize)
		dc.LineTo(x, y+pathSize-actualLength)
	}
	dc.Stroke()
}

func (r *Renderer) DrawCursor(dc *gg.Context, x, y, shiftX, shiftY float64) {
	userScale := r.s.Visuals.Cursor.Size

	visualSize := game.CursorDrawSize * r.ResScale * userScale
	lineWidth := r.s.Visuals.Cursor.Shape.LineWidth * r.ResScale * userScale

	screenX, screenY := shiftX+x, shiftY+y

	s := r.s.Visuals.Cursor.Shape
	f := r.s.Visuals.Cursor.Fill

	pathSize := visualSize - lineWidth
	drawX := screenX - pathSize/2
	drawY := screenY - pathSize/2

	if drawer, ok := noteShapes[s.NoteShape]; ok {
		drawer(s, dc, drawX, drawY, pathSize)
	} else {
		dc.DrawCircle(screenX, screenY, pathSize/2)
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
	dc.SetLineWidth(lineWidth)
	dc.Stroke()
}

func (r *Renderer) DrawHitbox(dc *gg.Context, x, y, size float64) {
	dc.SetRGBA255(255, 0, 0, 255)
	dc.SetLineWidth(1.0)
	dc.DrawRectangle(x, y, size, size)
	dc.Stroke()
}

func (r *Renderer) DrawBackground(dc *gg.Context) {
	c := r.s.Visuals.Background.RGB
	dc.SetRGB255(c.ToInt())
	dc.Clear()
}

func (r *Renderer) DrawCollision(dc *gg.Context, rn RenderNote, curX, curY float64, shiftX, shiftY float64) {
	noteDrawX := r.OffsetX + shiftX + rn.BaseX
	noteDrawY := r.OffsetY + shiftY + rn.BaseY

	cursorDrawX := r.OffsetX + shiftX + curX
	cursorDrawY := r.OffsetY + shiftY + curY

	hitboxSize := r.c.HitboxSize * r.ResScale

	dc.SetRGBA255(255, 0, 0, 150)
	dc.SetLineWidth(2.0)
	dc.DrawRectangle(noteDrawX-hitboxSize/2, noteDrawY-hitboxSize/2, hitboxSize, hitboxSize)
	dc.Stroke()

	dc.SetRGBA255(255, 0, 0, 255)
	dc.DrawCircle(cursorDrawX, cursorDrawY, 5)
	dc.Fill()

	dc.SetRGBA255(255, 255, 0, 100)
	dc.DrawLine(cursorDrawX, cursorDrawY, noteDrawX, noteDrawY)
	dc.Stroke()
}
