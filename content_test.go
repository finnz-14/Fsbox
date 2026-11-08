package main

import (
	"os"
	"path/filepath"
	"testing"
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
