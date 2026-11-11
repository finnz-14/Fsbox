package main

import (
	"math"

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

// buildBoneColliders fits one cheap sphere/capsule proxy to each bind-pose
// bone hull. The fit runs once when content is loaded; runtime only moves the
// proxy, which avoids re-cooking convex hulls every animation frame.
func buildBoneColliders(hulls []BoneHull) []BoneCollider {
	out := make([]BoneCollider, 0, len(hulls))
	for _, bh := range hulls {
		c, ok := fitBoneCollider(bh.Points)
		if !ok {
			continue
		}
		c.Bone = bh.Bone
		out = append(out, c)
	}
	return out
}

// fitBoneCollider returns a capsule when the point cloud is elongated and a
// sphere when it is compact. Points are in raw model space.
func fitBoneCollider(pts []box3d.Vec3) (BoneCollider, bool) {
	if len(pts) < 4 {
		return BoneCollider{}, false
	}
	lower, upper := hullBounds(pts)
	minDim := minf(upper.X-lower.X, minf(upper.Y-lower.Y, upper.Z-lower.Z))
	if minDim < 0.006 {
		return BoneCollider{}, false
	}

	var centroid box3d.Vec3
	for _, p := range pts {
		centroid.X += p.X
		centroid.Y += p.Y
		centroid.Z += p.Z
	}
	inv := 1 / float32(len(pts))
	centroid.X *= inv
	centroid.Y *= inv
	centroid.Z *= inv

	axis := principalAxis(pts, centroid)
	minT := float32(math.MaxFloat32)
	maxT := -float32(math.MaxFloat32)
	for _, p := range pts {
		t := vecDot(vecSub(p, centroid), axis)
		if t < minT {
			minT = t
		}
		if t > maxT {
			maxT = t
		}
	}

	a := vecAdd(centroid, vecMul(axis, minT))
	b := vecAdd(centroid, vecMul(axis, maxT))
	radius := float32(0)
	for _, p := range pts {
		d := pointSegmentDistance(p, a, b)
		if d > radius {
			radius = d
		}
	}
	halfLen := (maxT - minT) * 0.5

	// Compact blobs are better represented by a sphere. This also avoids
	// degenerate capsules for heads, hands and feet.
	if halfLen <= radius*0.75 {
		sphereR := float32(0)
		for _, p := range pts {
			d := vecLen(vecSub(p, centroid))
			if d > sphereR {
				sphereR = d
			}
		}
		if sphereR < 0.01 {
			return BoneCollider{}, false
		}
		return BoneCollider{
			Sphere: true,
			Center: centroid,
			Radius: sphereR + 0.005,
		}, true
	}

	if radius < 0.008 {
		radius = 0.008
	}
	return BoneCollider{A: a, B: b, Radius: radius + 0.005}, true
}

// principalAxis returns the dominant eigenvector of the point cloud's
// covariance matrix, found with a short power iteration.
func principalAxis(pts []box3d.Vec3, centroid box3d.Vec3) box3d.Vec3 {
	var xx, xy, xz, yy, yz, zz float64
	for _, p := range pts {
		dx := float64(p.X - centroid.X)
		dy := float64(p.Y - centroid.Y)
		dz := float64(p.Z - centroid.Z)
		xx += dx * dx
		xy += dx * dy
		xz += dx * dz
		yy += dy * dy
		yz += dy * dz
		zz += dz * dz
	}

	// Seed with the longest AABB axis so flat clouds still converge quickly.
	lower, upper := hullBounds(pts)
	ext := box3d.Vec3{X: upper.X - lower.X, Y: upper.Y - lower.Y, Z: upper.Z - lower.Z}
	v := box3d.Vec3{X: 1}
	if ext.Y > ext.X && ext.Y >= ext.Z {
		v = box3d.Vec3{Y: 1}
	} else if ext.Z > ext.X && ext.Z > ext.Y {
		v = box3d.Vec3{Z: 1}
	}

	for i := 0; i < 12; i++ {
		nx := xx*float64(v.X) + xy*float64(v.Y) + xz*float64(v.Z)
		ny := xy*float64(v.X) + yy*float64(v.Y) + yz*float64(v.Z)
		nz := xz*float64(v.X) + yz*float64(v.Y) + zz*float64(v.Z)
		n := math.Sqrt(nx*nx + ny*ny + nz*nz)
		if n < 1e-12 {
			break
		}
		v = box3d.Vec3{X: float32(nx / n), Y: float32(ny / n), Z: float32(nz / n)}
	}
	if vecLen(v) < 1e-6 {
		return box3d.Vec3{Y: 1}
	}
	return v
}

func vecAdd(a, b box3d.Vec3) box3d.Vec3 {
	return box3d.Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func vecSub(a, b box3d.Vec3) box3d.Vec3 {
	return box3d.Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func vecMul(v box3d.Vec3, s float32) box3d.Vec3 {
	return box3d.Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func vecDot(a, b box3d.Vec3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func vecLen(v box3d.Vec3) float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func pointSegmentDistance(p, a, b box3d.Vec3) float32 {
	ab := vecSub(b, a)
	denom := vecDot(ab, ab)
	if denom < 1e-12 {
		return vecLen(vecSub(p, a))
	}
	t := vecDot(vecSub(p, a), ab) / denom
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return vecLen(vecSub(p, vecAdd(a, vecMul(ab, t))))
}

// transformBonePoint applies the bone matrix and then the model transform,
// matching the old hull pipeline (bone-local -> model space -> body space).
func transformBonePoint(v box3d.Vec3, bone, model rl.Matrix) box3d.Vec3 {
	p := rl.Vector3Transform(rl.Vector3{X: v.X, Y: v.Y, Z: v.Z}, bone)
	p = rl.Vector3Transform(p, model)
	return box3d.Vec3{X: p.X, Y: p.Y, Z: p.Z}
}

// matrixMaxScale returns the largest axis scale encoded in a raylib matrix.
func matrixMaxScale(m rl.Matrix) float32 {
	sx := float32(math.Sqrt(float64(m.M0*m.M0 + m.M1*m.M1 + m.M2*m.M2)))
	sy := float32(math.Sqrt(float64(m.M4*m.M4 + m.M5*m.M5 + m.M6*m.M6)))
	sz := float32(math.Sqrt(float64(m.M8*m.M8 + m.M9*m.M9 + m.M10*m.M10)))
	return maxf(sx, maxf(sy, sz))
}

// createSkeletalColliderShapes attaches primitive shapes for one entity. The
// returned collider slice is entity-owned so runtime caches never leak back
// into the shared spawn preset.
func createSkeletalColliderShapes(body box3d.BodyID, sd *box3d.ShapeDef, colliders []BoneCollider, model rl.Model) ([]BoneCollider, []box3d.ShapeID) {
	if len(colliders) == 0 {
		return nil, nil
	}
	modelScale := matrixMaxScale(model.Transform)
	used := make([]BoneCollider, 0, len(colliders))
	shapes := make([]box3d.ShapeID, 0, len(colliders))
	for _, c := range colliders {
		c.lastA = box3d.Vec3{}
		c.lastB = box3d.Vec3{}
		c.lastC = box3d.Vec3{}
		c.lastR = 0
		c.hasLast = false

		var sid box3d.ShapeID
		if c.Sphere {
			center := transformBonePoint(c.Center, rl.MatrixIdentity(), model.Transform)
			sphere := box3d.Sphere{Center: center, Radius: c.Radius * modelScale}
			sid = body.CreateSphereShape(sd, &sphere)
		} else {
			a := transformBonePoint(c.A, rl.MatrixIdentity(), model.Transform)
			b := transformBonePoint(c.B, rl.MatrixIdentity(), model.Transform)
			capsule := box3d.Capsule{Center1: a, Center2: b, Radius: c.Radius * modelScale}
			sid = body.CreateCapsuleShape(sd, &capsule)
		}
		if sid.IsNull() {
			continue
		}
		used = append(used, c)
		shapes = append(shapes, sid)
	}
	return used, shapes
}

// updateSkeletalCollisions updates every skeletal entity. The game normally
// calls the per-entity variant from the draw path so the pose is evaluated
// once, but this helper is kept for headless tests and tools.
func (g *Game) updateSkeletalCollisions() {
	for _, e := range g.entities {
		if e == nil || !e.HasModel {
			continue
		}
		if len(e.SkeletalColliders) == 0 && len(e.Skeletal) == 0 {
			continue
		}
		applyEntityPose(e)
		g.updateSkeletalCollisionForEntity(e, true)
	}
}

// updateSkeletalCollisionForEntity moves one entity's collision proxies to the
// pose that was just applied to its model. Primitive colliders are preferred;
// hull re-cooking remains as a fallback for models that cannot be fitted.
func (g *Game) updateSkeletalCollisionForEntity(e *Entity, force bool) {
	if e == nil || !e.HasModel || e.Body.IsNull() {
		return
	}
	if len(e.SkeletalColliders) > 0 && len(e.SkeletalColliderShapes) == len(e.SkeletalColliders) {
		g.updatePrimitiveColliders(e, force)
		return
	}
	g.updateHullColliders(e, force)
}

func (g *Game) updatePrimitiveColliders(e *Entity, force bool) {
	if !force {
		if !e.AnimActive {
			return
		}
		if !g.skeletalUpdateDue(e) {
			return
		}
	}

	var boneMats []rl.Matrix
	if e.Model.BoneMatrices != nil && e.Model.Skeleton.BoneCount > 0 {
		boneMats = e.Model.GetBoneMatrices()
	}
	modelTransform := e.Model.Transform

	for i := range e.SkeletalColliders {
		c := &e.SkeletalColliders[i]
		shape := e.SkeletalColliderShapes[i]
		if shape.IsNull() {
			continue
		}

		bone := rl.MatrixIdentity()
		if int(c.Bone) >= 0 && int(c.Bone) < len(boneMats) {
			bone = boneMats[c.Bone]
		}
		scale := matrixMaxScale(bone) * matrixMaxScale(modelTransform)

		if c.Sphere {
			center := transformBonePoint(c.Center, bone, modelTransform)
			radius := c.Radius * scale
			if c.hasLast && nearVec3(c.lastC, center) && absf(c.lastR-radius) < 0.002 {
				continue
			}
			shape.SetSphere(&box3d.Sphere{Center: center, Radius: radius})
			c.lastC = center
			c.lastR = radius
			c.hasLast = true
			continue
		}

		a := transformBonePoint(c.A, bone, modelTransform)
		b := transformBonePoint(c.B, bone, modelTransform)
		radius := c.Radius * scale
		if c.hasLast && nearVec3(c.lastA, a) && nearVec3(c.lastB, b) && absf(c.lastR-radius) < 0.002 {
			continue
		}
		shape.SetCapsule(&box3d.Capsule{Center1: a, Center2: b, Radius: radius})
		c.lastA = a
		c.lastB = b
		c.lastR = radius
		c.hasLast = true
	}

	reapplyMotionLocks(e)
}

func (g *Game) updateHullColliders(e *Entity, force bool) {
	if len(e.Skeletal) == 0 || len(e.SkeletalShapes) != len(e.Skeletal) {
		return
	}
	if !force {
		if !e.AnimActive {
			return
		}
		if !g.skeletalUpdateDue(e) {
			return
		}
	}

	var boneMats []rl.Matrix
	if e.Model.BoneMatrices != nil && e.Model.Skeleton.BoneCount > 0 {
		boneMats = e.Model.GetBoneMatrices()
	}
	modelTransform := e.Model.Transform

	for i, bh := range e.Skeletal {
		shape := e.SkeletalShapes[i]
		if shape.IsNull() {
			continue
		}
		if cap(e.skeletalScratch) < len(bh.Points) {
			e.skeletalScratch = make([]box3d.Vec3, len(bh.Points))
		}
		pts := e.skeletalScratch[:len(bh.Points)]

		bone := rl.MatrixIdentity()
		if int(bh.Bone) >= 0 && int(bh.Bone) < len(boneMats) {
			bone = boneMats[bh.Bone]
		}
		for j, v := range bh.Points {
			pts[j] = transformBonePoint(v, bone, modelTransform)
		}
		hull := createLeafHull(pts, 40)
		if hull != nil {
			shape.SetHull(hull)
		}
	}

	reapplyMotionLocks(e)
}

// skeletalUpdateDue throttles collision shape updates by distance. Distant
// entities update less often, which keeps broadphase churn bounded when many
// animated models are on screen.
func (g *Game) skeletalUpdateDue(e *Entity) bool {
	interval := 1
	p := e.Body.GetPosition()
	dx := p.X - g.camera.Position.X
	dy := p.Y - g.camera.Position.Y
	dz := p.Z - g.camera.Position.Z
	d2 := dx*dx + dy*dy + dz*dz
	switch {
	case d2 < 18*18:
		interval = 1
	case d2 < 40*40:
		interval = 2
	case d2 < 70*70:
		interval = 4
	default:
		interval = 8
	}
	e.SkeletalTick++
	if interval <= 1 {
		return true
	}
	return (e.SkeletalTick+int(e.ID))%interval == 0
}

func reapplyMotionLocks(e *Entity) {
	locks := e.Body.GetMotionLocks()
	if locks.AngularX || locks.AngularY || locks.AngularZ {
		e.Body.SetMotionLocks(locks)
	}
}

func nearVec3(a, b box3d.Vec3) bool {
	const eps = 0.002
	return absf(a.X-b.X) < eps && absf(a.Y-b.Y) < eps && absf(a.Z-b.Z) < eps
}
