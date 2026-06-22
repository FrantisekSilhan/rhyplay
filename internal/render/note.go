package render

import (
	"math"
	"rhyplay/internal/config"
	"rhyplay/internal/game"

	"github.com/fogleman/gg"
)

func (r *Renderer) SetupNotes(dc *gg.Context, engineTime float64, noteWindowIdx int, shiftX, shiftY float64) {
	for i := len(r.RenderNotes) - 1; i >= noteWindowIdx; i-- {
		rn := r.RenderNotes[i]

		switch rn.Status {
		case StatusPending:
			timeDiff := rn.TargetTime - engineTime
			r.SetupNote(dc, rn, engineTime, timeDiff, r.nc.depthStep, r.s.Gameplay.ApproachDistance, shiftX, shiftY)
		case StatusMiss:
			if r.s.Visuals.Miss.Enabled && engineTime < rn.ResolvedAt+game.MissDuration {
				r.DrawMiss(dc, rn, engineTime, shiftX, shiftY, (i&1)*2-1)
			}
		}
	}
}

func (r *Renderer) SetupNote(dc *gg.Context, rn RenderNote, engineTime, timeDiff, depthStep, ad, shiftX, shiftY float64) {
	v := r.s.Visuals

	depth := timeDiff * depthStep

	if depth > ad || depth < 0 {
		return
	}

	perspective := game.CalcPerspective(depth)
	progress := 1.0 - (depth / ad)

	drawX := shiftX + (rn.BaseX * perspective)
	drawY := shiftY + (rn.BaseY * perspective)

	alpha := r.calculateNoteAlpha(progress, timeDiff, v.Modifiers)

	if alpha > 0 {
		noteSize := (game.NoteDrawSize - v.Note.Shape.LineWidth) * r.ResScale * perspective
		r.DrawNote(dc, alpha, rn.NoteIdx, drawX, drawY, noteSize, perspective)
		if v.Note.ShowHitbox {
			hitboxSize := r.c.HitboxSize * r.ResScale * perspective
			r.DrawHitbox(dc, drawX-hitboxSize/2, drawY-hitboxSize/2, hitboxSize)
		}
	}
}

func (r *Renderer) calculateNoteAlpha(progress, timeDiff float64, modifiers config.Modifiers) float64 {
	fadeIn := game.FadeIn / 100.0
	alpha := 1.0
	if progress < fadeIn {
		alpha = progress / fadeIn
	}

	if modifiers.Ghost {
		startFade := 0.25
		endFade := 0.9
		if progress > startFade {
			ratio := (progress - startFade) / (endFade - startFade)

			if ratio > 1.0 {
				ratio = 1.0
			}

			alpha -= ratio
		}
	} else if modifiers.FadeOut {
		fadeOut := game.FadeOut / 100.0
		alpha -= 1 - math.Min(1, (1-progress)/fadeOut)
		if alpha < game.MinFadeOut {
			alpha = game.MinFadeOut
		}
	}

	if !modifiers.Pushback && timeDiff < 0 {
		alpha = 0
	}

	return alpha
}
