package animation

import (
	"sort"
	"unsafe"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// BuildModelHulls decomposes a model into multiple convex hulls with an
// adaptive k-d tree. A uniform grid leaves gaps wherever a limb or the head
// happens to sit between cells, and it silently drops cells with too few
// vertices. Splitting the point cloud in half along its longest axis instead
// keeps every vertex inside a leaf, so thin limbs and the head always end up
// with their own hulls.
func BuildModelHulls(model rl.Model) []*box3d.HullData {
	var verts []box3d.Vec3
	for _, mesh := range model.GetMeshes() {
		if mesh.VertexCount <= 0 || mesh.Vertices == nil {
			continue
		}
		vs := unsafe.Slice(mesh.Vertices, int(mesh.VertexCount)*3)
		for i := 0; i < int(mesh.VertexCount); i++ {
			v := rl.Vector3{X: vs[i*3], Y: vs[i*3+1], Z: vs[i*3+2]}
			t := rl.Vector3Transform(v, model.Transform)
			verts = append(verts, box3d.Vec3{X: t.X, Y: t.Y, Z: t.Z})
		}
	}
	if len(verts) < 4 {
		return nil
	}

	hulls := BuildHullsFromPoints(verts)
	if len(hulls) == 0 {
		h := BuildModelHull(model)
		if h != nil {
			hulls = append(hulls, h)
		}
	}
	return hulls
}

// BuildBoneHulls groups a skinned model's vertices by their dominant bone and
// decomposes each group into convex-hull leaves. Points are kept in raw mesh
// space (the model transform is applied later) so the bone matrix can be
// applied first. Returns nil for non-skinned models.
func BuildBoneHulls(model rl.Model) []BoneHull {
	var out []BoneHull
	for _, mesh := range model.GetMeshes() {
		if mesh.VertexCount <= 0 || mesh.Vertices == nil || mesh.BoneIndices == nil || mesh.BoneWeights == nil {
			continue
		}
		vs := unsafe.Slice(mesh.Vertices, int(mesh.VertexCount)*3)
		bi := unsafe.Slice(mesh.BoneIndices, int(mesh.VertexCount)*4)
		bw := unsafe.Slice(mesh.BoneWeights, int(mesh.VertexCount)*4)

		groups := make(map[int32][]box3d.Vec3)
		for i := 0; i < int(mesh.VertexCount); i++ {
			bestBone := int32(0)
			bestW := float32(0)
			for j := 0; j < 4; j++ {
				w := bw[i*4+j]
				if w > bestW {
					bestW = w
					bestBone = int32(bi[i*4+j])
				}
			}
			if bestW <= 0 {
				continue
			}
			pt := box3d.Vec3{X: vs[i*3], Y: vs[i*3+1], Z: vs[i*3+2]}
			groups[bestBone] = append(groups[bestBone], pt)
		}

		for bone, pts := range groups {
			if len(pts) < 4 {
				continue
			}
			lower, upper := HullBounds(pts)
			minDim := minf(upper.X-lower.X, minf(upper.Y-lower.Y, upper.Z-lower.Z))
			if minDim < 0.006 {
				continue
			}
			leaves := SplitIntoLeaves(pts)
			for _, leaf := range leaves {
				l, u := HullBounds(leaf)
				leafMin := minf(u.X-l.X, minf(u.Y-l.Y, u.Z-l.Z))
				if leafMin < 0.006 {
					continue
				}
				out = append(out, BoneHull{Bone: bone, Points: leaf})
			}
		}
	}
	return out
}

// SplitIntoLeaves splits a vertex cloud with an adaptive k-d tree and returns
// the leaf point sets. Shared by the bind-pose hull builder and the per-bone
// skeletal builder.
func SplitIntoLeaves(verts []box3d.Vec3) [][]box3d.Vec3 {
	if len(verts) < 4 {
		return nil
	}

	const (
		leafSize = 700
		maxDepth = 24
	)

	var leaves [][]box3d.Vec3
	var split func(pts []box3d.Vec3, depth int)
	split = func(pts []box3d.Vec3, depth int) {
		if len(pts) < 4 {
			return
		}
		if len(pts) <= leafSize || depth >= maxDepth {
			leaf := make([]box3d.Vec3, len(pts))
			copy(leaf, pts)
			leaves = append(leaves, leaf)
			return
		}

		lower, upper := HullBounds(pts)
		axis := HullLongestAxis(lower, upper)
		mid := (HullAxisValue(lower, axis) + HullAxisValue(upper, axis)) * 0.5
		lo, hi := HullPartition(pts, axis, mid)

		if len(lo) < 4 || len(hi) < 4 {
			sort.Slice(pts, func(i, j int) bool {
				return HullAxisValue(pts[i], axis) < HullAxisValue(pts[j], axis)
			})
			midIdx := len(pts) / 2
			lo, hi = pts[:midIdx], pts[midIdx:]
		}
		if len(lo) < 4 || len(hi) < 4 {
			return
		}

		split(lo, depth+1)
		split(hi, depth+1)
	}
	split(verts, 0)
	return leaves
}

// BuildHullsFromPoints splits a model-space vertex cloud with an adaptive k-d
// tree and turns each leaf into a convex hull.
func BuildHullsFromPoints(verts []box3d.Vec3) []*box3d.HullData {
	if len(verts) < 4 {
		return nil
	}

	const maxHullVerts = 40

	leaves := SplitIntoLeaves(verts)
	var hulls []*box3d.HullData
	for _, leaf := range leaves {
		if h := CreateLeafHull(leaf, maxHullVerts); h != nil {
			hulls = append(hulls, h)
		}
	}
	if len(hulls) == 0 {
		// fallback whole cloud
		if h := CreateLeafHull(verts, maxHullVerts); h != nil {
			hulls = append(hulls, h)
		}
	}
	return hulls
}

// CreateLeafHull turns one k-d leaf into a convex hull. Flat leaves (surface
// strips with no volume) are rejected by the hull builder, so a tiny
// deterministic jitter is applied to give them just enough thickness to
// produce a thin slab instead of a hole in the collision.
func CreateLeafHull(pts []box3d.Vec3, maxVerts int) *box3d.HullData {
	if h := box3d.CreateHull(pts, len(pts), maxVerts); h != nil {
		return h
	}
	lower, upper := HullBounds(pts)
	scale := maxf(upper.X-lower.X, maxf(upper.Y-lower.Y, upper.Z-lower.Z))
	if scale < 1e-4 {
		return nil
	}
	jitter := scale * 1e-3
	jittered := make([]box3d.Vec3, len(pts))
	for i, p := range pts {
		f := float32((i%7)-3) / 3
		jittered[i] = box3d.Vec3{
			X: p.X + jitter*f,
			Y: p.Y + jitter*f*f,
			Z: p.Z + jitter*f*f*f,
		}
	}
	return box3d.CreateHull(jittered, len(jittered), maxVerts)
}

func HullBounds(pts []box3d.Vec3) (box3d.Vec3, box3d.Vec3) {
	lower, upper := pts[0], pts[0]
	for _, p := range pts {
		lower.X = minf(lower.X, p.X)
		lower.Y = minf(lower.Y, p.Y)
		lower.Z = minf(lower.Z, p.Z)
		upper.X = maxf(upper.X, p.X)
		upper.Y = maxf(upper.Y, p.Y)
		upper.Z = maxf(upper.Z, p.Z)
	}
	return lower, upper
}

func HullLongestAxis(lower, upper box3d.Vec3) int {
	dx := upper.X - lower.X
	dy := upper.Y - lower.Y
	dz := upper.Z - lower.Z
	if dx >= dy && dx >= dz {
		return 0
	}
	if dy >= dz {
		return 1
	}
	return 2
}

func HullAxisValue(p box3d.Vec3, axis int) float32 {
	switch axis {
	case 0:
		return p.X
	case 1:
		return p.Y
	default:
		return p.Z
	}
}

// HullPartition moves every point at or below mid to the front of the slice
// and returns the two halves. It partitions in place, so no points are copied.
func HullPartition(pts []box3d.Vec3, axis int, mid float32) ([]box3d.Vec3, []box3d.Vec3) {
	i, j := 0, len(pts)-1
	for i <= j {
		if HullAxisValue(pts[i], axis) <= mid {
			i++
		} else {
			pts[i], pts[j] = pts[j], pts[i]
			j--
		}
	}
	return pts[:i], pts[i:]
}

func BuildModelHull(model rl.Model) *box3d.HullData {
	var pts []box3d.Vec3
	for _, mesh := range model.GetMeshes() {
		if mesh.VertexCount <= 0 || mesh.Vertices == nil {
			continue
		}
		vs := unsafe.Slice(mesh.Vertices, int(mesh.VertexCount)*3)
		for i := 0; i < int(mesh.VertexCount); i++ {
			v := rl.Vector3{X: vs[i*3], Y: vs[i*3+1], Z: vs[i*3+2]}
			t := rl.Vector3Transform(v, model.Transform)
			pts = append(pts, box3d.Vec3{X: t.X, Y: t.Y, Z: t.Z})
		}
	}
	if len(pts) < 4 {
		return nil
	}
	// Down-sample if the mesh is huge so hull construction stays cheap.
	if len(pts) > 4000 {
		step := len(pts) / 2000
		if step < 1 {
			step = 1
		}
		sampled := pts[:0]
		// Keep a separate slice to avoid aliasing surprises.
		sampled = make([]box3d.Vec3, 0, len(pts)/step+1)
		for i := 0; i < len(pts); i += step {
			sampled = append(sampled, pts[i])
		}
		pts = sampled
	}
	if len(pts) < 4 {
		return nil
	}
	return box3d.CreateHull(pts, len(pts), 36)
}
