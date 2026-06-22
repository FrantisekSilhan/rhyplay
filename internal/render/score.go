package render

import (
	"math"
	"rhyplay/internal/game"

	"github.com/fogleman/gg"
)

const (
	StatusPending = iota
	StatusHit
	StatusMiss
)

func (r *Renderer) updateScoreLogic(dc *gg.Context, engineTime float64, replayWindow []RenderFrame, shiftX, shiftY float64) {
	hitWindow := game.GetEffectiveHitWindow(r.Replay.ModState.SpeedMultiplier)
	halfHitbox := (r.c.HitboxSize*r.ResScale)/2.0 + 1.0

	for _, f := range replayWindow {
		if !f.Hit {
			continue
		}

		if r.s.Debug.ShowCollision {
			r.drawCursorHit(dc, f.X, f.Y, shiftX, shiftY)
		}

		for i := 0; i < len(r.RenderNotes); i++ {
			rn := &r.RenderNotes[i]

			if rn.Status != StatusPending {
				continue
			}

			if f.Progress < rn.TargetTime || f.Progress > rn.TargetTime+hitWindow {
				continue
			}

			if i > 0 && r.RenderNotes[i-1].Status == StatusPending {
				if rn.BaseX == r.RenderNotes[i-1].BaseX && rn.BaseY == r.RenderNotes[i-1].BaseY {
					continue
				}
			}

			dx := math.Abs(f.X - rn.BaseX)
			dy := math.Abs(f.Y - rn.BaseY)

			if dx <= halfHitbox && dy <= halfHitbox {
				rn.Status = StatusHit
				rn.ResolvedAt = engineTime
				r.HitCount++
				break
			}
		}
	}

	for i := 0; i < len(r.RenderNotes); i++ {
		rn := &r.RenderNotes[i]
		if rn.Status == StatusPending && engineTime > rn.TargetTime+hitWindow {
			rn.Status = StatusMiss
			rn.ResolvedAt = engineTime
			r.MissCount++
		}
	}
}

func (r *Renderer) drawCursorHit(dc *gg.Context, fx, fy, shiftX, shiftY float64) {
	cursorDrawX := shiftX + fx
	cursorDrawY := shiftY + fy

	dc.SetRGBA255(0, 255, 0, 255)
	dc.DrawCircle(cursorDrawX, cursorDrawY, 8)
	dc.Stroke()
}
