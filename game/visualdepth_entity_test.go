package game

import (
	"testing"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"

	"sandbox3d/engine"
)

func TestVisualDepthRasterizeEntityTransform(t *testing.T) {
	verts := []float32{
		-1, -1, 0,
		1, -1, 0,
		0, 1, 0,
	}
	indices := []uint16{0, 1, 2}
	mesh := rl.Mesh{
		VertexCount:   3,
		TriangleCount: 1,
		Vertices:      &verts[0],
		Indices:       &indices[0],
	}
	model := rl.Model{
		Transform: rl.MatrixIdentity(),
		MeshCount: 1,
		Meshes:    &mesh,
	}
	e := &Entity{HasModel: true, Model: model, AnimActive: true}

	var b engine.Buffer
	b.Reset(testVisualDepthCamera(), 800, 600)
	// Entity origin at z=-5, so the local triangle ends up there.
	rasterizeEntityVisualDepth(&b, e, rlv(0, 0, -5), box3d.Quat{S: 1})

	if !b.Occludes(rlv(0, 0, -10)) {
		t.Fatal("point behind the posed entity should be occluded")
	}
	if b.Occludes(rlv(0, 0, -2)) {
		t.Fatal("point in front of the posed entity should not be occluded")
	}
}

func testVisualDepthCamera() rl.Camera3D {
	return rl.Camera3D{
		Position: rlv(0, 0, 0),
		Target:   rlv(0, 0, -1),
		Up:       rlv(0, 1, 0),
		Fovy:     60,
	}
}
