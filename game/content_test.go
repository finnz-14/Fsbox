package game

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	box3d "github.com/gataengine/box3d-go"
)

func TestContentManifestSelectsMainModel(t *testing.T) {
	objectDir := filepath.Join(t.TempDir(), "decorate", "crate")
	if err := os.MkdirAll(objectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"crate.glb", "crate.obj", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(objectDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"main object":"crate.obj","name":"Wooden Crate","density":42,"half":[0.5,0.6,0.7]}`
	if err := os.WriteFile(filepath.Join(objectDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	mf, path, ok := readContentManifest(objectDir)
	if !ok || path == "" {
		t.Fatalf("manifest was not read")
	}
	if mf.Main != "crate.obj" {
		t.Fatalf("manifest main = %q, want crate.obj", mf.Main)
	}
	if mf.Name != "Wooden Crate" {
		t.Fatalf("manifest name = %q, want Wooden Crate", mf.Name)
	}
	if mf.Density != 42 {
		t.Fatalf("manifest density = %v, want 42", mf.Density)
	}
	if len(mf.Half) != 3 || mf.Half[2] != 0.7 {
		t.Fatalf("manifest half = %v, want [0.5 0.6 0.7]", mf.Half)
	}

	models := collectModelFiles(objectDir)
	if len(models) != 2 {
		t.Fatalf("found %d models, want 2: %v", len(models), models)
	}
	if got := resolveModelFile(objectDir, models, mf.Main); filepath.Base(got) != "crate.obj" {
		t.Fatalf("resolved main model = %q, want crate.obj", got)
	}
	if got := chooseDefaultModel(models); filepath.Base(got) != "crate.glb" {
		t.Fatalf("default model = %q, want crate.glb", got)
	}
}

func TestContentModelResolutionFallsBackToBasename(t *testing.T) {
	objectDir := filepath.Join(t.TempDir(), "props", "barrel")
	if err := os.MkdirAll(objectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	model := filepath.Join(objectDir, "Barrel.gltf")
	if err := os.WriteFile(model, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	models := collectModelFiles(objectDir)
	if got := resolveModelFile(objectDir, models, "subdir/barrel.gltf"); filepath.Base(got) != "Barrel.gltf" {
		t.Fatalf("resolved model = %q, want Barrel.gltf", got)
	}
}

func TestContentSearchCandidatesAreUnique(t *testing.T) {
	candidates := contentSearchCandidates()
	if len(candidates) == 0 {
		t.Fatal("no content search candidates")
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		key := filepath.Clean(candidate)
		if seen[key] {
			t.Fatalf("duplicate candidate: %s", candidate)
		}
		seen[key] = true
	}
}

// syntheticHumanoidPoints builds a point cloud with the same shape as a simple
// character: a sphere head, a box torso, and thin cylindrical limbs. It exists
// to make sure the hull decomposition covers regions that the old uniform grid
// used to miss.
func syntheticHumanoidPoints() []box3d.Vec3 {
	var pts []box3d.Vec3
	appendSphere := func(cx, cy, cz, r float32, n int) {
		for i := 0; i < n; i++ {
			t := float32(i)
			y := 1 - 2*(t+0.5)/float32(n)
			ring := float32(math.Sqrt(float64(1 - y*y)))
			phi := t * 2.399963
			pts = append(pts, box3d.Vec3{
				X: cx + r*ring*float32(math.Cos(float64(phi))),
				Y: cy + r*y,
				Z: cz + r*ring*float32(math.Sin(float64(phi))),
			})
		}
	}
	appendCylinder := func(cx, cy0, cy1, r float32, n int) {
		for i := 0; i < n; i++ {
			a := float32(i) * 0.7
			h := float32(i%89) / 88
			pts = append(pts, box3d.Vec3{
				X: cx + r*float32(math.Cos(float64(a))),
				Y: cy0 + (cy1-cy0)*h,
				Z: r * float32(math.Sin(float64(a))),
			})
		}
	}
	appendSphere(0, 0.78, 0, 0.18, 1400)
	appendCylinder(0, -0.1, 0.55, 0.22, 1200) // torso
	appendCylinder(-0.4, 0.1, 0.55, 0.06, 700)
	appendCylinder(0.4, 0.1, 0.55, 0.06, 700)
	appendCylinder(-0.15, -0.95, -0.1, 0.09, 900)
	appendCylinder(0.15, -0.95, -0.1, 0.09, 900)
	return pts
}

func pointInsideAnyHull(p box3d.Vec3, hulls []*box3d.HullData, eps float32) bool {
	for _, h := range hulls {
		lo := h.AABB.LowerBound
		hi := h.AABB.UpperBound
		if p.X >= lo.X-eps && p.X <= hi.X+eps &&
			p.Y >= lo.Y-eps && p.Y <= hi.Y+eps &&
			p.Z >= lo.Z-eps && p.Z <= hi.Z+eps {
			return true
		}
	}
	return false
}

func TestBuildHullsFromPointsCoversHeadAndLimbs(t *testing.T) {
	pts := syntheticHumanoidPoints()
	original := append([]box3d.Vec3(nil), pts...)
	hulls := BuildHullsFromPoints(pts)
	if len(hulls) < 2 {
		t.Fatalf("got %d hulls, want a multi-hull decomposition", len(hulls))
	}

	const eps = 0.01 // hull simplification may shave a few millimetres
	for i, p := range original {
		if !pointInsideAnyHull(p, hulls, eps) {
			t.Fatalf("point %d at (%.3f,%.3f,%.3f) is not covered by any hull", i, p.X, p.Y, p.Z)
		}
	}

	// The head and thin limbs are the regions the uniform grid used to leave
	// empty, so check them explicitly as well as via the full coverage sweep.
	for _, probe := range []box3d.Vec3{
		{X: 0, Y: 0.95, Z: 0},
		{X: -0.4, Y: 0.3, Z: 0},
		{X: 0.4, Y: 0.3, Z: 0},
		{X: -0.15, Y: -0.9, Z: 0},
		{X: 0.15, Y: -0.9, Z: 0},
	} {
		if !pointInsideAnyHull(probe, hulls, 0.02) {
			t.Fatalf("probe (%.2f,%.2f,%.2f) is not covered by any hull", probe.X, probe.Y, probe.Z)
		}
	}
}

func TestBuildHullsFromPointsHandlesFlatLeaf(t *testing.T) {
	// A perfectly flat patch has no volume and the hull builder rejects it.
	// The jitter fallback should still turn it into a thin slab.
	var pts []box3d.Vec3
	for i := 0; i < 40; i++ {
		for j := 0; j < 40; j++ {
			pts = append(pts, box3d.Vec3{X: float32(i) * 0.01, Y: 1, Z: float32(j) * 0.01})
		}
	}
	hulls := BuildHullsFromPoints(pts)
	if len(hulls) == 0 {
		t.Fatal("flat point cloud produced no hulls")
	}
	if !pointInsideAnyHull(box3d.Vec3{X: 0.2, Y: 1, Z: 0.2}, hulls, 0.01) {
		t.Fatal("flat point cloud is not covered by its hulls")
	}
}

func TestHullLongestAxis(t *testing.T) {
	lower := box3d.Vec3{X: -1, Y: -2, Z: -1}
	upper := box3d.Vec3{X: 1, Y: 2, Z: 5}
	if got := HullLongestAxis(lower, upper); got != 2 {
		t.Fatalf("longest axis = %d, want Z", got)
	}
	if got := HullLongestAxis(lower, box3d.Vec3{X: 5, Y: 2, Z: 3}); got != 0 {
		t.Fatalf("longest axis = %d, want X", got)
	}
}
