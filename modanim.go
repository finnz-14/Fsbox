package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// animProximityDist is how close the player must be to an animated entity
// before its animation GUI opens.
const animProximityDist = 3.5

// updateAnimationProximity checks all animated entities and sets
// g.animGUIEntity to the closest one that the player is both near AND aiming
// at (or -1 if none). This prevents the GUI from popping up when just walking
// past an animated entity.
func (g *Game) updateAnimationProximity() {
	if g.player == nil {
		g.animGUIEntity = -1
		return
	}
	p := g.player.body.GetPosition()
	fwd := g.cameraForward()

	bestID := int32(-1)
	bestDist := float32(animProximityDist)

	for _, e := range g.entities {
		if !e.HasModel || len(e.Animations) == 0 {
			continue
		}
		ep := e.position()
		dx := ep.X - p.X
		dy := ep.Y - p.Y
		dz := ep.Z - p.Z
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if dist >= bestDist {
			continue
		}
		// Only show GUI if the entity is roughly in front of the camera
		// (dot product of forward vector and direction to entity).
		toEnt := rlv(ep.X-p.X, ep.Y-p.Y, ep.Z-p.Z)
		toEnt = normRl(toEnt)
		dot := fwd.X*toEnt.X + fwd.Y*toEnt.Y + fwd.Z*toEnt.Z
		if dot < 0.5 {
			continue
		}
		bestDist = dist
		bestID = e.ID
	}
	g.animGUIEntity = bestID
}

// updateAnimations advances the frame of the currently-playing animation on
// the entity shown in the animation GUI (or any entity that is playing).
func (g *Game) updateAnimations(dt float32) {
	if dt <= 0 {
		return
	}
	for _, e := range g.entities {
		if !e.HasModel || len(e.Animations) == 0 || !e.AnimPlaying {
			continue
		}
		if int(e.AnimCurrent) < 0 || int(e.AnimCurrent) >= len(e.Animations) {
			continue
		}
		anim := e.Animations[e.AnimCurrent]
		// Advance frames assuming ~30 FPS animation source; AnimSpeed is a
		// multiplier (1.0 = normal speed, 2.0 = double speed, etc.).
		e.AnimFrame += e.AnimSpeed * 30.0 * dt
		if e.AnimFrame >= float32(anim.KeyframeCount) {
			e.AnimFrame = 0
		}
		rl.UpdateModelAnimation(e.Model, anim, e.AnimFrame)
	}
}

// drawAnimGUI draws the animation selection panel near the entity on screen.
func (g *Game) drawAnimGUI() {
	if g.animGUIEntity < 0 {
		return
	}
	e := g.resolveAnyEntity(g.animGUIEntity)
	if e == nil || !e.HasModel || len(e.Animations) == 0 {
		return
	}

	// World-to-screen position of the entity.
	screenPos := rl.GetWorldToScreen(e.position(), g.camera)

	panelW := float32(300)
	panelH := float32(340)
	rec := rl.Rectangle{
		X:      screenPos.X - panelW/2,
		Y:      screenPos.Y - panelH - 30,
		Width:  panelW,
		Height: panelH,
	}

	// Clamp to screen bounds.
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	if rec.X < 0 {
		rec.X = 0
	}
	if rec.Y < 0 {
		rec.Y = 0
	}
	if rec.X+panelW > sw {
		rec.X = sw - panelW
	}
	if rec.Y+panelH > sh {
		rec.Y = sh - panelH
	}

	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.textCenteredSpaced("ANIMATIONS", rec.X+rec.Width/2, rec.Y+22, 18, 2.5, colText)

	// Entity name.
	g.textCenteredSpaced(e.Name, rec.X+rec.Width/2, rec.Y+44, 12, 1.5, colMuted)

	// Close button.
	closeBtn := rl.Rectangle{X: rec.X + rec.Width - 26, Y: rec.Y + 10, Width: 18, Height: 18}
	g.drawMenuButton(closeBtn, "X", true)
	if rl.CheckCollisionPointRec(g.uiMouse(), closeBtn) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		g.animGUIEntity = -1
		return
	}

	// Animation list.
	startY := rec.Y + 72
	rowH := float32(26)
	mouse := g.uiMouse()

	for i, anim := range e.Animations {
		name := anim.GetName()
		if name == "" {
			name = fmt.Sprintf("anim_%d", i)
		}

		row := rl.Rectangle{X: rec.X + 12, Y: startY + float32(i)*rowH, Width: rec.Width - 24, Height: rowH - 3}
		hover := rl.CheckCollisionPointRec(mouse, row)

		fill := colPanelSoft
		if hover {
			fill = colPanelHover
		}
		if int32(i) == e.AnimCurrent {
			fill = colRowSelected
		}

		rl.DrawRectangleRec(row, fill)
		rl.DrawRectangleLinesEx(row, 1, colBorderHi)

		label := name
		if int32(i) == e.AnimCurrent && e.AnimPlaying {
			label = "\u25B6 " + name // play triangle
		} else if int32(i) == e.AnimCurrent && !e.AnimPlaying {
			label = "\u23F8 " + name // pause
		}

		g.drawText(label, row.X+8, row.Y+5, 11, colText)

		// Frame info.
		frameTxt := fmt.Sprintf("%.0f/%.0f", e.AnimFrame, float32(anim.KeyframeCount-1))
		fw := g.textWidth(frameTxt, 10)
		g.drawText(frameTxt, row.X+row.Width-fw-8, row.Y+5, 10, colMuted)

		if hover && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			g.playEntityAnim(e, int32(i))
		}
	}

	// Play / Pause / Stop row.
	btnY := startY + float32(len(e.Animations))*rowH + 14
	halfW := (rec.Width - 24 - 6) / 2

	playBtn := rl.Rectangle{X: rec.X + 12, Y: btnY, Width: halfW, Height: 26}
	pauseBtn := rl.Rectangle{X: rec.X + 12 + halfW + 6, Y: btnY, Width: halfW, Height: 26}

	if e.AnimPlaying {
		g.drawMenuButton(playBtn, "PAUSE", false)
	} else {
		g.drawMenuButton(playBtn, "PLAY", false)
	}
	g.drawMenuButton(pauseBtn, "STOP", true)

	if rl.CheckCollisionPointRec(mouse, playBtn) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		if len(e.Animations) > 0 {
			e.AnimPlaying = !e.AnimPlaying
			if e.AnimPlaying && e.AnimFrame >= float32(e.Animations[e.AnimCurrent].KeyframeCount-1) {
				e.AnimFrame = 0
			}
		}
	}
	if rl.CheckCollisionPointRec(mouse, pauseBtn) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		e.AnimPlaying = false
		e.AnimFrame = 0
	}

	// Speed slider.
	slY := btnY + 34
	slRec := rl.Rectangle{X: rec.X + 12, Y: slY, Width: rec.Width - 24, Height: 4}
	rl.DrawRectangleRec(slRec, colBorderHi)

	knobX := rec.X + 12 + (e.AnimSpeed-0.25)/(2.0-0.25)*(rec.Width-24)
	if knobX < rec.X+12 {
		knobX = rec.X + 12
	}
	if knobX > rec.X+rec.Width-12 {
		knobX = rec.X + rec.Width - 12
	}
	rl.DrawCircleV(rl2(knobX, slY+2), 5, colAccent)

	speedTxt := fmt.Sprintf("speed: %.2fx", e.AnimSpeed)
	g.drawText(speedTxt, rec.X+12, slY-14, 11, colMuted)

	// Speed adjustment via click on track.
	if rl.CheckCollisionPointRec(mouse, slRec) && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		t := (mouse.X - (rec.X + 12)) / (rec.Width - 24)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		e.AnimSpeed = 0.25 + t*(2.0-0.25)
	}
}

// playEntityAnim selects and plays an animation on an entity.
func (g *Game) playEntityAnim(e *Entity, animIndex int32) {
	if e == nil || animIndex < 0 || int(animIndex) >= len(e.Animations) {
		return
	}
	e.AnimCurrent = animIndex
	e.AnimFrame = 0
	e.AnimPlaying = true
	e.AnimSpeed = 1
}
