package render

import (
	"fmt"
	"math"
	"rhyplay/internal/config"
	"rhyplay/internal/game"
	"strconv"
	"strings"

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
	dc.SetRGBA255(color.ToIntAlpha(alpha * r.s.Visuals.Note.Opacity))
	dc.SetLineWidth(s.LineWidth * r.ResScale * perspective)
	dc.Stroke()
}

func (r *Renderer) DrawUI(dc *gg.Context, shiftX, shiftY float64) {
	r.DrawCorners(dc, shiftX, shiftY)
	dc.SetFontFace(r.Font.ExtraBold)

	pathSize := r.c.BackgroundDrawBound * r.ResScale
	padding := 120.0 * r.ResScale

	pl := shiftX - pathSize/2
	pr := shiftX + pathSize/2
	pt := shiftY - pathSize/2

	processed := r.Score.HitCount + r.Score.MissCount
	i := r.s.Visuals.Interface
	rightPanelStats := []Stat{}
	if i.RightPanel.ShowScore {
		rightPanelStats = append(rightPanelStats, Stat{Label: "SCORE", Value: numberToString(r.Score.Score)})
	}
	if i.RightPanel.ShowPoints {
		rightPanelStats = append(rightPanelStats, Stat{Label: "POINTS", Value: fmt.Sprintf("%d", r.calculateRP())})
	}
	if i.RightPanel.ShowMisses {
		rightPanelStats = append(rightPanelStats, Stat{Label: "MISSES", Value: fmt.Sprintf("%d", r.Score.MissCount)})
	}
	if i.RightPanel.ShowNotes {
		rightPanelStats = append(rightPanelStats, Stat{Label: "NOTES", Value: fmt.Sprintf("%d/%d", r.Score.HitCount, processed)})
	}
	r.drawRightPanel(dc, pr+padding, pt, pathSize, rightPanelStats)

	acc := -1.0
	if processed > 0 {
		acc = (float64(r.Score.HitCount) / float64(processed)) * 100.0
	}

	leftPanelStats := []Stat{}
	if i.LeftPanel.ShowCombo {
		leftPanelStats = append(leftPanelStats, Stat{Label: "COMBO", Value: fmt.Sprintf("%d", r.Score.Combo)})
	}
	if i.LeftPanel.ShowGrade {
		label := "GRADE"
		if acc < 0 {
			leftPanelStats = append(leftPanelStats, Stat{Label: label, Value: "--", IsRank: true})
		} else {
			rank, color := getRank(acc)
			leftPanelStats = append(leftPanelStats, Stat{Label: label, Value: rank, IsRank: true, Color: color})
		}
	}
	if i.LeftPanel.ShowAccuracy {
		label := "ACCURACY"
		val := "--"
		if acc >= 0 {
			val = formatAccuracy(acc)
		}
		leftPanelStats = append(leftPanelStats, Stat{Label: label, Value: val})
	}

	r.drawLeftPanel(dc, pl-padding, pt, pathSize, leftPanelStats)
}

func (r *Renderer) drawRightPanel(dc *gg.Context, alignX, pt float64, pathSize float64, stats []Stat) {
	items := len(stats)
	if items == 0 {
		return
	}
	slotHeight := pathSize / float64(items)

	for i, stat := range stats {
		y := pt + (float64(i) * slotHeight) + (slotHeight / 2.0)

		r.drawStat(dc, stat.Label, stat.Value, alignX, y)
	}
}

func (r *Renderer) drawLeftPanel(dc *gg.Context, alignX, pt float64, pathSize float64, stats []Stat) {
	items := len(stats)
	if items == 0 {
		return
	}
	slotHeight := pathSize / float64(items)

	for i, stat := range stats {
		y := pt + (float64(i) * slotHeight) + (slotHeight / 2.0)

		if stat.IsRank {
			if stat.Value != "--" {
				r.drawRank(dc, stat.Value, stat.Color, alignX, y)
			}
		} else {
			r.drawStat(dc, stat.Label, stat.Value, alignX, y)
		}

	}
}

type Stat struct {
	Label  string
	Value  string
	IsRank bool
	Color  config.RGB
}

func numberToString(n int) string {
	s := strconv.Itoa(n)
	res := ""
	for i, v := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			res += ","
		}
		res += string(v)
	}
	return res
}

func formatAccuracy(acc float64) string {
	s := strconv.FormatFloat(acc, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s + "%"
}

func getRank(acc float64) (string, config.RGB) {
	if acc == 100 {
		return "SS", config.RGB{150, 82, 227}
	}
	if acc >= 98 {
		return "S", config.RGB{152, 145, 255}
	}
	if acc >= 95 {
		return "A", config.RGB{145, 255, 146}
	}
	if acc >= 90 {
		return "B", config.RGB{231, 255, 192}
	}
	if acc >= 85 {
		return "C", config.RGB{252, 247, 179}
	}
	return "D", config.RGB{252, 208, 179}
}

func (r *Renderer) drawRank(dc *gg.Context, rank string, color config.RGB, x, y float64) {
	dc.SetFontFace(r.Font.Large)
	dc.SetRGB255(color.ToInt())
	dc.DrawStringAnchored(rank, x, y, 0.5, 0.5)
}

func (r *Renderer) drawStat(dc *gg.Context, label, value string, x, y float64) {
	verticalItemGap := 16.0 * r.ResScale

	dc.SetFontFace(r.Font.ExtraBold)
	dc.SetRGB255(160, 160, 160)
	dc.DrawStringAnchored(label, x, y-verticalItemGap, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(value, x, y+verticalItemGap, 0.5, 0.5)
}

func (r *Renderer) DrawCorners(dc *gg.Context, shiftX, shiftY float64) {
	c := r.s.Visuals.Interface.Corners

	pathSize := r.c.BackgroundDrawBound * r.ResScale
	lineWidth := r.s.Visuals.Interface.Corners.LineWidth * r.ResScale

	x, y := shiftX-pathSize/2, shiftY-pathSize/2

	dc.SetRGBA255(c.RGBA.ToInt())
	dc.SetLineWidth(lineWidth)

	actualLength := (pathSize / 2.0) * c.Length

	radius := (pathSize / 2.0) * c.RoundCorners
	if radius > actualLength {
		radius = actualLength
	}

	if c.Length == 1 {
		dc.DrawRoundedRectangle(x, y, pathSize, pathSize, radius)
		dc.Stroke()
		return
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

	visualSize := r.c.CursorDrawSize * r.ResScale * userScale
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
	dc.SetRGBA255(255, 255, 255, 255)
	dc.SetLineWidth(1.0)
	dc.DrawRectangle(x, y, size, size)
	dc.Stroke()
}

func (r *Renderer) DrawBackground(dc *gg.Context) {
	c := r.s.Visuals.Interface.BackgroundRGB
	dc.SetRGB255(c.ToInt())
	dc.Clear()
}

func (r *Renderer) DrawCollision(dc *gg.Context, rn RenderNote, curX, curY float64, shiftX, shiftY float64) {
	noteDrawX := shiftX + rn.BaseX
	noteDrawY := shiftY + rn.BaseY

	cursorDrawX := shiftX + curX
	cursorDrawY := shiftY + curY

	hitboxSize := r.c.NoteHitboxDrawSize * r.ResScale

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

func (r *Renderer) DrawMiss(dc *gg.Context, rn RenderNote, engineTime, shiftX, shiftY float64, dir int) {
	progress := (engineTime - rn.ResolvedAt) / game.MissDuration
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		return
	}

	angle := float64(dir) * 20.0 * (math.Pi / 180.0)
	lineWidth := r.s.Visuals.Miss.LineWidth * r.ResScale
	halfSize := 2.0 * lineWidth

	var scale, alpha, rotation float64

	if progress <= 0.25 {
		p := progress / 0.25
		scale = p
		alpha = p
	} else if progress <= 0.50 {
		scale = 1.0
		alpha = 1.0
	} else {
		p := (progress - 0.50) / 0.50
		scale = 1.0 - p
		alpha = 1.0 - p
	}

	if progress < 0.20 {
		rotation = angle
	} else if progress > 0.55 {
		rotation = 0
	} else {
		rp := (progress - 0.20) / (0.55 - 0.20)
		smoothP := rp * rp * (3 - 2*rp)
		rotation = angle * (1.0 - smoothP)
	}

	drawX := shiftX + rn.BaseX
	drawY := shiftY + rn.BaseY

	dc.Push()
	dc.Translate(drawX, drawY)
	dc.Rotate(rotation)
	dc.Scale(scale, scale)

	dc.SetRGBA255(r.s.Visuals.Miss.RGB.ToIntAlpha(alpha))
	dc.SetLineWidth(lineWidth)
	dc.SetLineCap(gg.LineCapRound)

	dc.DrawLine(-halfSize, -halfSize, halfSize, halfSize)
	dc.DrawLine(halfSize, -halfSize, -halfSize, halfSize)
	dc.Stroke()
	dc.Pop()
}
