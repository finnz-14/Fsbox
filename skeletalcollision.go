package main

import (
	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// setBoneMatricesIdentity ensures a model's bone matrices are identity so that
// un-animated (bind-pose) skeletal collision uses the raw bind vertices.
func setBoneMatricesIdentity(m rl.Model) {
	if m.BoneMatrices == nil || m.Skeleton.BoneCount <= 0 {
		return
	}
	mats := m.GetBoneMatrices()
	for i := range mats {
		mats[i] = rl.MatrixIdentity()
	}
}

// updateSkeletalCollisions rebuilds per-bone hulls from the current pose for
// every skeletal entity. Only entities that are animated (or freshly spawned)
// need updates; bind-pose entities keep their initial shapes.
func (g *Game) updateSkeletalCollisions() {
	for _, e := range g.entities {
		if e == nil || !e.HasModel || len(e.Skeletal) == 0 || len(e.SkeletalShapes) != len(e.Skeletal) {
			continue
		}
		if e.Body.IsNull() {
			continue
		}

		// Populate bone matrices for the current pose.
		if len(e.Animations) > 0 && e.AnimActive {
			idx := int(e.AnimCurrent)
			if idx >= 0 && idx < len(e.Animations) {
				anim := e.Animations[idx]
				if anim.KeyframeCount > 0 {
					rl.UpdateModelAnimation(e.Model, anim, e.AnimFrame)
				} else {
					setBoneMatricesIdentity(e.Model)
				}
			} else {
				setBoneMatricesIdentity(e.Model)
			}
		} else {
			setBoneMatricesIdentity(e.Model)
		}

		// Only re-cook when the pose actually matters (animated or just spawned).
		if !(e.AnimPlaying || e.AnimActive) {
			// Still ensure bind-pose shapes exist (they were created at spawn).
			continue
		}

		boneMats := []rl.Matrix{}
		if e.Model.BoneMatrices != nil && e.Model.Skeleton.BoneCount > 0 {
			boneMats = e.Model.GetBoneMatrices()
		}
		modelTransform := e.Model.Transform

		for i, bh := range e.Skeletal {
			shape := e.SkeletalShapes[i]
			if shape.IsNull() {
				continue
			}
			pts := make([]box3d.Vec3, len(bh.Points))
			var bm rl.Matrix = rl.MatrixIdentity()
			if int(bh.Bone) >= 0 && int(bh.Bone) < len(boneMats) {
				bm = boneMats[bh.Bone]
			}
			for j, v := range bh.Points {
				bv := rl.Vector3Transform(rl.Vector3{X: v.X, Y: v.Y, Z: v.Z}, bm)
				mv := rl.Vector3Transform(bv, modelTransform)
				pts[j] = box3d.Vec3{X: mv.X, Y: mv.Y, Z: mv.Z}
			}
			var hull *box3d.HullData
			func() {
				defer func() {
					if r := recover(); r != nil {
						hull = nil
					}
				}()
				hull = box3d.CreateHull(pts, len(pts), 40)
			}()
			if hull == nil {
				hull = createLeafHull(pts, 40)
			}
			if hull != nil {
				shape.SetHull(hull)
			}
		}
		// Re-apply motion locks after SetHull (which may reset body flags).
		locks := e.Body.GetMotionLocks()
		if locks.AngularX || locks.AngularY || locks.AngularZ {
			e.Body.SetMotionLocks(locks)
		}
	}
}

// updateSkeletalCollisionsForEntity updates a single entity (used at spawn).
func (g *Game) updateSkeletalCollisionsForEntity(e *Entity) {
	if e == nil || !e.HasModel || len(e.Skeletal) == 0 {
		return
	}
	setBoneMatricesIdentity(e.Model)
}
