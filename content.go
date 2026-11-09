package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unsafe"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// contentSpawnable is the legacy loose-JSON format that used to live directly
// in Content/Spawnables. Folder based content is preferred, but keeping this
// format working avoids breaking existing content packs.
type contentSpawnable struct {
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Half        []float32 `json:"half"`
	Radius      float32   `json:"radius"`
	Height      float32   `json:"height"`
	Density     float32   `json:"density"`
	Friction    float32   `json:"friction"`
	Restitution float32   `json:"restitution"`
	Color       []int     `json:"color"`
	Category    string    `json:"category"`
}

// contentManifest is the optional manifest.json stored inside an object folder.
// "main" points at the model file that should represent the spawnable. A few
// common aliases are accepted so content packs authored by hand keep working.
type contentManifest struct {
	Main        string
	Name        string
	Density     float32
	Friction    float32
	Restitution float32
	Color       []int
	Half        []float32
}

//go:embed Content/Spawnables
var embeddedSpawnables embed.FS

//go:embed Content/Maps/*.json
var embeddedMaps embed.FS

var contentLoaded bool
var missingTexture rl.Texture2D

type mapVec struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type mapStatic struct {
	Kind    string  `json:"kind"`
	Name    string  `json:"name"`
	Pos     mapVec  `json:"pos"`
	Half    mapVec  `json:"half"`
	Radius  float32 `json:"radius"`
	Height  float32 `json:"height"`
	Color   []int   `json:"color"`
	Yaw     float32 `json:"yaw"`
	Pitch   float32 `json:"pitch"`
	Roll    float32 `json:"roll"`
}

type mapProp struct {
	Preset string  `json:"preset"`
	Pos    mapVec  `json:"pos"`
	Color  []int   `json:"color"`
	Yaw    float32 `json:"yaw"`
	Pitch  float32 `json:"pitch"`
	Roll   float32 `json:"roll"`
}

type mapFile struct {
	Name   string     `json:"name"`
	Static []mapStatic `json:"static"`
	Props  []mapProp   `json:"props"`
}

func initMissingTexture() {
	if missingTexture.ID != 0 {
		return
	}
	img := rl.GenImageChecked(8, 8, 4, 4, rl.Black, rl.NewColor(255, 0, 255, 255))
	missingTexture = rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
}

// contentSearchCandidates lists the places Content/Spawnables may live. The
// game is commonly launched from bin\fsbox.exe, so the working directory alone
// is not enough: the folder also needs to be found next to (or above) the
// executable.
func contentSearchCandidates() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = filepath.Clean(p)
		}
		key := strings.ToLower(abs)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, abs)
	}

	add(filepath.Join("Content", "Spawnables"))
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		add(filepath.Join(exeDir, "Content", "Spawnables"))
		add(filepath.Join(exeDir, "..", "Content", "Spawnables"))
		add(filepath.Join(exeDir, "..", "..", "Content", "Spawnables"))
	}
	return out
}

func findContentSpawnablesDir() (string, []string) {
	candidates := contentSearchCandidates()

	if wd, err := os.Getwd(); err == nil {
		fmt.Printf("[content] working directory: %s\n", wd)
	}
	if exe, err := os.Executable(); err == nil {
		fmt.Printf("[content] executable: %s\n", exe)
	}
	fmt.Println("[content] searching for Content/Spawnables:")
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			fmt.Printf("[content]   MISS %s (%v)\n", candidate, err)
			continue
		}
		if !info.IsDir() {
			fmt.Printf("[content]   MISS %s (not a directory)\n", candidate)
			continue
		}
		fmt.Printf("[content]   FOUND %s\n", candidate)
		return candidate, candidates
	}
	return "", candidates
}

func loadContentSpawnables() {
	fmt.Println("[content] loadContentSpawnables called")
	if contentLoaded {
		fmt.Println("[content] already loaded, skipping")
		return
	}
	contentLoaded = true
	initMissingTexture()

	before := len(presets)
	dir, _ := findContentSpawnablesDir()
	if dir == "" {
		fmt.Println("[content] WARNING: no Content/Spawnables folder found on disk; using embedded defaults")
		loadEmbeddedSpawnables()
		sortPresets()
		rebuildSpawnCategories()
		fmt.Printf("[content] done: %d presets (embedded defaults), categories=%v\n", len(presets), spawnCategories)
		return
	}

	loadLooseJSONSpawnables(dir)
	loadFolderSpawnables(dir)
	sortPresets()
	rebuildSpawnCategories()

	fmt.Printf("[content] done: %d presets (%d built-in + %d content), categories=%v\n",
		len(presets), before, len(presets)-before, spawnCategories)
}

// loadLooseJSONSpawnables keeps the old "drop a .json in Spawnables" format
// alive. Folder based content is the preferred layout.
func loadLooseJSONSpawnables(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("[content] cannot read %s: %v\n", dir, err)
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			// Recurse into category subfolders.
			loadLooseJSONSpawnables(filepath.Join(dir, e.Name()))
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		if strings.EqualFold(e.Name(), "manifest.json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("[content] legacy JSON %s: read failed: %v\n", path, err)
			continue
		}
		var cs contentSpawnable
		if err := json.Unmarshal(data, &cs); err != nil {
			fmt.Printf("[content] legacy JSON %s: parse failed: %v\n", path, err)
			continue
		}
		if cs.Name == "" {
			cs.Name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}
		p := ShapePreset{Name: cs.Name, Density: cs.Density, Friction: cs.Friction, Restitution: cs.Restitution}
		switch strings.ToLower(cs.Kind) {
		case "sphere", "ball":
			p.Kind = PropSphere
			if cs.Radius <= 0 {
				cs.Radius = 0.5
			}
			p.Radius = cs.Radius
		case "cylinder", "barrel":
			p.Kind = PropCylinder
			if cs.Radius <= 0 {
				cs.Radius = 0.4
			}
			if cs.Height <= 0 {
				cs.Height = 1.0
			}
			p.Radius = cs.Radius
			p.Height = cs.Height
		default:
			p.Kind = PropBox
			hx, hy, hz := float32(0.5), float32(0.5), float32(0.5)
			if len(cs.Half) >= 1 {
				hx = cs.Half[0]
			}
			if len(cs.Half) >= 2 {
				hy = cs.Half[1]
			}
			if len(cs.Half) >= 3 {
				hz = cs.Half[2]
			}
			p.Half = bv(hx, hy, hz)
		}
		if len(cs.Color) >= 3 {
			p.Color = rl.NewColor(uint8(cs.Color[0]), uint8(cs.Color[1]), uint8(cs.Color[2]), 255)
		} else {
			p.Color = rl.Gray
		}
		if p.Density <= 0 {
			p.Density = 150
		}
		if p.Friction <= 0 {
			p.Friction = 0.6
		}
		if cs.Category != "" {
			p.Category = strings.ToUpper(cs.Category)
		} else {
			p.Category = "MISC"
		}
		presets = append(presets, p)
		fmt.Printf("[content] legacy JSON spawnable: %q category=%s\n", p.Name, p.Category)
	}
}

// loadEmbeddedSpawnables loads the default spawnables embedded in the binary.
// This keeps tests and headless runs working even when the working directory
// does not contain a Content/Spawnables folder.
func loadEmbeddedSpawnables() {
	var walk func(base string)
	walk = func(base string) {
		entries, err := embeddedSpawnables.ReadDir(base)
		if err != nil {
			return
		}
		for _, e := range entries {
			full := base + "/" + e.Name()
			if e.IsDir() {
				walk(full)
				continue
			}
			if !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			if strings.EqualFold(e.Name(), "manifest.json") {
				continue
			}
			data, err := embeddedSpawnables.ReadFile(full)
			if err != nil {
				continue
			}
			var cs contentSpawnable
			if err := json.Unmarshal(data, &cs); err != nil {
				continue
			}
			if cs.Name == "" {
				cs.Name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			}
		p := ShapePreset{Name: cs.Name, Density: cs.Density, Friction: cs.Friction, Restitution: cs.Restitution}
		switch strings.ToLower(cs.Kind) {
		case "sphere", "ball":
			p.Kind = PropSphere
			if cs.Radius <= 0 {
				cs.Radius = 0.5
			}
			p.Radius = cs.Radius
		case "cylinder", "barrel":
			p.Kind = PropCylinder
			if cs.Radius <= 0 {
				cs.Radius = 0.4
			}
			if cs.Height <= 0 {
				cs.Height = 1.0
			}
			p.Radius = cs.Radius
			p.Height = cs.Height
		default:
			p.Kind = PropBox
			hx, hy, hz := float32(0.5), float32(0.5), float32(0.5)
			if len(cs.Half) >= 1 {
				hx = cs.Half[0]
			}
			if len(cs.Half) >= 2 {
				hy = cs.Half[1]
			}
			if len(cs.Half) >= 3 {
				hz = cs.Half[2]
			}
			p.Half = bv(hx, hy, hz)
		}
		if len(cs.Color) >= 3 {
			p.Color = rl.NewColor(uint8(cs.Color[0]), uint8(cs.Color[1]), uint8(cs.Color[2]), 255)
		} else {
			p.Color = rl.Gray
		}
		if p.Density <= 0 {
			p.Density = 150
		}
		if p.Friction <= 0 {
			p.Friction = 0.6
		}
		if cs.Category != "" {
			p.Category = strings.ToUpper(cs.Category)
		} else {
			p.Category = "MISC"
		}
		presets = append(presets, p)
		}
	}
	walk("Content/Spawnables")
}

// loadFolderSpawnables implements the documented layout:
//
//	Content/Spawnables/<category>/<object>/<model.obj|glb|gltf>
//	Content/Spawnables/<category>/<object>/manifest.json   (optional)
func loadFolderSpawnables(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Printf("[content] cannot list %s: %v\n", root, err)
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, categoryEntry := range entries {
		if !categoryEntry.IsDir() {
			continue
		}
		category := categoryEntry.Name()
		categoryDir := filepath.Join(root, category)
		fmt.Printf("[content] category %q: %s\n", category, categoryDir)

		objects, err := os.ReadDir(categoryDir)
		if err != nil {
			fmt.Printf("[content]   cannot list category: %v\n", err)
			continue
		}

		// Be forgiving when a pack skips the object folder and drops the model
		// directly into the category folder.
		if models := collectModelFiles(categoryDir); len(models) > 0 {
			fmt.Printf("[content]   NOTE: models are directly inside category %q; expected <category>/<object>/<model>\n", category)
			loadContentObject(category, categoryDir)
		}

		for _, objectEntry := range objects {
			if !objectEntry.IsDir() {
				continue
			}
			loadContentObject(category, filepath.Join(categoryDir, objectEntry.Name()))
		}
	}
}

func loadContentObject(category, objectDir string) {
	objectName := filepath.Base(objectDir)
	fmt.Printf("[content]   object %q: %s\n", objectName, objectDir)

	manifest, manifestPath, hasManifest := readContentManifest(objectDir)
	if hasManifest {
		fmt.Printf("[content]     manifest: %s main=%q name=%q\n", manifestPath, manifest.Main, manifest.Name)
	} else {
		fmt.Println("[content]     manifest: none")
	}

	models := collectModelFiles(objectDir)
	if len(models) == 0 {
		// A pack may keep the actual model in a subfolder (for example
		// <object>/model/crate.glb). Only do this fallback when the object
		// folder itself has no model files.
		models = collectModelFilesRecursive(objectDir)
	}
	if len(models) == 0 {
		fmt.Printf("[content]     SKIP: no .obj/.glb/.gltf found in %s\n", objectDir)
		return
	}

	names := make([]string, 0, len(models))
	for _, modelPath := range models {
		names = append(names, filepath.Base(modelPath))
	}
	fmt.Printf("[content]     model files: %v\n", names)

	mainModel := ""
	if manifest.Main != "" {
		mainModel = resolveModelFile(objectDir, models, manifest.Main)
		if mainModel == "" {
			fmt.Printf("[content]     WARNING: manifest main object %q not found, falling back to default selection\n", manifest.Main)
		}
	}
	if mainModel == "" {
		mainModel = chooseDefaultModel(models)
		fmt.Printf("[content]     default main object: %q\n", filepath.Base(mainModel))
	} else {
		fmt.Printf("[content]     manifest main object: %q\n", filepath.Base(mainModel))
	}

	fmt.Printf("[content]     loading model: %s\n", mainModel)
	model := rl.LoadModel(mainModel)
	fmt.Printf("[content]     raylib: meshes=%d materials=%d\n", model.MeshCount, model.MaterialCount)
	if model.MeshCount == 0 {
		fmt.Printf("[content]     SKIP: raylib loaded no meshes from %s\n", mainModel)
		return
	}

	applyMissingTexture(&model)
	bb := rl.GetModelBoundingBox(model)
	hx := (bb.Max.X - bb.Min.X) * 0.5
	hy := (bb.Max.Y - bb.Min.Y) * 0.5
	hz := (bb.Max.Z - bb.Min.Z) * 0.5
	if hx < 0.1 {
		hx = 0.5
	}
	if hy < 0.1 {
		hy = 0.5
	}
	if hz < 0.1 {
		hz = 0.5
	}
	if len(manifest.Half) >= 3 {
		hx, hy, hz = manifest.Half[0], manifest.Half[1], manifest.Half[2]
		fmt.Printf("[content]     manifest half override: (%.2f, %.2f, %.2f)\n", hx, hy, hz)
	}
	fmt.Printf("[content]     bounds min=(%.2f, %.2f, %.2f) max=(%.2f, %.2f, %.2f) half=(%.2f, %.2f, %.2f)\n",
		bb.Min.X, bb.Min.Y, bb.Min.Z, bb.Max.X, bb.Max.Y, bb.Max.Z, hx, hy, hz)

	// The physics collider is centred on the body origin, but model files are
	// rarely authored that way (feet at y=0, origin off to one side, ...).
	// Offset the model so its bounding-box centre sits on the body origin;
	// otherwise the rendered mesh and the hitbox drift apart.
	cx := (bb.Min.X + bb.Max.X) * 0.5
	cy := (bb.Min.Y + bb.Max.Y) * 0.5
	cz := (bb.Min.Z + bb.Max.Z) * 0.5
	if absf(cx) > 0.001 || absf(cy) > 0.001 || absf(cz) > 0.001 {
		model.Transform = rl.MatrixMultiply(rl.MatrixTranslate(-cx, -cy, -cz), model.Transform)
		fmt.Printf("[content]     centered model offset=(%.3f, %.3f, %.3f)\n", cx, cy, cz)
	}

	// Build an exact triangle mesh from the model's vertices so collision
	// follows every triangle of the rendered geometry (not just a hull or AABB).
	mesh := buildModelMesh(model)
	if mesh != nil {
		fmt.Printf("[content]     mesh: vertices=%d triangles=%d\n", mesh.VertexCount, mesh.TriangleCount)
	} else {
		fmt.Printf("[content]     WARNING: mesh generation failed\n")
	}

	// Build a set of convex hulls (spatial decomposition) so the collision
	// shape follows concave models without filling every empty region, while
	// remaining compatible with dynamic rigid-body collisions.
	hulls := buildModelHulls(model)
	if len(hulls) > 0 {
		fmt.Printf("[content]     hulls: %d pieces\n", len(hulls))
	} else {
		fmt.Printf("[content]     WARNING: hull decomposition failed\n")
	}

	// Also build a single convex hull as a fallback.
	hull := buildModelHull(model)
	if hull != nil {
		fmt.Printf("[content]     hull: vertices=%d faces=%d\n", hull.VertexCount, hull.FaceCount)
	} else {
		fmt.Printf("[content]     WARNING: hull generation failed; falling back to box collider\n")
	}

	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		name = objectName
	}

	p := ShapePreset{
		Name:        name,
		Category:    strings.ToUpper(category),
		Kind:        PropBox,
		Half:        bv(hx, hy, hz),
		Density:     150,
		Friction:    0.6,
		Restitution: 0.03,
		Color:       rl.White,
		Model:       model,
		HasModel:    true,
		Hull:        hull,
		Hulls:       hulls,
		Mesh:        mesh,
	}
	if manifest.Density > 0 {
		p.Density = manifest.Density
	}
	if manifest.Friction > 0 {
		p.Friction = manifest.Friction
	}
	if manifest.Restitution > 0 {
		p.Restitution = manifest.Restitution
	}
	if len(manifest.Color) >= 3 {
		p.Color = rl.NewColor(uint8(manifest.Color[0]), uint8(manifest.Color[1]), uint8(manifest.Color[2]), 255)
	}

	presets = append(presets, p)
	fmt.Printf("[content]     ADDED preset %q category=%s density=%.1f friction=%.2f restitution=%.2f\n",
		p.Name, p.Category, p.Density, p.Friction, p.Restitution)
}

func collectModelFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isModelFile(e.Name()) {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sortModels(out)
	return out
}

func collectModelFilesRecursive(dir string) []string {
	var out []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if isModelFile(path) {
			out = append(out, path)
		}
		return nil
	})
	sortModels(out)
	return out
}

func isModelFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".obj", ".glb", ".gltf":
		return true
	default:
		return false
	}
}

func modelPriority(path string) int {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".glb":
		return 0
	case ".gltf":
		return 1
	case ".obj":
		return 2
	default:
		return 3
	}
}

func sortModels(models []string) {
	sort.Slice(models, func(i, j int) bool {
		pi, pj := modelPriority(models[i]), modelPriority(models[j])
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(filepath.Base(models[i])) < strings.ToLower(filepath.Base(models[j]))
	})
}

func chooseDefaultModel(models []string) string {
	if len(models) == 0 {
		return ""
	}
	// sortModels already puts glb/gltf before obj and then sorts by name.
	return models[0]
}

func resolveModelFile(objectDir string, models []string, want string) string {
	want = strings.TrimSpace(want)
	if want == "" {
		return ""
	}
	want = filepath.FromSlash(want)

	if filepath.IsAbs(want) {
		if info, err := os.Stat(want); err == nil && !info.IsDir() && isModelFile(want) {
			return want
		}
	}

	direct := filepath.Join(objectDir, want)
	if info, err := os.Stat(direct); err == nil && !info.IsDir() && isModelFile(direct) {
		return direct
	}

	wantBase := strings.ToLower(filepath.Base(want))
	wantClean := strings.ToLower(filepath.Clean(want))
	for _, modelPath := range models {
		if strings.ToLower(filepath.Base(modelPath)) == wantBase {
			return modelPath
		}
		rel, err := filepath.Rel(objectDir, modelPath)
		if err == nil && strings.ToLower(filepath.Clean(rel)) == wantClean {
			return modelPath
		}
	}
	return ""
}

func findManifestFile(objectDir string) string {
	entries, err := os.ReadDir(objectDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(e.Name(), "manifest.json") {
			return filepath.Join(objectDir, e.Name())
		}
	}
	return ""
}

func readContentManifest(objectDir string) (contentManifest, string, bool) {
	var mf contentManifest
	path := findManifestFile(objectDir)
	if path == "" {
		return mf, "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[content]     manifest read failed: %v\n", err)
		return mf, path, false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		fmt.Printf("[content]     manifest parse failed: %v\n", err)
		return mf, path, false
	}
	if v, ok := manifestValue(raw, "main", "mainobject", "mainmodel", "mainfile", "object", "model", "file", "spawnable", "prefab"); ok {
		mf.Main = jsonStringValue(v)
	}
	if v, ok := manifestValue(raw, "name", "displayname", "title"); ok {
		mf.Name = jsonStringValue(v)
	}
	if v, ok := manifestValue(raw, "density"); ok {
		mf.Density = jsonNumberValue(v)
	}
	if v, ok := manifestValue(raw, "friction"); ok {
		mf.Friction = jsonNumberValue(v)
	}
	if v, ok := manifestValue(raw, "restitution", "bounce"); ok {
		mf.Restitution = jsonNumberValue(v)
	}
	if v, ok := manifestValue(raw, "color", "colour"); ok {
		mf.Color = jsonIntSlice(v)
	}
	if v, ok := manifestValue(raw, "half"); ok {
		mf.Half = jsonFloatSlice(v)
	}
	return mf, path, true
}

func normalizeJSONKey(key string) string {
	key = strings.ToLower(key)
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(key)
}

func manifestValue(m map[string]json.RawMessage, keys ...string) (json.RawMessage, bool) {
	for _, key := range keys {
		want := normalizeJSONKey(key)
		for actual, value := range m {
			if normalizeJSONKey(actual) == want {
				return value, true
			}
		}
	}
	return nil, false
}

func jsonStringValue(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	// Also accept a nested object such as {"file": "crate.glb"}.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		if v, ok := manifestValue(obj, "file", "model", "main", "name", "object", "path"); ok {
			return jsonStringValue(v)
		}
	}
	return ""
}

func jsonNumberValue(raw json.RawMessage) float32 {
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return float32(f)
	}
	return 0
}

func jsonIntSlice(raw json.RawMessage) []int {
	var ints []int
	if err := json.Unmarshal(raw, &ints); err == nil {
		return ints
	}
	var floats []float64
	if err := json.Unmarshal(raw, &floats); err == nil {
		ints = make([]int, len(floats))
		for i, f := range floats {
			ints[i] = int(f)
		}
		return ints
	}
	return nil
}

func jsonFloatSlice(raw json.RawMessage) []float32 {
	var floats []float64
	if err := json.Unmarshal(raw, &floats); err == nil {
		out := make([]float32, len(floats))
		for i, f := range floats {
			out[i] = float32(f)
		}
		return out
	}
	return nil
}

// buildModelHull samples a content model's mesh vertices (transformed by the
// model's own transform) and builds a convex hull suitable for box3d. The
// result is capped to a safe vertex count so it stays within box3d's limits.
// buildModelMesh turns a raylib model into a box3d triangle mesh so the
// collision shape follows every vertex/triangle of the rendered geometry.
func buildModelMesh(model rl.Model) *box3d.MeshData {
	var verts []box3d.Vec3
	var inds []int32
	for _, mesh := range model.GetMeshes() {
		if mesh.VertexCount <= 0 || mesh.Vertices == nil {
			continue
		}
		base := int32(len(verts))
		vs := unsafe.Slice(mesh.Vertices, int(mesh.VertexCount)*3)
		for i := 0; i < int(mesh.VertexCount); i++ {
			v := rl.Vector3{X: vs[i*3], Y: vs[i*3+1], Z: vs[i*3+2]}
			t := rl.Vector3Transform(v, model.Transform)
			verts = append(verts, box3d.Vec3{X: t.X, Y: t.Y, Z: t.Z})
		}
		tc := int(mesh.TriangleCount)
		if tc <= 0 {
			tc = int(mesh.VertexCount) / 3
		}
		if mesh.Indices != nil {
			idxs := unsafe.Slice(mesh.Indices, tc*3)
			for t := 0; t < tc; t++ {
				i0 := int32(idxs[t*3])
				i1 := int32(idxs[t*3+1])
				i2 := int32(idxs[t*3+2])
				inds = append(inds, base+i0, base+i1, base+i2)
			}
		} else {
			for t := 0; t < tc; t++ {
				inds = append(inds, base+int32(t*3), base+int32(t*3+1), base+int32(t*3+2))
			}
		}
	}
	if len(verts) < 3 || len(inds) < 3 {
		return nil
	}
	def := box3d.MeshDef{
		Vertices:      verts,
		Indices:       inds,
		VertexCount:   int32(len(verts)),
		TriangleCount: int32(len(inds) / 3),
	}
	return box3d.CreateMesh(&def, nil)
}

// buildModelHulls decomposes a model into multiple convex hulls by spatially
// partitioning its vertices. This approximates concave geometry better than a
// single hull while staying compatible with dynamic rigid bodies.
func buildModelHulls(model rl.Model) []*box3d.HullData {
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
	minX, minY, minZ := verts[0].X, verts[0].Y, verts[0].Z
	maxX, maxY, maxZ := verts[0].X, verts[0].Y, verts[0].Z
	for _, v := range verts {
		if v.X < minX {
			minX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.Z < minZ {
			minZ = v.Z
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
		if v.Z > maxZ {
			maxZ = v.Z
		}
	}
	sx := maxX - minX
	sy := maxY - minY
	sz := maxZ - minZ
	if sx < 1e-6 {
		sx = 1
	}
	if sy < 1e-6 {
		sy = 1
	}
	if sz < 1e-6 {
		sz = 1
	}
	const divs = 4
	type cell struct {
		pts []box3d.Vec3
	}
	cells := make(map[int]*cell)
	for _, v := range verts {
		ix := int((v.X - minX) / sx * divs)
		iy := int((v.Y - minY) / sy * divs)
		iz := int((v.Z - minZ) / sz * divs)
		if ix < 0 {
			ix = 0
		}
		if ix >= divs {
			ix = divs - 1
		}
		if iy < 0 {
			iy = 0
		}
		if iy >= divs {
			iy = divs - 1
		}
		if iz < 0 {
			iz = 0
		}
		if iz >= divs {
			iz = divs - 1
		}
		key := ix + iy*divs + iz*divs*divs
		c, ok := cells[key]
		if !ok {
			c = &cell{}
			cells[key] = c
		}
		c.pts = append(c.pts, v)
	}

	var hulls []*box3d.HullData
	for _, c := range cells {
		if len(c.pts) < 4 {
			continue
		}
		h := box3d.CreateHull(c.pts, len(c.pts), 24)
		if h != nil {
			hulls = append(hulls, h)
		}
	}
	if len(hulls) == 0 {
		h := buildModelHull(model)
		if h != nil {
			hulls = append(hulls, h)
		}
	}
	return hulls
}

func buildModelHull(model rl.Model) *box3d.HullData {
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

func modelMaterial(model *rl.Model, index int) *rl.Material {
	return (*rl.Material)(unsafe.Pointer(uintptr(unsafe.Pointer(model.Materials)) + uintptr(index)*unsafe.Sizeof(rl.Material{})))
}

func applyMissingTexture(model *rl.Model) {
	for i := 0; i < int(model.MaterialCount); i++ {
		mat := modelMaterial(model, i)
		if diffuse := mat.GetMap(0); diffuse != nil && diffuse.Texture.ID == 0 {
			diffuse.Texture = missingTexture
		}
	}
}

func presetOrder(name string) int {
	order := map[string]int{
		"Crate": 0, "Ball": 1, "Barrel": 2, "Slab": 3, "Anvil": 4,
		"Block": 5, "Pebble": 6, "Boulder": 7, "Plank": 8, "Column": 9,
	}
	if v, ok := order[name]; ok {
		return v
	}
	return 1000 + len(name)
}

func sortPresets() {
	sort.SliceStable(presets, func(i, j int) bool {
		return presetOrder(presets[i].Name) < presetOrder(presets[j].Name)
	})
}

func rebuildSpawnCategories() {
	cats := []string{"ALL"}
	seen := map[string]bool{}
	for _, p := range presets {
		c := presetCategoryName(p)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		cats = append(cats, c)
	}
	spawnCategories = cats
}

// Keep box3d import used.
var _ = box3d.Vec3Zero

func findMapFile() (string, bool) {
	candidates := []string{
		filepath.Join("Content", "Maps", "default.json"),
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "Content", "Maps", "default.json"),
			filepath.Join(exeDir, "..", "Content", "Maps", "default.json"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, true
		}
	}
	return "", false
}

func (g *Game) loadMap(withStarter bool) {
	path, ok := findMapFile()
	var data []byte
	var err error
	if ok {
		data, err = os.ReadFile(path)
	}
	if err != nil || len(data) == 0 {
		// embedded fallback
		data, err = embeddedMaps.ReadFile("Content/Maps/default.json")
	}
	if err != nil {
		fmt.Printf("[map] failed to load map: %v\n", err)
		return
	}
	var m mapFile
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Printf("[map] parse failed: %v\n", err)
		return
	}

	// Static geometry
	for _, s := range m.Static {
		col := rl.Color{R: 128, G: 128, B: 128, A: 255}
		if len(s.Color) >= 3 {
			col = rl.NewColor(uint8(s.Color[0]), uint8(s.Color[1]), uint8(s.Color[2]), 255)
		}
		rot := box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, s.Yaw)
		rot = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, s.Pitch), rot)
		rot = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{Z: 1}, s.Roll), rot)
		rot = normalizeQuat(rot)
		hx, hy, hz := float32(0.5), float32(0.5), float32(0.5)
		if s.Half.X != 0 { hx = s.Half.X }
		if s.Half.Y != 0 { hy = s.Half.Y }
		if s.Half.Z != 0 { hz = s.Half.Z }
		g.addBoxEntity(true, bp(s.Pos.X, s.Pos.Y, s.Pos.Z), rot, bv(hx, hy, hz), col, s.Name, 1000, 0.7, 0.0)
	}

	if !withStarter {
		return
	}

	// Starter props
	for _, pp := range m.Props {
		preset := findPreset(pp.Preset)
		if preset == nil {
			continue
		}
		col := preset.Color
		if len(pp.Color) >= 3 {
			col = rl.NewColor(uint8(pp.Color[0]), uint8(pp.Color[1]), uint8(pp.Color[2]), 255)
		}
		rot := box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, pp.Yaw)
		rot = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, pp.Pitch), rot)
		rot = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{Z: 1}, pp.Roll), rot)
		rot = normalizeQuat(rot)
		g.spawnPreset(*preset, bp(pp.Pos.X, pp.Pos.Y, pp.Pos.Z), rot, col)
	}
}

func findPreset(name string) *ShapePreset {
	for i := range presets {
		if presets[i].Name == name {
			return &presets[i]
		}
	}
	return nil
}
