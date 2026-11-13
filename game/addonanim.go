package game

import (
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// advanceAnimations advances the frame counter of every entity that is
// currently playing an animation. The pose itself is applied immediately
// before that entity is drawn (applyEntityPose), because raylib's CPU skinning
// writes the animated vertices into the model's shared vertex buffers. If all
// poses were applied in one update pass, every instance of the same model
// would end up showing only the last entity's pose.
func (g *Game) advanceAnimations(dt float32) {
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
		if anim.KeyframeCount <= 0 {
			continue
		}
		e.AnimFrame += e.AnimSpeed * 30.0 * dt
		if e.AnimFrame >= float32(anim.KeyframeCount) {
			e.AnimFrame = 0
		}
	}
}

// applyEntityPose uploads this entity's current pose to its model. It must be
// called immediately before drawing the entity so that two instances of the
// same model (which share mesh data and vertex buffers) do not overwrite each
// other's animation.
func applyEntityPose(e *Entity) {
	if e == nil || !e.HasModel || len(e.Animations) == 0 {
		return
	}
	if e.AnimActive {
		idx := int(e.AnimCurrent)
		if idx >= 0 && idx < len(e.Animations) {
			anim := e.Animations[idx]
			if anim.KeyframeCount > 0 {
				rl.UpdateModelAnimation(e.Model, anim, e.AnimFrame)
				return
			}
		}
	}
	resetModelBindPose(e.Model)
}

// copyEntityAnimation carries animation state over to a copied entity so the
// copy keeps its own playback state instead of inheriting the source's
// instance identity or sharing its frame counter.
func copyEntityAnimation(dst, src *Entity) {
	if dst == nil || src == nil || len(src.Animations) == 0 {
		return
	}
	dst.Animations = src.Animations
	dst.AnimCurrent = src.AnimCurrent
	dst.AnimFrame = src.AnimFrame
	dst.AnimPlaying = src.AnimPlaying
	dst.AnimActive = src.AnimActive
	dst.AnimSpeed = src.AnimSpeed
}

// resetModelBindPose restores a model's skinned vertex buffers to the bind
// pose. Preview viewports draw the same shared model as the entities, so
// without this they would show whichever entity pose was uploaded last.
func resetModelBindPose(m rl.Model) {
	if m.MeshCount <= 0 || m.Meshes == nil {
		return
	}
	meshes := m.GetMeshes()
	for i := range meshes {
		mesh := &meshes[i]
		n := int(mesh.VertexCount) * 3
		if n <= 0 || mesh.Vertices == nil || mesh.AnimVertices == nil {
			continue
		}
		animVerts := unsafe.Slice(mesh.AnimVertices, n)
		copy(animVerts, unsafe.Slice(mesh.Vertices, n))
		if mesh.VboID == nil {
			continue
		}
		vbo := unsafe.Slice(mesh.VboID, rl.ShaderLocVertexNormal+1)
		rl.UpdateVertexBuffer(vbo[rl.ShaderLocVertexPosition], animVerts, 0)
		if mesh.Normals != nil && mesh.AnimNormals != nil {
			animNormals := unsafe.Slice(mesh.AnimNormals, n)
			copy(animNormals, unsafe.Slice(mesh.Normals, n))
			rl.UpdateVertexBuffer(vbo[rl.ShaderLocVertexNormal], animNormals, 0)
		}
	}
}
