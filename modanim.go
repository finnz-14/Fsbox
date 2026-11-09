package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// updateAnimations advances the frame of every entity that is currently
// playing an animation. This is required by raylib to actually render the
// animated pose. Which animation plays, at what speed, and whether it plays
// at all is controlled entirely by mods via the entity.play_animation /
// entity.pause_animation / entity.stop_animation / entity.set_animation_speed
// commands. The engine does not show any GUI or decide what to play.
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
		e.AnimFrame += e.AnimSpeed * 30.0 * dt
		if e.AnimFrame >= float32(anim.KeyframeCount) {
			e.AnimFrame = 0
		}
		rl.UpdateModelAnimation(e.Model, anim, e.AnimFrame)
	}
}
