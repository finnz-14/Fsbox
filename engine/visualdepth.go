// Package visualdepth implements a small software depth buffer built from the
// animated vertex data of skeletal models.
//
// Physics hulls stay at the bind pose, so a ray test against them cannot see
// where a moving arm or leg actually is. Rasterising the posed triangles gives
// billboard text an accurate occlusion test without touching the GPU pipeline.
//
// The buffer is screen-aligned with the player camera at half resolution.
// Only animated models are rasterised because static geometry and unposed
// props are already covered by the physics ray.
package engine

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Bias is the world-space slack used when testing a billboard point against
// the visual depth buffer. Rasterising at half resolution and sampling the
// nearest texel can make a surface appear a fraction of a unit closer than it
// really is; the bias keeps silhouettes from swallowing text that sits right
// beside an animated limb.
const Bias = 0.06

// Buffer is a tiny software depth buffer built from the animated vertex data
// of skeletal models. Physics hulls stay at the bind pose, so a ray test
// against them cannot see where a moving arm or leg actually is. Rasterising
// the posed triangles gives billboard text an accurate occlusion test without
// touching the GPU pipeline.
//
// The buffer is screen-aligned with the player camera at half resolution.
// Only animated models are rasterised because static geometry and unposed
// props are already covered by the physics ray.
type Buffer struct {
	W, H  int
	depth []float32

	camPos rl.Vector3
	fwd    rl.Vector3
	right  rl.Vector3
	up     rl.Vector3

	tanHalf float32
	aspect  float32
	valid   bool
}

func rlv(x, y, z float32) rl.Vector3 { return rl.Vector3{X: x, Y: y, Z: z} }

func lenRl(v rl.Vector3) float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

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

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Reset prepares the buffer for a new frame. screenW/screenH are the window
// dimensions in pixels; the grid itself is half that size in each axis.
func (b *Buffer) Reset(cam rl.Camera3D, screenW, screenH int) {
	b.valid = false
	if screenW <= 0 || screenH <= 0 {
		return
	}

	b.W = screenW / 2
	if b.W < 1 {
		b.W = 1
	}
	b.H = screenH / 2
	if b.H < 1 {
		b.H = 1
	}

	n := b.W * b.H
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

// Project maps a world point into half-resolution screen coordinates and
// returns its view-space depth. ok is false when the point is behind the near
// plane or otherwise unprojectable.
func (b *Buffer) Project(p rl.Vector3) (sx, sy, depth float32, ok bool) {
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
	sx = (ndcX*0.5 + 0.5) * float32(b.W)
	sy = (0.5 - ndcY*0.5) * float32(b.H)
	return sx, sy, depth, true
}

// RasterizeTriangle writes the depth of one world-space triangle into the
// grid. Both winding orders are accepted because content models are drawn
// double-sided. Depth is interpolated perspective-correctly via 1/z.
func (b *Buffer) RasterizeTriangle(a, c, d rl.Vector3) {
	if !b.valid {
		return
	}
	ax, ay, az, oka := b.Project(a)
	bx, by, bz, okb := b.Project(c)
	cx, cy, cz, okc := b.Project(d)
	if !oka || !okb || !okc {
		return
	}

	minX := int(math.Floor(float64(minf(ax, minf(bx, cx)))))
	maxX := int(math.Ceil(float64(maxf(ax, maxf(bx, cx)))))
	minY := int(math.Floor(float64(minf(ay, minf(by, cy)))))
	maxY := int(math.Ceil(float64(maxf(ay, maxf(by, cy)))))
	minX = clampInt(minX, 0, b.W-1)
	maxX = clampInt(maxX, 0, b.W-1)
	minY = clampInt(minY, 0, b.H-1)
	maxY = clampInt(maxY, 0, b.H-1)
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
		row := y * b.W
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

// Occludes reports whether the visual model is closer to the camera than the
// supplied world point at that point's screen position.
func (b *Buffer) Occludes(p rl.Vector3) bool {
	if !b.valid {
		return false
	}
	sx, sy, depth, ok := b.Project(p)
	if !ok {
		return false
	}
	x := int(sx)
	y := int(sy)
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return false
	}
	d := b.depth[y*b.W+x]
	if d == float32(math.MaxFloat32) {
		return false
	}
	return d < depth-Bias
}
