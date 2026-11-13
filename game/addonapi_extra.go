package game

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
) // ---------------------------------------------------------------------------
// Path safety for config (root-level) file I/O
// ---------------------------------------------------------------------------

func safeRootPath(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("path is required")
	}

	root, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("cannot determine root directory: %w", err)
	}

	// Resolve the requested path relative to the exe root so drive-letter
	// absolute paths (C:\Windows\...) on Windows are joined under root,
	// not allowed to escape it.
	full := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))

	// Ensure the resolved path stays inside the root (exe folder).
	relFromRoot, err := filepath.Rel(root, full)
	if err != nil {
		return "", fmt.Errorf("path escapes root directory")
	}
	if relFromRoot == ".." || strings.HasPrefix(relFromRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes root directory")
	}

	return full, nil
}

// ---------------------------------------------------------------------------
// Config API
// ---------------------------------------------------------------------------

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

// MotionLocksInfo is the JSON-serializable form of box3d.MotionLocks.
type MotionLocksInfo struct {
	LinearX  bool `json:"linear_x"`
	LinearY  bool `json:"linear_y"`
	LinearZ  bool `json:"linear_z"`
	AngularX bool `json:"angular_x"`
	AngularY bool `json:"angular_y"`
	AngularZ bool `json:"angular_z"`
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
	return err == nil, err
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
		return nil, fmt.Errorf("model %q has no meshes", a.Path)
	}

	applyMissingTexture(&model)

	// Physics colliders are centred on the body origin, but model files are
	// usually authored with their feet on y=0 or an offset pivot. Move the
	// model so its bounding-box centre sits on the origin, keeping the mesh
	// and the generated hulls aligned.
	bb := rl.GetModelBoundingBox(model)
	cx := (bb.Min.X + bb.Max.X) * 0.5
	cy := (bb.Min.Y + bb.Max.Y) * 0.5
	cz := (bb.Min.Z + bb.Max.Z) * 0.5
	if absf(cx) > 0.001 || absf(cy) > 0.001 || absf(cz) > 0.001 {
		model.Transform = rl.MatrixMultiply(rl.MatrixTranslate(-cx, -cy, -cz), model.Transform)
	}

	// Apply the game's main shader to all materials so the model gets proper
	// lighting and shadows (not fullbright).
	if g.mainShader.ID != 0 {
		for j := 0; j < int(model.MaterialCount); j++ {
			modelMaterial(&model, j).Shader = g.mainShader
		}
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
	el := AddonUIElement{
		ID:    g.modUIElementLast,
		Kind:  a.Kind,
		X:     a.X,
		Y:     a.Y,
		W:     a.W,
		H:     a.H,
		Label: a.Label,
		Value: a.Value,
		Min:   a.Min,
		Max:   a.Max,
		Owner: owner,
	}
	if a.Color.provided() {
		el.Color = a.Color.RL()
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
		ID    int32      `json:"id"`
		X     *float32   `json:"x"`
		Y     *float32   `json:"y"`
		W     *float32   `json:"w"`
		H     *float32   `json:"h"`
		Label *string    `json:"label"`
		Color *ColorInfo `json:"color"`
		Value *float32   `json:"value"`
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
			g.modUIElements[i].Color = a.Color.RL()
		}
		if a.Value != nil {
			g.modUIElements[i].Value = *a.Value
		}
		return true, nil
	}
	return false, nil
}

func (g *Game) updateUIElements() {
	if len(g.modUIElements) == 0 {
		return
	}
	mouse := g.uiMouse()
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	for _, el := range g.modUIElements {
		rec := rl.Rectangle{X: el.X, Y: el.Y, Width: el.W, Height: el.H}
		if rl.CheckCollisionPointRec(mouse, rec) {
			if g.addons != nil {
				g.addons.FireEvent("ui.click", map[string]any{"id": el.ID, "kind": el.Kind, "label": el.Label})
				if el.Kind == "button" {
					g.addons.FireEvent("ui.button_click", map[string]any{"id": el.ID, "label": el.Label})
				}
			}
			return
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
			trackY := rec.Y + rec.Height*0.5
			rl.DrawLineEx(rl2(rec.X+4, trackY), rl2(rec.X+rec.Width-4, trackY), 2, colBorderHi)
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
// Billboard rendering - panels render in 3D world space, always facing camera
// Text renders at screen-space size so it's always readable regardless of distance
// ---------------------------------------------------------------------------

// cameraViewForward is the direction the actual render camera looks. It is
// derived from the camera transform rather than the player yaw so billboards
// stay screen-aligned in third person too. Falls back to the player look
// direction when the camera target is not set.
func (g *Game) cameraViewForward() rl.Vector3 {
	fwd := rlv(
		g.camera.Target.X-g.camera.Position.X,
		g.camera.Target.Y-g.camera.Position.Y,
		g.camera.Target.Z-g.camera.Position.Z,
	)
	if lenRl(fwd) < 1e-6 {
		return g.cameraForward()
	}
	return normRl(fwd)
}

// billboardBasis returns the camera's screen-aligned right/up axes. Using the
// camera axes (instead of the panel-to-camera vector) keeps every panel
// parallel to the near plane, so its projection is a plain axis-aligned
// rectangle. That makes the 3D buttons line up exactly with the 2D text drawn
// over them at any camera angle.
func (g *Game) billboardBasis() (rl.Vector3, rl.Vector3) {
	fwd := g.cameraViewForward()
	upHint := g.camera.Up
	if lenRl(upHint) < 1e-6 {
		upHint = rlv(0, 1, 0)
	}
	right := normRl(rlv(
		fwd.Y*upHint.Z-fwd.Z*upHint.Y,
		fwd.Z*upHint.X-fwd.X*upHint.Z,
		fwd.X*upHint.Y-fwd.Y*upHint.X,
	))
	if lenRl(right) < 1e-6 {
		right = rlv(1, 0, 0)
	}
	up := normRl(rlv(
		right.Y*fwd.Z-right.Z*fwd.Y,
		right.Z*fwd.X-right.X*fwd.Z,
		right.X*fwd.Y-right.Y*fwd.X,
	))
	return right, up
}

// billboardsVisible reports whether world-space addon billboards should render.
// Billboards are world geometry, so they must not bleed through full-screen
// menus and overlays. Hiding them while any overlay is open also keeps hover
// and click state from leaking into the menu.
func (g *Game) billboardsVisible() bool {
	return g.state == statePlaying
}

func (g *Game) drawBillboards() {
	if len(g.modBillboards) == 0 {
		return
	}
	if !g.billboardsVisible() {
		return
	}
	camPos := g.camera.Position
	right, up := g.billboardBasis()

	for _, b := range g.modBillboards {
		// Only the face offset needs the direction to the camera; the panel
		// itself is aligned with the screen.
		toCam := rlv(camPos.X-b.Pos.X, camPos.Y-b.Pos.Y, camPos.Z-b.Pos.Z)
		if lenRl(toCam) < 0.01 {
			continue
		}
		toCam = normRl(toCam)

		hw := b.Width / 2
		hh := b.Height / 2

		// Draw panel quad
		tl := rlv(b.Pos.X+right.X*(-hw)+up.X*(-hh), b.Pos.Y+right.Y*(-hw)+up.Y*(-hh), b.Pos.Z+right.Z*(-hw)+up.Z*(-hh))
		tr := rlv(b.Pos.X+right.X*(hw)+up.X*(-hh), b.Pos.Y+right.Y*(hw)+up.Y*(-hh), b.Pos.Z+right.Z*(hw)+up.Z*(-hh))
		br := rlv(b.Pos.X+right.X*(hw)+up.X*(hh), b.Pos.Y+right.Y*(hw)+up.Y*(hh), b.Pos.Z+right.Z*(hw)+up.Z*(hh))
		bl := rlv(b.Pos.X+right.X*(-hw)+up.X*(hh), b.Pos.Y+right.Y*(-hw)+up.Y*(hh), b.Pos.Z+right.Z*(-hw)+up.Z*(hh))

		bg := rgba(24, 26, 30, 220)
		border := colBorderHi

		rl.DrawTriangle3D(tl, tr, br, bg)
		rl.DrawTriangle3D(tl, br, bl, bg)
		rl.DrawTriangle3D(tr, tl, bl, bg)
		rl.DrawTriangle3D(br, tr, bl, bg)

		// Border lines are nudged a hair toward the camera so they do not
		// z-fight with the panel triangles along the exact same edge.
		bc := rlv(toCam.X*0.004, toCam.Y*0.004, toCam.Z*0.004)
		tl2 := rlv(tl.X+bc.X, tl.Y+bc.Y, tl.Z+bc.Z)
		tr2 := rlv(tr.X+bc.X, tr.Y+bc.Y, tr.Z+bc.Z)
		br2 := rlv(br.X+bc.X, br.Y+bc.Y, br.Z+bc.Z)
		bl2 := rlv(bl.X+bc.X, bl.Y+bc.Y, bl.Z+bc.Z)
		rl.DrawLine3D(tl2, tr2, border)
		rl.DrawLine3D(tr2, br2, border)
		rl.DrawLine3D(br2, bl2, border)
		rl.DrawLine3D(bl2, tl2, border)

		// Draw element quads. Text itself is drawn later in the 2D pass:
		// raylib's text renderer is a 2D primitive and cannot be used while a
		// 3D camera is active.
		for _, el := range b.Elements {
			if el.Kind != "button" {
				continue
			}
			elX := b.Pos.X + right.X*el.X + up.X*el.Y
			elY := b.Pos.Y + right.Y*el.X + up.Y*el.Y
			elZ := b.Pos.Z + right.Z*el.X + up.Z*el.Y

			// Push buttons toward the camera so they never share a plane with
			// the panel face (that caused the flickering/z-fighting).
			const faceOffset = 0.02
			elX += toCam.X * faceOffset
			elY += toCam.Y * faceOffset
			elZ += toCam.Z * faceOffset

			ehw := el.W / 2
			ehh := el.H / 2
			btnTL := rlv(elX+right.X*(-ehw)+up.X*(-ehh), elY+right.Y*(-ehw)+up.Y*(-ehh), elZ+right.Z*(-ehw)+up.Z*(-ehh))
			btnTR := rlv(elX+right.X*(ehw)+up.X*(-ehh), elY+right.Y*(ehw)+up.Y*(-ehh), elZ+right.Z*(ehw)+up.Z*(-ehh))
			btnBR := rlv(elX+right.X*(ehw)+up.X*(ehh), elY+right.Y*(ehw)+up.Y*(ehh), elZ+right.Z*(ehw)+up.Z*(ehh))
			btnBL := rlv(elX+right.X*(-ehw)+up.X*(ehh), elY+right.Y*(-ehw)+up.Y*(ehh), elZ+right.Z*(-ehw)+up.Z*(ehh))

			btnCol := el.Color
			if el.Hovered {
				btnCol = withAlpha(el.Color, 210)
			}
			rl.DrawTriangle3D(btnTL, btnTR, btnBR, btnCol)
			rl.DrawTriangle3D(btnTL, btnBR, btnBL, btnCol)
			rl.DrawLine3D(btnTL, btnTR, colBorderHi)
			rl.DrawLine3D(btnTR, btnBR, colBorderHi)
			rl.DrawLine3D(btnBR, btnBL, colBorderHi)
			rl.DrawLine3D(btnBL, btnTL, colBorderHi)
		}
	}
}

// drawBillboardLabels renders billboard element text in the 2D UI pass. Text
// uses screen-space pixel sizes so it stays readable at any distance, while
// per-glyph physics rays hide only the parts of a label that are behind walls
// or props.
func (g *Game) drawBillboardLabels() {
	if len(g.modBillboards) == 0 {
		return
	}
	if !g.billboardsVisible() {
		return
	}
	camPos := g.camera.Position
	camFwd := g.cameraViewForward()
	s := g.uiScale()
	if s <= 0 {
		s = 1
	}
	right, up := g.billboardBasis()

	for _, b := range g.modBillboards {
		toCam := rlv(camPos.X-b.Pos.X, camPos.Y-b.Pos.Y, camPos.Z-b.Pos.Z)
		dist := lenRl(toCam)
		if dist < 0.01 {
			continue
		}
		toCam = normRl(toCam)
		// toCam points from the panel to the camera. The panel is in front of
		// the camera when the opposite vector agrees with the view direction.
		if -toCam.X*camFwd.X-toCam.Y*camFwd.Y-toCam.Z*camFwd.Z <= 0.05 {
			continue
		}

		for _, el := range b.Elements {
			if el.Label == "" {
				continue
			}
			world := rlv(
				b.Pos.X+right.X*el.X+up.X*el.Y,
				b.Pos.Y+right.Y*el.X+up.Y*el.Y,
				b.Pos.Z+right.Z*el.X+up.Z*el.Y,
			)
			sp := rl.GetWorldToScreen(world, g.camera)
			size := el.TextSize
			if size <= 0 {
				size = 15
			}
			if el.TextWorld {
				// TextSize is a world-space height. Project a segment of that
				// length along the camera up axis at the label position and use
				// the screen distance as the font's pixel size. This makes text
				// grow and shrink with the panel as the player approaches.
				tip := rlv(
					world.X+up.X*size,
					world.Y+up.Y*size,
					world.Z+up.Z*size,
				)
				tp := rl.GetWorldToScreen(tip, g.camera)
				dx := tp.X - sp.X
				dy := tp.Y - sp.Y
				size = float32(math.Sqrt(float64(dx*dx+dy*dy))) / s
				if size < 1 {
					size = 1
				}
			}
			// The label lives on the screen-aligned panel plane, so one world
			// unit per screen pixel at the panel depth lets each glyph be
			// tested against geometry individually.
			depth := (world.X-camPos.X)*camFwd.X + (world.Y-camPos.Y)*camFwd.Y + (world.Z-camPos.Z)*camFwd.Z
			if depth <= 0.01 {
				continue
			}
			fovy := g.camera.Fovy
			if fovy <= 1 {
				fovy = 60
			}
			scrH := float32(rl.GetScreenHeight())
			if scrH <= 0 {
				scrH = 1
			}
			worldPerPixel := 2 * depth * float32(math.Tan(float64(fovy)*math.Pi/360)) / scrH
			tw := g.textWidth(el.Label, size)
			g.drawBillboardLabel(el.Label, world, sp, size, tw, right, worldPerPixel, el.TextColor)
		}
	}
}

// drawBillboardLabel draws one label glyph by glyph, testing each character
// against world geometry. A single ray at the label center used to hide the
// whole string as soon as its middle was blocked, letting the remaining
// characters show through the object. Sampling per glyph and drawing only the
// visible runs keeps partially occluded text correct.
func (g *Game) drawBillboardLabel(label string, world rl.Vector3, sp rl.Vector2, size, tw float32, right rl.Vector3, worldPerPixel float32, col rl.Color) {
	s := g.uiScale()
	if s <= 0 {
		s = 1
	}
	x := sp.X/s - tw*0.5
	y := sp.Y/s - size*0.5

	// Multi-line labels are uncommon; fall back to a single center test so the
	// simple case never loses its shadow or line spacing.
	if strings.ContainsAny(label, "\n\r") {
		if g.billboardPointVisible(world) {
			g.drawText(label, x+1, y+1, size, rgba(0, 0, 0, 210))
			g.drawText(label, x, y, size, col)
		}
		return
	}

	runes := []rune(label)
	if len(runes) == 0 {
		return
	}

	// raylib lays out glyphs by advance + 1px spacing. Reproduce that here so
	// each sampled world point lines up with the pixel the glyph is drawn at.
	const spacing = 1.0
	offsets := make([]float32, len(runes))
	visible := make([]bool, len(runes))
	cursor := float32(0)
	for i, r := range runes {
		cw := g.textWidth(string(r), size)
		offsets[i] = cursor
		mid := cursor + cw*0.5
		// Screen-space horizontal offset from the label center, converted to
		// world units on the screen-aligned panel plane.
		dx := (mid - tw*0.5) * s * worldPerPixel
		p := rlv(
			world.X+right.X*dx,
			world.Y+right.Y*dx,
			world.Z+right.Z*dx,
		)
		visible[i] = g.billboardPointVisible(p)
		cursor += cw + spacing
	}

	// Draw maximal runs of visible glyphs so text stays batched instead of
	// issuing one draw call per character.
	for i := 0; i < len(runes); {
		if !visible[i] {
			i++
			continue
		}
		j := i
		for j < len(runes) && visible[j] {
			j++
		}
		run := string(runes[i:j])
		rx := x + offsets[i]
		g.drawText(run, rx+1, y+1, size, rgba(0, 0, 0, 210))
		g.drawText(run, rx, y, size, col)
		i = j
	}
}

// billboardPointVisible keeps billboard labels from showing through solid
// geometry. Every model is tested against the software visual depth buffer
// because its physics hulls are a coarse bind-pose approximation of the drawn
// mesh. The physics ray is still used for static geometry and non-model props,
// and it skips bodies that belong to model entities.
func (g *Game) billboardPointVisible(p rl.Vector3) bool {
	if g.visualDepth.Occludes(p) {
		return false
	}
	if g.world.IsNull() {
		return true
	}
	origin := g.camera.Position
	dx := p.X - origin.X
	dy := p.Y - origin.Y
	dz := p.Z - origin.Z
	l := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if l < 0.01 {
		return true
	}
	dx /= l
	dy /= l
	dz /= l

	// Step past model bodies instead of treating their bind-pose hulls as
	// occluders. The visual depth buffer already covers the rendered mesh, so
	// the physics hull must not double-count (or contradict) it.
	from := origin
	remaining := l
	for iter := 0; iter < 16 && remaining > 0.01; iter++ {
		res := g.world.CastRayClosest(toBp(from), bv(dx*remaining, dy*remaining, dz*remaining), queryFilter(catStatic|catProp))
		if !res.Hit {
			return true
		}
		hitDist := float32(res.Fraction) * remaining
		if remaining-hitDist < 0.2 {
			return true
		}
		body := res.ShapeId.GetBody()
		e := g.byBody[body]
		if e == nil || !e.HasModel {
			return false
		}
		step := hitDist + 0.02
		if step >= remaining {
			return true
		}
		from = rlv(
			from.X+dx*step,
			from.Y+dy*step,
			from.Z+dz*step,
		)
		remaining -= step
	}
	return true
}

// ---------------------------------------------------------------------------
// Billboard raycast - checks if a ray hits any billboard panel
// Returns the panel ID and element ID hit, or 0,0 if nothing
// ---------------------------------------------------------------------------

func (g *Game) raycastBillboard(origin, dir rl.Vector3, maxDist float32) (int32, int32) {
	if len(g.modBillboards) == 0 {
		return 0, 0
	}
	bestPanel := int32(0)
	bestElem := int32(0)
	bestDist := maxDist

	// Panels are screen-aligned, so their plane normal is the camera view
	// axis and the local axes are the camera's right/up vectors.
	normal := g.cameraViewForward()
	right, up := g.billboardBasis()

	for _, b := range g.modBillboards {
		denom := normal.X*dir.X + normal.Y*dir.Y + normal.Z*dir.Z
		if float32(math.Abs(float64(denom))) < 1e-6 {
			continue
		}
		t := ((b.Pos.X-origin.X)*normal.X + (b.Pos.Y-origin.Y)*normal.Y + (b.Pos.Z-origin.Z)*normal.Z) / denom
		if t < 0.01 || t > bestDist {
			continue
		}

		hitX := origin.X + dir.X*t
		hitY := origin.Y + dir.Y*t
		hitZ := origin.Z + dir.Z*t
		hit := rlv(hitX, hitY, hitZ)

		// Do not let clicks pass through solid geometry. The panel itself has
		// no physics body, so any static/prop hit closer than the panel plane
		// means the panel is occluded.
		if !g.world.IsNull() {
			occ := g.world.CastRayClosest(toBp(origin), bv(dir.X*t, dir.Y*t, dir.Z*t), queryFilter(catStatic|catProp))
			if occ.Hit && float32(occ.Fraction)*t < t-0.15 {
				continue
			}
		}

		toHit := rlv(hit.X-b.Pos.X, hit.Y-b.Pos.Y, hit.Z-b.Pos.Z)
		localX := toHit.X*right.X + toHit.Y*right.Y + toHit.Z*right.Z
		localY := toHit.X*up.X + toHit.Y*up.Y + toHit.Z*up.Z

		if localX < -b.Width/2 || localX > b.Width/2 {
			continue
		}
		if localY < -b.Height/2 || localY > b.Height/2 {
			continue
		}

		hitElem := int32(0)
		for _, el := range b.Elements {
			if el.Kind != "button" {
				continue
			}
			if localX >= el.X-el.W/2 && localX <= el.X+el.W/2 &&
				localY >= el.Y-el.H/2 && localY <= el.Y+el.H/2 {
				hitElem = el.ID
				break
			}
		}

		if t < bestDist {
			bestDist = t
			bestPanel = b.ID
			bestElem = hitElem
		}
	}
	return bestPanel, bestElem
}

// ---------------------------------------------------------------------------
// Billboard API command handlers
// ---------------------------------------------------------------------------

func (g *Game) apiBillboardAdd(owner string, args json.RawMessage) (any, error) {
	var a struct {
		X      float32  `json:"x"`
		Y      float32  `json:"y"`
		Z      float32  `json:"z"`
		FX     float32  `json:"face_x"`
		FY     float32  `json:"face_y"`
		FZ     float32  `json:"face_z"`
		RX     float32  `json:"right_x"`
		RY     float32  `json:"right_y"`
		RZ     float32  `json:"right_z"`
		UX     float32  `json:"up_x"`
		UY     float32  `json:"up_y"`
		UZ     float32  `json:"up_z"`
		Width  float32  `json:"width"`
		Height float32  `json:"height"`
		Tools  []string `json:"tools"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Width <= 0 {
		a.Width = 1
	}
	if a.Height <= 0 {
		a.Height = 1
	}
	face := normRl(rlv(a.FX, a.FY, a.FZ))
	right := normRl(rlv(a.RX, a.RY, a.RZ))
	up := normRl(rlv(a.UX, a.UY, a.UZ))
	g.modBillboardLast++
	b := &AddonBillboard{
		ID:           g.modBillboardLast,
		Owner:        owner,
		Pos:          rlv(a.X, a.Y, a.Z),
		Face:         face,
		Right:        right,
		Up:           up,
		Width:        a.Width,
		Height:       a.Height,
		Elements:     []*AddonBillboardElement{},
		AllowedTools: a.Tools,
	}
	g.modBillboards = append(g.modBillboards, b)
	return b.ID, nil
}

func (g *Game) apiBillboardButton(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Panel     int32      `json:"panel"`
		X         float32    `json:"x"`
		Y         float32    `json:"y"`
		W         float32    `json:"w"`
		H         float32    `json:"h"`
		Label     string     `json:"label"`
		Color     *ColorInfo `json:"color"`
		TextColor *ColorInfo `json:"text_color"`
		Size      float32    `json:"size"`
		World     bool       `json:"world"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	b := g.findBillboard(a.Panel, owner)
	if b == nil {
		return nil, fmt.Errorf("billboard panel %d not found", a.Panel)
	}
	if a.W <= 0 {
		a.W = 0.1
	}
	if a.H <= 0 {
		a.H = 0.05
	}
	btnColor := colAccent
	if a.Color != nil {
		btnColor = a.Color.RL()
	}
	textColor := colText
	if a.TextColor != nil {
		textColor = a.TextColor.RL()
	}
	g.modBillboardLast++
	el := &AddonBillboardElement{
		ID:        g.modBillboardLast,
		Kind:      "button",
		Label:     a.Label,
		X:         a.X,
		Y:         a.Y,
		W:         a.W,
		H:         a.H,
		Color:     btnColor,
		TextColor: textColor,
		TextSize:  a.Size,
		TextWorld: a.World,
	}
	b.Elements = append(b.Elements, el)
	return el.ID, nil
}

func (g *Game) apiBillboardText(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Panel int32      `json:"panel"`
		X     float32    `json:"x"`
		Y     float32    `json:"y"`
		Label string     `json:"label"`
		Size  float32    `json:"size"`
		Color *ColorInfo `json:"color"`
		World bool       `json:"world"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	b := g.findBillboard(a.Panel, owner)
	if b == nil {
		return nil, fmt.Errorf("billboard panel %d not found", a.Panel)
	}
	if a.Size <= 0 {
		a.Size = 22
	}
	textColor := colText
	if a.Color != nil {
		textColor = a.Color.RL()
	}
	g.modBillboardLast++
	el := &AddonBillboardElement{
		ID:        g.modBillboardLast,
		Kind:      "text",
		Label:     a.Label,
		X:         a.X,
		Y:         a.Y,
		Color:     textColor,
		TextColor: textColor,
		TextSize:  a.Size,
		TextWorld: a.World,
	}
	b.Elements = append(b.Elements, el)
	return el.ID, nil
}

func (g *Game) apiBillboardMove(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Panel int32   `json:"panel"`
		X     float32 `json:"x"`
		Y     float32 `json:"y"`
		Z     float32 `json:"z"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	b := g.findBillboard(a.Panel, owner)
	if b == nil {
		return false, nil
	}
	b.Pos = rlv(a.X, a.Y, a.Z)
	return true, nil
}

// apiBillboardUpdate changes an existing billboard element in place. Any field
// left out keeps its current value, so addons can recolour a row or swap its
// label without rebuilding the whole panel.
func (g *Game) apiBillboardUpdate(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Element   int32      `json:"element"`
		X         *float32   `json:"x"`
		Y         *float32   `json:"y"`
		W         *float32   `json:"w"`
		H         *float32   `json:"h"`
		Label     *string    `json:"label"`
		Color     *ColorInfo `json:"color"`
		TextColor *ColorInfo `json:"text_color"`
		Size      *float32   `json:"size"`
		World     *bool      `json:"world"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	for _, b := range g.modBillboards {
		if b.Owner != owner {
			continue
		}
		for _, el := range b.Elements {
			if el.ID != a.Element {
				continue
			}
			if a.X != nil {
				el.X = *a.X
			}
			if a.Y != nil {
				el.Y = *a.Y
			}
			if a.W != nil {
				el.W = *a.W
			}
			if a.H != nil {
				el.H = *a.H
			}
			if a.Label != nil {
				el.Label = *a.Label
			}
			if a.Color != nil {
				el.Color = a.Color.RL()
			}
			if a.TextColor != nil {
				el.TextColor = a.TextColor.RL()
			}
			if a.Size != nil {
				el.TextSize = *a.Size
			}
			if a.World != nil {
				el.TextWorld = *a.World
			}
			return true, nil
		}
	}
	return false, nil
}

func (g *Game) apiBillboardRemove(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Panel int32 `json:"panel"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	for i, b := range g.modBillboards {
		if b.ID == a.Panel && b.Owner == owner {
			g.modBillboards = append(g.modBillboards[:i], g.modBillboards[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (g *Game) apiBillboardClear(owner string, args json.RawMessage) (any, error) {
	kept := g.modBillboards[:0]
	for _, b := range g.modBillboards {
		if b.Owner != owner {
			kept = append(kept, b)
		}
	}
	g.modBillboards = kept
	return true, nil
}

func (g *Game) apiBillboardSetTools(owner string, args json.RawMessage) (any, error) {
	var a struct {
		Panel int32    `json:"panel"`
		Tools []string `json:"tools"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	for _, b := range g.modBillboards {
		if b.ID == a.Panel && (owner == "" || b.Owner == owner) {
			b.AllowedTools = a.Tools
			return true, nil
		}
	}
	return false, nil
}

func (g *Game) findBillboard(id int32, owner string) *AddonBillboard {
	for _, b := range g.modBillboards {
		if b.ID == id && b.Owner == owner {
			return b
		}
	}
	return nil
}

func (g *Game) updateBillboardHover() {
	g.modBillboardHoverPanel = 0
	g.modBillboardHoverElem = 0
	if len(g.modBillboards) == 0 || !g.billboardsVisible() {
		return
	}
	camFwd := g.cameraViewForward()
	camPos := g.camera.Position
	panelID, elemID := g.raycastBillboard(camPos, camFwd, 64)
	g.modBillboardHoverPanel = panelID
	g.modBillboardHoverElem = elemID

	for _, b := range g.modBillboards {
		for _, el := range b.Elements {
			el.Hovered = false
		}
	}

	if elemID > 0 {
		for _, b := range g.modBillboards {
			for _, el := range b.Elements {
				if el.ID == elemID {
					el.Hovered = true
				}
			}
		}
	}
}

func (g *Game) updateThirdPersonCamera() {
	if g.cameraMode == 0 || g.player == nil {
		return
	}
	p := g.player.body.GetPosition()
	fwd := g.cameraForward()
	camPos := rlv(p.X-fwd.X*g.cameraDistance+g.cameraOffset.X,
		p.Y+g.cameraOffset.Y+1.5, p.Z-fwd.Z*g.cameraDistance+g.cameraOffset.Z)
	g.camera.Position = camPos
	g.camera.Target = rlv(p.X, p.Y+1.2, p.Z)
	g.camera.Up = rlv(0, 1, 0)
	g.camera.Fovy = g.settings.FOV
	g.camera.Projection = rl.CameraPerspective
}

func (g *Game) drawPlayerModel() {
	if !g.playerHasModel || g.player == nil || g.cameraMode == 0 {
		return
	}
	p := g.player.body.GetPosition()
	q := g.player.body.GetRotation()
	axis, angle := quatAxisAngle(q)
	rl.DrawModelEx(g.playerModel, rlv(p.X, p.Y, p.Z), axis, angle, rlv(1, 1, 1), rl.White)
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
