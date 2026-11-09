package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ---------------------------------------------------------------------------
// Path safety for config (root-level) file I/O
// ---------------------------------------------------------------------------

func safeRootPath(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("path is required")
	}
	full := filepath.Clean(filepath.Join(".", filepath.FromSlash(rel)))
	// Prevent escaping the working directory.
	if filepath.IsAbs(full) {
		// strip leading separator then re-clean relative
		full = filepath.Clean(strings.TrimPrefix(full, string(filepath.Separator)))
	}
	if full == ".." || strings.HasPrefix(full, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes root directory")
	}
	return full, nil
}

func (g *Game) apiConfigRead(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeRootPath(a.Path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (g *Game) apiConfigWrite(args json.RawMessage) (any, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Append  bool   `json:"append"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeRootPath(a.Path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return nil, err
	}
	if a.Append {
		f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		_, err = f.WriteString(a.Content)
		return err == nil, err
	}
	err = os.WriteFile(full, []byte(a.Content), 0644)
	return err == nil, err
}

func (g *Game) apiConfigExists(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeRootPath(a.Path)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(full)
	return err == nil, nil
}

func (g *Game) apiConfigDelete(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeRootPath(a.Path)
	if err != nil {
		return nil, err
	}
	err = os.Remove(full)
	return err == nil, err
}

func (g *Game) apiConfigList(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeRootPath(a.Path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	return out, nil
}

func (g *Game) apiConfigMkdir(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	full, err := safeRootPath(a.Path)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(full, 0755)
	return err == nil, err
}

// ---------------------------------------------------------------------------
// 3D model loading
// ---------------------------------------------------------------------------

func (g *Game) apiModelLoad(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	model := rl.LoadModel(a.Path)
	if model.MeshCount == 0 && model.MaterialCount == 0 {
		// LoadModel may still succeed for some formats; check mesh pointer.
	}
	anims := rl.LoadModelAnimations(a.Path)
	g.modModelLast++
	g.modModels[g.modModelLast] = model
	g.modModelAnims[g.modModelLast] = anims
	return g.modModelLast, nil
}

func (g *Game) apiModelUnload(args json.RawMessage) (any, error) {
	var a struct {
		Model int32 `json:"model"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if m, ok := g.modModels[a.Model]; ok {
		rl.UnloadModel(m)
		delete(g.modModels, a.Model)
	}
	if anims, ok := g.modModelAnims[a.Model]; ok {
		rl.UnloadModelAnimations(anims)
		delete(g.modModelAnims, a.Model)
	}
	return true, nil
}

func (g *Game) apiModelListFiles(args json.RawMessage) (any, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	dir := a.Path
	if dir == "" {
		dir = "."
	}
	full, err := safeRootPath(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".glb" || ext == ".gltf" || ext == ".obj" || ext == ".iqm" || ext == ".vox" || ext == ".png" {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Shader loading
// ---------------------------------------------------------------------------

func (g *Game) apiShaderLoad(args json.RawMessage) (any, error) {
	var a struct {
		Vertex   string `json:"vertex"`
		Fragment string `json:"fragment"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	shader := rl.LoadShader(a.Vertex, a.Fragment)
	g.modShaderLast++
	g.modShaders[g.modShaderLast] = shader
	return g.modShaderLast, nil
}

func (g *Game) apiShaderUnload(args json.RawMessage) (any, error) {
	var a struct {
		Shader int32 `json:"shader"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if s, ok := g.modShaders[a.Shader]; ok {
		rl.UnloadShader(s)
		delete(g.modShaders, a.Shader)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Interactive UI elements
// ---------------------------------------------------------------------------

func (g *Game) apiUIAddElement(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Kind  string    `json:"kind"`
		X     float32   `json:"x"`
		Y     float32   `json:"y"`
		W     float32   `json:"w"`
		H     float32   `json:"h"`
		Label string    `json:"label"`
		Color ColorInfo `json:"color"`
		Value float32   `json:"value"`
		Min   float32   `json:"min"`
		Max   float32   `json:"max"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Kind == "" {
		a.Kind = "button"
	}
	if a.W <= 0 {
		a.W = 120
	}
	if a.H <= 0 {
		a.H = 36
	}
	g.modUIElementLast++
	el := ModUIElement{
		ID: g.modUIElementLast, Kind: a.Kind, X: a.X, Y: a.Y, W: a.W, H: a.H,
		Label: a.Label, Value: a.Value, Min: a.Min, Max: a.Max, Owner: owner,
	}
	if a.Color.provided() {
		el.Color = a.Color.rl()
	} else {
		el.Color = colAccent
	}
	g.modUIElements = append(g.modUIElements, el)
	return el.ID, nil
}

func (g *Game) apiUIRemoveElement(args json.RawMessage) (any, error) {
	var a struct {
		ID int32 `json:"id"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	for i, el := range g.modUIElements {
		if el.ID == a.ID {
			g.modUIElements = append(g.modUIElements[:i], g.modUIElements[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (g *Game) apiUISetElement(args json.RawMessage) (any, error) {
	var a struct {
		ID    int32     `json:"id"`
		X     *float32  `json:"x"`
		Y     *float32  `json:"y"`
		W     *float32  `json:"w"`
		H     *float32  `json:"h"`
		Label *string   `json:"label"`
		Color *ColorInfo `json:"color"`
		Value *float32  `json:"value"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	for i := range g.modUIElements {
		if g.modUIElements[i].ID != a.ID {
			continue
		}
		if a.X != nil {
			g.modUIElements[i].X = *a.X
		}
		if a.Y != nil {
			g.modUIElements[i].Y = *a.Y
		}
		if a.W != nil {
			g.modUIElements[i].W = *a.W
		}
		if a.H != nil {
			g.modUIElements[i].H = *a.H
		}
		if a.Label != nil {
			g.modUIElements[i].Label = *a.Label
		}
		if a.Color != nil {
			g.modUIElements[i].Color = a.Color.rl()
		}
		if a.Value != nil {
			g.modUIElements[i].Value = *a.Value
		}
		return true, nil
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// UI element hit-testing and rendering (called from render.go Draw)
// ---------------------------------------------------------------------------

func (g *Game) updateUIElements() {
	if g.mods == nil {
		return
	}
	mouse := g.uiMouse()
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	for _, el := range g.modUIElements {
		rec := rl.Rectangle{X: el.X, Y: el.Y, Width: el.W, Height: el.H}
		if rl.CheckCollisionPointRec(mouse, rec) {
			g.mods.FireEvent("ui.click", map[string]any{"id": el.ID, "kind": el.Kind, "label": el.Label})
			if el.Kind == "button" {
				g.mods.FireEvent("ui.button_click", map[string]any{"id": el.ID, "label": el.Label})
			}
		}
	}
}

func (g *Game) drawUIElements() {
	for _, el := range g.modUIElements {
		rec := rl.Rectangle{X: el.X, Y: el.Y, Width: el.W, Height: el.H}
		switch el.Kind {
		case "panel":
			g.panel(rec, colPanelSoft, colBorderHi)
		case "button":
			hover := rl.CheckCollisionPointRec(g.uiMouse(), rec)
			fill := colPanelSoft
			if hover {
				fill = colPanelHover
			}
			rl.DrawRectangleRec(rec, fill)
			rl.DrawRectangleLinesEx(rec, 1, el.Color)
			if el.Label != "" {
				tw := g.textWidth(el.Label, 13)
				g.drawText(el.Label, rec.X+(rec.Width-tw)*0.5, rec.Y+(rec.Height-13)*0.5, 13, colText)
			}
		case "slider":
			g.panel(rec, colPanelSoft, colBorderHi)
			// track
			trackY := rec.Y + rec.Height*0.5
			rl.DrawLineEx(rl2(rec.X+4, trackY), rl2(rec.X+rec.Width-4, trackY), 2, colBorderHi)
			// knob
			t := float32(0)
			if el.Max > el.Min {
				t = (el.Value - el.Min) / (el.Max - el.Min)
			}
			kx := rec.X + 4 + t*(rec.Width-8)
			rl.DrawCircleV(rl2(kx, trackY), 6, el.Color)
		case "text_input":
			g.panel(rec, colPanelDeep, colBorderHi)
			if el.Label != "" {
				g.drawText(el.Label, rec.X+6, rec.Y+4, 12, colMuted)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Camera update (third person) - called from render.go Draw when cameraMode != 0
// ---------------------------------------------------------------------------

func (g *Game) updateThirdPersonCamera() {
	if g.cameraMode == 0 || g.player == nil {
		return
	}
	p := g.player.body.GetPosition()
	fwd := g.cameraForward()
	// place camera behind player
	camPos := rlv(
		p.X - fwd.X*g.cameraDistance + g.cameraOffset.X,
		p.Y + g.cameraOffset.Y + 1.5,
		p.Z - fwd.Z*g.cameraDistance + g.cameraOffset.Z,
	)
	g.camera.Position = camPos
	g.camera.Target = rlv(p.X, p.Y+1.2, p.Z)
	g.camera.Up = rlv(0, 1, 0)
	g.camera.Fovy = g.settings.FOV
	g.camera.Projection = rl.CameraPerspective
}

// ---------------------------------------------------------------------------
// Player model drawing (third person)
// ---------------------------------------------------------------------------

func (g *Game) drawPlayerModel() {
	if !g.playerHasModel || g.player == nil || g.cameraMode == 0 {
		return
	}
	p := g.player.body.GetPosition()
	q := g.player.body.GetRotation()
	axis, angle := quatAxisAngle(q)
	rl.DrawModelEx(g.playerModel, rlv(p.X, p.Y, p.Z), axis, angle, rlv(1, 1, 1), rl.White)
}

// suppress unused
var _ = box3d.QuatIdentity