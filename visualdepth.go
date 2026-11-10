package main

import (
	"math"
	"unsafe"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// visualDepthBias is the world-space slack used when testing a billboard point
// against the visual depth buffer. Rasterising at half resolution and sampling
// the nearest texel can make a surface appear a fraction of a unit closer than
// it really is; the bias keeps silhouettes from swallowing text that sits right
// beside an animated limb.
const visualDepthBias = 0.06

// visualDepthBuffer is a tiny software depth buffer built from the animated
// vertex data of skeletal models. Physics hulls stay at the bind pose, so a
// ray test against them cannot see where a moving arm or leg actually is.
// Rasterising the posed triangles gives billboard text an accurate occlusion
// test without touching the GPU pipeline.
//
// The buffer is screen-aligned with the player camera at half resolution. Only
// animated models are rasterised because static geometry and unposed props are
// already covered by the physics ray.
type visualDepthBuffer struct {
	w, h  int
	depth []float32

	camPos rl.Vector3
	fwd    rl.Vector3
	right  rl.Vector3
	up     rl.Vector3

	tanHalf float32
	aspect  float32
	valid   bool
}

// reset prepares the buffer for a new frame. screenW/screenH are the window
// dimensions in pixels; the grid itself is half that size in each axis.
func (b *visualDepthBuffer) reset(cam rl.Camera3D, screenW, screenH int) {
	b.valid = false
	if screenW <= 0 || screenH <= 0 {
		return
	}

	b.w = screenW / 2
	if b.w < 1 {
		b.w = 1
	}
	b.h = screenH / 2
	if b.h < 1 {
		b.h = 1
	}

	n := b.w * b.h
	if cap(b.depth) < n {
		b.depth = make([]float32, n)
	} else {
		b.depth = b.depth[:n]
	}
	maxDepth := float32(math.MaxFloat32)
	for i := range b.depth {
		b.depth[i] = maxDepth
	}

	fwd := rlv(
		cam.Target.X-cam.Position.X,
		cam.Target.Y-cam.Position.Y,
		cam.Target.Z-cam.Position.Z,
	)
	l := lenRl(fwd)
	if l < 1e-6 {
		return
	}
	fwd = rlv(fwd.X/l, fwd.Y/l, fwd.Z/l)

	upHint := cam.Up
	if lenRl(upHint) < 1e-6 {
		upHint = rlv(0, 1, 0)
	}
	right := rlv(
		fwd.Y*upHint.Z-fwd.Z*upHint.Y,
		fwd.Z*upHint.X-fwd.X*upHint.Z,
		fwd.X*upHint.Y-fwd.Y*upHint.X,
	)
	rlen := lenRl(right)
	if rlen < 1e-6 {
		right = rlv(1, 0, 0)
	} else {
		right = rlv(right.X/rlen, right.Y/rlen, right.Z/rlen)
	}
	up := rlv(
		right.Y*fwd.Z-right.Z*fwd.Y,
		right.Z*fwd.X-right.X*fwd.Z,
		right.X*fwd.Y-right.Y*fwd.X,
	)

	fovy := cam.Fovy
	if fovy <= 1 || fovy >= 179 {
		fovy = 60
	}

	b.camPos = cam.Position
	b.fwd = fwd
	b.right = right
	b.up = up
	b.tanHalf = float32(math.Tan(float64(fovy) * math.Pi / 360))
	b.aspect = float32(screenW) / float32(screenH)
	b.valid = true
}

// project maps a world point into half-resolution screen coordinates and
// returns its view-space depth. ok is false when the point is behind the near
// plane or otherwise unprojectable.
func (b *visualDepthBuffer) project(p rl.Vector3) (sx, sy, depth float32, ok bool) {
	if !b.valid {
		return 0, 0, 0, false
	}
	rx := p.X - b.camPos.X
	ry := p.Y - b.camPos.Y
	rz := p.Z - b.camPos.Z

	depth = rx*b.fwd.X + ry*b.fwd.Y + rz*b.fwd.Z
	if depth <= 0.02 {
		return 0, 0, 0, false
	}

	x := rx*b.right.X + ry*b.right.Y + rz*b.right.Z
	y := rx*b.up.X + ry*b.up.Y + rz*b.up.Z

	inv := 1 / (depth * b.tanHalf)
	ndcX := x * inv / b.aspect
	ndcY := y * inv
	sx = (ndcX*0.5 + 0.5) * float32(b.w)
	sy = (0.5 - ndcY*0.5) * float32(b.h)
	return sx, sy, depth, true
}

// rasterizeTriangle writes the depth of one world-space triangle into the
// grid. Both winding orders are accepted because content models are drawn
// double-sided. Depth is interpolated perspective-correctly via 1/z.
func (b *visualDepthBuffer) rasterizeTriangle(a, c, d rl.Vector3) {
	if !b.valid {
		return
	}
	ax, ay, az, oka := b.project(a)
	bx, by, bz, okb := b.project(c)
	cx, cy, cz, okc := b.project(d)
	if !oka || !okb || !okc {
		return
	}

	minX := int(math.Floor(float64(minf(ax, minf(bx, cx)))))
	maxX := int(math.Ceil(float64(maxf(ax, maxf(bx, cx)))))
	minY := int(math.Floor(float64(minf(ay, minf(by, cy)))))
	maxY := int(math.Ceil(float64(maxf(ay, maxf(by, cy)))))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= b.w {
		maxX = b.w - 1
	}
	if maxY >= b.h {
		maxY = b.h - 1
	}
	if minX > maxX || minY > maxY {
		return
	}

	area := (bx-ax)*(cy-ay) - (by-ay)*(cx-ax)
	if absf(area) < 1e-6 {
		return
	}
	if area < 0 {
		// Flip to a consistent winding so the barycentric weights stay
		// positive inside the triangle.
		bx, cx = cx, bx
		by, cy = cy, by
		bz, cz = cz, bz
		area = -area
	}

	invZ0 := 1 / az
	invZ1 := 1 / bz
	invZ2 := 1 / cz

	for y := minY; y <= maxY; y++ {
		py := float32(y) + 0.5
		row := y * b.w
		for x := minX; x <= maxX; x++ {
			px := float32(x) + 0.5
			w0 := ((bx-px)*(cy-py) - (by-py)*(cx-px)) / area
			w1 := ((cx-px)*(ay-py) - (cy-py)*(ax-px)) / area
			w2 := 1 - w0 - w1
			if w0 < -1e-4 || w1 < -1e-4 || w2 < -1e-4 {
				continue
			}
			invZ := w0*invZ0 + w1*invZ1 + w2*invZ2
			if invZ <= 0 {
				continue
			}
			z := 1 / invZ
			idx := row + x
			if z < b.depth[idx] {
				b.depth[idx] = z
			}
		}
	}
}

// rasterizeEntity rasterises one posed model using exactly the transform that
// DrawModelEx applies: model-space vertex, then the model's own transform, then
// the body rotation and translation.
func (b *visualDepthBuffer) rasterizeEntity(e *Entity, p rl.Vector3, q box3d.Quat) {
	if !b.valid || e == nil || !e.HasModel || e.Model.MeshCount <= 0 {
		return
	}

	axis, angleDeg := quatAxisAngle(q)
	mat := rl.MatrixMultiply(
		e.Model.Transform,
		rl.MatrixMultiply(
			rl.MatrixRotate(axis, angleDeg*float32(math.Pi)/180),
			rl.MatrixTranslate(p.X, p.Y, p.Z),
		),
	)

	meshes := e.Model.GetMeshes()
	for i := range meshes {
		mesh := &meshes[i]
		if mesh.VertexCount <= 0 {
			continue
		}

		var verts []float32
		if mesh.AnimVertices != nil {
			verts = unsafe.Slice(mesh.AnimVertices, int(mesh.VertexCount)*3)
		} else if mesh.Vertices != nil {
			verts = unsafe.Slice(mesh.Vertices, int(mesh.VertexCount)*3)
		} else {
			continue
		}

		vertex := func(idx int) (rl.Vector3, bool) {
			if idx < 0 || idx >= int(mesh.VertexCount) {
				return rl.Vector3{}, false
			}
			v := rl.Vector3{
				X: verts[idx*3],
				Y: verts[idx*3+1],
				Z: verts[idx*3+2],
			}
			return rl.Vector3Transform(v, mat), true
		}

		tris := int(mesh.TriangleCount)
		if tris <= 0 {
			tris = int(mesh.VertexCount) / 3
		}
		if tris <= 0 {
			continue
		}

		if mesh.Indices != nil {
			idx := unsafe.Slice(mesh.Indices, tris*3)
			for t := 0; t < tris; t++ {
				a, oka := vertex(int(idx[t*3]))
				c, okb := vertex(int(idx[t*3+1]))
				d, okc := vertex(int(idx[t*3+2]))
				if oka && okb && okc {
					b.rasterizeTriangle(a, c, d)
				}
			}
			continue
		}

		for t := 0; t < tris; t++ {
			a, oka := vertex(t * 3)
			c, okb := vertex(t*3 + 1)
			d, okc := vertex(t*3 + 2)
			if oka && okb && okc {
				b.rasterizeTriangle(a, c, d)
			}
		}
	}
}

// occludes reports whether the visual model is closer to the camera than the
// supplied world point at that point's screen position.
func (b *visualDepthBuffer) occludes(p rl.Vector3) bool {
	if !b.valid {
		return false
	}
	sx, sy, depth, ok := b.project(p)
	if !ok {
		return false
	}
	x := int(sx)
	y := int(sy)
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return false
	}
	d := b.depth[y*b.w+x]
	if d == float32(math.MaxFloat32) {
		return false
	}
	return d < depth-visualDepthBias
}
