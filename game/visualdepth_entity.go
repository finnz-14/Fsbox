package game

import (
	"math"
	"unsafe"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"

	"sandbox3d/engine"
)

// rasterizeEntity rasterises one posed model using exactly the transform that
// DrawModelEx applies: model-space vertex, then the model's own transform, then
// the body rotation and translation. The resulting triangles are written into
// the visual depth buffer so billboard text can be occluded by the moving body.
func rasterizeEntityVisualDepth(b *engine.Buffer, e *Entity, p rl.Vector3, q box3d.Quat) {
	if e == nil || !e.HasModel || e.Model.MeshCount <= 0 {
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
					b.RasterizeTriangle(a, c, d)
				}
			}
			continue
		}

		for t := 0; t < tris; t++ {
			a, oka := vertex(t * 3)
			c, okb := vertex(t*3 + 1)
			d, okc := vertex(t*3 + 2)
			if oka && okb && okc {
				b.RasterizeTriangle(a, c, d)
			}
		}
	}
}
