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
	hitWindow := game.GetEffectiveHitWindow(r.Replay.ScoreData.Speed)

	for _, f := range replayWindow {
		for i := r.Score.NextPendingNoteIdx; i < len(r.RenderNotes); i++ {
			rn := &r.RenderNotes[i]

			if rn.Status != StatusPending {
				if i == r.Score.NextPendingNoteIdx {
					r.Score.NextPendingNoteIdx++
				}
				continue
			}

			if f.Progress < rn.TargetTime || f.Progress > rn.TargetTime+hitWindow {
				if f.Progress > rn.TargetTime+hitWindow {
					rn.Status = StatusMiss
					rn.ResolvedAt = rn.TargetTime + hitWindow
					r.Score.MissCount++
					r.Score.Combo = 0
				}
				continue
			}

			if !f.Hit {
				continue
			}

			if r.s.Debug.ShowCollision {
				r.drawCursorHit(dc, f.X, f.Y, shiftX, shiftY)
			}

			dx := math.Abs(f.RawX - rn.RawX)
			dy := math.Abs(f.RawY - rn.RawY)

			if dx <= r.c.HitThreshold && dy <= r.c.HitThreshold {
				rn.Status = StatusHit
				rn.ResolvedAt = f.Progress
				r.Score.HitCount++
				r.Score.Combo++
				r.Score.Score += r.Score.Combo * 100
				break
			}
		}
	}

	for i := r.Score.NextPendingNoteIdx; i < len(r.RenderNotes); i++ {
		rn := &r.RenderNotes[i]
		if rn.Status == StatusPending && engineTime > rn.TargetTime+hitWindow {
			rn.Status = StatusMiss
			rn.ResolvedAt = rn.TargetTime + hitWindow
			r.Score.MissCount++
			r.Score.Combo = 0
		} else if rn.Status == StatusPending {
			break
		} else if i == r.Score.NextPendingNoteIdx {
			r.Score.NextPendingNoteIdx++
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

func (r *Renderer) calculateRP() int {
	processed := r.Score.HitCount + r.Score.MissCount
	if processed == 0 {
		return 0
	}

	accuracy := float64(r.Score.HitCount) / float64(processed)

	ease := 0.0
	if accuracy > 0 {
		ease = math.Pow(2.0, r.Score.EaseBase*accuracy-r.Score.EaseBase)
	}

	starMultiplier := r.Score.StarMultiplier

	if false { // No fail penalty
		starMultiplier *= math.Pow(0.95, float64(r.Score.MissCount))
	}

	val := (starMultiplier * ease * 100.0 / 2.0)
	rp := math.Round(math.Pow(val, 2.0)/1000.0) * 2.0

	return int(rp)
}
