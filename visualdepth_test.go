package main

import (
	"testing"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func testVisualDepthCamera() rl.Camera3D {
	return rl.Camera3D{
		Position: rlv(0, 0, 0),
		Target:   rlv(0, 0, -1),
		Up:       rlv(0, 1, 0),
		Fovy:     60,
	}
}

func TestVisualDepthProjectsCenter(t *testing.T) {
	var b visualDepthBuffer
	b.reset(testVisualDepthCamera(), 800, 600)

	sx, sy, depth, ok := b.project(rlv(0, 0, -10))
	if !ok {
		t.Fatal("center point should project")
	}
	if absf(sx-float32(b.w)*0.5) > 0.01 || absf(sy-float32(b.h)*0.5) > 0.01 {
		t.Fatalf("center projected to (%.2f, %.2f), want (%.2f, %.2f)", sx, sy, float32(b.w)*0.5, float32(b.h)*0.5)
	}
	if absf(depth-10) > 0.01 {
		t.Fatalf("depth = %.2f, want 10", depth)
	}
}

func TestVisualDepthOcclusion(t *testing.T) {
	var b visualDepthBuffer
	b.reset(testVisualDepthCamera(), 800, 600)

	// A triangle at z=-5 spanning the middle of the view.
	b.rasterizeTriangle(rlv(-2, -2, -5), rlv(2, -2, -5), rlv(0, 2, -5))

	if !b.occludes(rlv(0, 0, -10)) {
		t.Fatal("point behind the triangle should be occluded")
	}
	if b.occludes(rlv(0, 0, -2)) {
		t.Fatal("point in front of the triangle should not be occluded")
	}
	if b.occludes(rlv(8, 0, -10)) {
		t.Fatal("point outside the triangle should not be occluded")
	}
}

func TestVisualDepthTriangleBehindPoint(t *testing.T) {
	var b visualDepthBuffer
	b.reset(testVisualDepthCamera(), 800, 600)

	// Triangle far away, point between it and the camera.
	b.rasterizeTriangle(rlv(-8, -8, -20), rlv(8, -8, -20), rlv(0, 8, -20))
	if b.occludes(rlv(0, 0, -5)) {
		t.Fatal("triangle behind the point must not occlude it")
	}
	if !b.occludes(rlv(0, 0, -30)) {
		t.Fatal("point behind the triangle should be occluded")
	}
}

func TestVisualDepthEmptyBuffer(t *testing.T) {
	var b visualDepthBuffer
	if b.occludes(rlv(0, 0, -10)) {
		t.Fatal("uninitialised buffer must not report occlusion")
	}
	b.reset(testVisualDepthCamera(), 0, 0)
	if b.occludes(rlv(0, 0, -10)) {
		t.Fatal("zero-sized buffer must not report occlusion")
	}
}

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

	var b visualDepthBuffer
	b.reset(testVisualDepthCamera(), 800, 600)
	// Entity origin at z=-5, so the local triangle ends up there.
	b.rasterizeEntity(e, rlv(0, 0, -5), box3d.Quat{S: 1})

	if !b.occludes(rlv(0, 0, -10)) {
		t.Fatal("point behind the posed entity should be occluded")
	}
	if b.occludes(rlv(0, 0, -2)) {
		t.Fatal("point in front of the posed entity should not be occluded")
	}
}
