package main

import (
	"math"
	"testing"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func boxPts(cx, cy, cz, hx, hy, hz float32) []box3d.Vec3 {
	var pts []box3d.Vec3
	for _, sx := range []float32{-1, 1} {
		for _, sy := range []float32{-1, 1} {
			for _, sz := range []float32{-1, 1} {
				pts = append(pts, box3d.Vec3{X: cx + sx*hx, Y: cy + sy*hy, Z: cz + sz*hz})
			}
		}
	}
	return pts
}

func syntheticSkeleton(jitter float32) []BoneHull {
	return []BoneHull{
		{Bone: 0, Points: boxPts(0, -0.55+jitter, 0, 0.16, 0.45, 0.12)},
		{Bone: 1, Points: boxPts(0, 0.25, jitter, 0.22, 0.35, 0.14)},
		{Bone: 2, Points: boxPts(0, 0.85, 0, 0.12, 0.14, 0.12)},
	}
}

// TestSkeletalMotionLocks isolates whether angular motion locks survive the
// per-frame ShapeID.SetHull re-cooking used by skeletal collision.
func TestSkeletalMotionLocks(t *testing.T) {
	// Run from an empty directory so content discovery falls back to the
	// embedded presets and never tries to load a GLB without a GPU context.
	t.Chdir(t.TempDir())

	cases := []struct {
		name    string
		gravity float32
		jitter  bool
		animate bool
	}{
		{"gravity0_static", 0, false, false},
		{"gravity0_animated", 0, true, true},
		{"gravity1_static", 1, false, false},
		{"gravity1_animated", 1, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newHeadlessGame(t)
			e := g.addSkeletalEntity(false, bp(0, 2, 0), box3d.QuatIdentity,
				bv(0.25, 0.9, 0.15), colWhite(), "TestJenny", 150, 0.6, 0.03,
				syntheticSkeleton(0), nil, rl.Model{Transform: rl.MatrixIdentity()}, true)
			if e == nil {
				t.Fatal("spawn failed")
			}
			e.Body.SetGravityScale(tc.gravity)
			e.Body.SetMotionLocks(box3d.MotionLocks{AngularX: true, AngularY: true, AngularZ: true})
			e.AnimActive = tc.animate

			var maxTilt float64
			for i := 0; i < 600; i++ {
				if tc.jitter && i%30 == 0 {
					// Simulate an animated pose by re-cooking hulls with a
					// slightly different shape, exactly like the real path.
					f := float32(i%60) / 60 * 0.05
					pts := syntheticSkeleton(f)
					for k, bh := range pts {
						var hull *box3d.HullData
						hull = box3d.CreateHull(bh.Points, len(bh.Points), 40)
						if hull != nil && k < len(e.SkeletalShapes) {
							e.SkeletalShapes[k].SetHull(hull)
						}
					}
				}
				g.tick(1.0 / 120.0)
				g.updateSkeletalCollisions()

				up := upOf(e.Body.GetRotation())
				tilt := float64(math.Acos(float64(clampf(up.Y, -1, 1)))) * 180 / math.Pi
				if tilt > maxTilt {
					maxTilt = tilt
				}
				if i%120 == 0 {
					p := e.Body.GetPosition()
					q := e.Body.GetRotation()
					locks := e.Body.GetMotionLocks()
					t.Logf("f=%d tilt=%.2f pos=(%.2f %.2f %.2f) q=(%.3f %.3f %.3f %.3f) angX=%v angY=%v angZ=%v",
						i, tilt, p.X, p.Y, p.Z, q.V.X, q.V.Y, q.V.Z, q.S,
						locks.AngularX, locks.AngularY, locks.AngularZ)
				}
			}
			t.Logf("max tilt: %.2f deg", maxTilt)
			if maxTilt > 10 {
				t.Fatalf("body tilted %.1f deg despite angular locks", maxTilt)
			}
		})
	}
}

// TestSkeletalPrimitiveColliders verifies the fast path: bind-pose hulls are
// fitted to capsule/sphere proxies, those proxies are attached to the body,
// and the body stays upright while they are moved by the animation pose.
func TestSkeletalPrimitiveColliders(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	hulls := syntheticSkeleton(0)
	colliders := buildBoneColliders(hulls)
	if len(colliders) == 0 {
		t.Fatal("buildBoneColliders returned no colliders")
	}
	e := g.addSkeletalEntity(false, bp(0, 2, 0), box3d.QuatIdentity,
		bv(0.25, 0.9, 0.15), colWhite(), "TestPrimitives", 150, 0.6, 0.03,
		hulls, colliders, rl.Model{Transform: rl.MatrixIdentity()}, true)
	if e == nil {
		t.Fatal("spawn failed")
	}
	if len(e.SkeletalColliderShapes) == 0 {
		t.Fatal("no primitive shapes were created")
	}
	if len(e.SkeletalColliderShapes) != len(e.SkeletalColliders) {
		t.Fatalf("collider/shape mismatch: %d/%d", len(e.SkeletalColliders), len(e.SkeletalColliderShapes))
	}
	if len(e.SkeletalShapes) != 0 {
		t.Fatalf("primitive path should not create hull shapes, got %d", len(e.SkeletalShapes))
	}

	e.Body.SetGravityScale(1)
	e.Body.SetMotionLocks(box3d.MotionLocks{AngularX: true, AngularY: true, AngularZ: true})
	e.AnimActive = true

	var maxTilt float64
	for i := 0; i < 600; i++ {
		g.tick(1.0 / 120.0)
		g.updateSkeletalCollisions()
		up := upOf(e.Body.GetRotation())
		tilt := float64(math.Acos(float64(clampf(up.Y, -1, 1)))) * 180 / math.Pi
		if tilt > maxTilt {
			maxTilt = tilt
		}
	}
	t.Logf("max tilt: %.2f deg, colliders=%d", maxTilt, len(e.SkeletalColliders))
	if maxTilt > 10 {
		t.Fatalf("body tilted %.1f deg despite angular locks", maxTilt)
	}
}

func upOf(q box3d.Quat) box3d.Vec3 {
	x, y, z, w := q.V.X, q.V.Y, q.V.Z, q.S
	return box3d.Vec3{
		X: 2 * (x*y - w*z),
		Y: 1 - 2*(x*x+z*z),
		Z: 2 * (y*z + w*x),
	}
}
