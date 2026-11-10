package main

import (
	"math"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ToolID identifies one operating mode of the tool gun. The gun is the single
// interface for physics interaction; spawning, deleting and editing are all
// tool modes rather than free-floating hotkeys.
type ToolID int

const (
	toolHand ToolID = iota
	toolPhysgun
	toolSpawner
	toolRemover
	toolImpact
	toolFreeze
	toolDuplicator
	toolGravity
	toolCount
)

type ToolDef struct {
	Name      string
	Short     string
	Tag       string
	Color     rl.Color
	Desc      string
	Primary   string
	Secondary string
}

var toolDefs = []ToolDef{
	{
		Name: "HAND", Short: "HND", Tag: "INTERACT", Color: colAccentHi,
		Desc:      "Your bare hand. Click UI elements to interact, grab and throw props.",
		Primary:   "INTERACT / GRAB",
		Secondary: "THROW",
	},
	{
		Name: "PHYSGUN", Short: "PHY", Tag: "PHYSICS", Color: colAccent,
		Desc:      "Magnetic gravity manipulator. Holds a body in a stable field and launches it on release.",
		Primary:   "GRAB / UNGRAB",
		Secondary: "SHOOT",
	},
	{
		Name: "SPAWNER", Short: "SPN", Tag: "BUILD", Color: colSpawn,
		Desc:      "Materialises the prop selected in the spawn menu at the aim point.",
		Primary:   "SPAWN PROP",
		Secondary: "OPEN SPAWN MENU",
	},
	{
		Name: "REMOVER", Short: "DEL", Tag: "EDIT", Color: colDanger,
		Desc:      "Dissolves targeted matter. The secondary mode clears a short radius around the impact.",
		Primary:   "DELETE TARGET",
		Secondary: "CLEAR RADIUS 3.2 m",
	},
	{
		Name: "IMPACT", Short: "IMP", Tag: "PHYSICS", Color: rgb(214, 140, 86),
		Desc:      "Kinetic pulse emitter. Fires a radial shockwave or injects a directional impulse.",
		Primary:   "SHOCKWAVE",
		Secondary: "LAUNCH TARGET",
	},
	{
		Name: "FREEZE", Short: "FRZ", Tag: "CONTROL", Color: colFreeze,
		Desc:      "Locks a rigid body in world space. Frozen props ignore gravity and impulses.",
		Primary:   "TOGGLE TARGET",
		Secondary: "TOGGLE ALL PROPS",
	},
	{
		Name: "DUPLICATOR", Short: "DUP", Tag: "BUILD", Color: colDuplicate,
		Desc:      "Copies the targeted body with its exact mass, material and orientation.",
		Primary:   "COPY OFFSET",
		Secondary: "COPY TO SURFACE",
	},
	{
		Name: "GRAVITY", Short: "GRV", Tag: "WORLD", Color: colGravityTool,
		Desc:      "Field authorisation for world gravity. Cycle the presets or cut gravity entirely.",
		Primary:   "CYCLE PRESET",
		Secondary: "ZERO-G TOGGLE",
	},
}

func (g *Game) toolDef() ToolDef {
	if int(g.tool) < 0 || int(g.tool) >= len(toolDefs) {
		return toolDefs[0]
	}
	return toolDefs[g.tool]
}

func (g *Game) toolColor() rl.Color {
	return g.toolDef().Color
}

func (g *Game) equipTool(id ToolID) {
	if int(id) < 0 || int(id) >= len(toolDefs) {
		return
	}
	if id != toolPhysgun && g.held != nil {
		g.held = nil
	}
	g.tool = id
	g.toolMenuSel = id
}

func (g *Game) cycleTool(dir int) {
	n := int(toolCount)
	next := (int(g.tool) + dir%n + n) % n
	g.equipTool(ToolID(next))
}

// ---------------------------------------------------------------------------
// Tool actions
// ---------------------------------------------------------------------------

func (g *Game) primaryAction() {
	if g.state != statePlaying {
		return
	}
	g.toolKick = 1
	if g.activeModTool >= 0 {
		_, _ = g.fireToolAction("tool.primary", nil, "")
		return
	}
	switch g.tool {
	case toolHand:
		// If already holding something, release it (ungrab).
		if g.held != nil {
			if g.mods != nil {
				g.mods.FireEvent("entity.release", g.entityInfo(g.held))
			}
			g.held = nil
			g.toast("RELEASED", colMuted)
			return
		}
		// Raycast against 3D billboards first - if crosshair is over a
		// billboard button, trigger its click instead of grabbing.
		if g.handRaycastBillboard() {
			return
		}
		// First try to click a 2D UI element (mod UI panel, etc.).
		if g.clickUIElement() {
			return
		}
		// Otherwise grab the aimed entity.
		g.grab()
	case toolPhysgun:
		if g.held != nil {
			if g.mods != nil {
				g.mods.FireEvent("entity.release", g.entityInfo(g.held))
			}
			g.held = nil
		} else {
			g.grab()
		}
	case toolSpawner:
		g.spawnSelectedAt(g.spawnPoint())
	case toolRemover:
		g.deleteAimed()
	case toolImpact:
		g.blast()
	case toolFreeze:
		e, _ := g.castPropRay(28)
		g.toggleFrozen(e)
	case toolDuplicator:
		g.duplicateAimed(false)
	case toolGravity:
		g.cycleGravity()
	}
}

func (g *Game) secondaryAction() {
	if g.state != statePlaying {
		return
	}
	g.toolKick = 1
	if g.activeModTool >= 0 {
		_, _ = g.fireToolAction("tool.secondary", nil, "")
		return
	}
	switch g.tool {
	case toolHand:
		if g.held != nil {
			g.throwHeld()
		} else {
			g.pushAimed()
		}
	case toolPhysgun:
		if g.held != nil {
			g.throwHeld()
		} else {
			g.pushAimed()
		}
	case toolSpawner:
		g.openSpawnMenu()
	case toolRemover:
		center := g.aimPoint
		if vlen(center.X, center.Y, center.Z) < 0.01 {
			center = g.eyePosition()
		}
		n := g.removeInRadius(center, 3.2)
		if n == 0 {
			g.toast("No props in radius", colMuted)
		} else {
			g.toast("Radius cleared: "+itoa(n), colDanger)
		}
	case toolImpact:
		g.launchAimed()
	case toolFreeze:
		freeze := false
		for _, e := range g.props {
			if !e.Frozen {
				freeze = true
				break
			}
		}
		n := g.setAllFrozen(freeze)
		if freeze {
			g.toast("Frozen all: "+itoa(n), colFreeze)
		} else {
			g.toast("Thawed all: "+itoa(n), colFreeze)
		}
	case toolDuplicator:
		g.duplicateAimed(true)
	case toolGravity:
		g.toggleZeroGravity()
	}
}

// clickUIElement checks if the mouse is over any interactive UI element and
// triggers its click. Returns true if something was clicked.
func (g *Game) clickUIElement() bool {
	mouse := g.uiMouse()

	// Mod UI elements.
	for _, el := range g.modUIElements {
		rec := rl.Rectangle{X: el.X, Y: el.Y, Width: el.W, Height: el.H}
		if rl.CheckCollisionPointRec(mouse, rec) {
			if g.mods != nil {
				g.mods.FireEvent("ui.click", map[string]any{"id": el.ID, "kind": el.Kind, "label": el.Label})
				if el.Kind == "button" {
					g.mods.FireEvent("ui.button_click", map[string]any{"id": el.ID, "label": el.Label})
				}
			}
			return true
		}
	}

	return false
}

// handRaycastBillboard raycasts from the camera and checks if the crosshair
// hits a 3D billboard button. Returns true if a billboard button was clicked.
func (g *Game) handRaycastBillboard() bool {
	if g.player == nil || len(g.modBillboards) == 0 {
		return false
	}
	camPos := g.camera.Position
	camFwd := g.cameraForward()
	panelID, elemID := g.raycastBillboard(camPos, camFwd, 64)
	if panelID == 0 {
		return false
	}
	// Aiming at any part of the panel consumes the click, so the entity behind
	// it can never be grabbed through the interface.
	if elemID != 0 && g.mods != nil {
		g.mods.FireEvent("ui.click", map[string]any{"id": elemID, "kind": "billboard_button", "panel": panelID})
		g.mods.FireEvent("ui.button_click", map[string]any{"id": elemID, "panel": panelID})
	}
	return true
}

func (g *Game) spawnPoint() rl.Vector3 {
	if g.ghostOK {
		return g.ghostPos
	}
	eye := g.eyePosition()
	f := g.cameraForward()
	d := g.placementDistance()
	return rlv(eye.X+f.X*d, eye.Y+f.Y*d, eye.Z+f.Z*d)
}

// freezeAimed is the global freeze key (F). It prefers a carried body so the
// player can lock something they are already holding, then falls back to the
// aimed prop.
func (g *Game) freezeAimed() {
	e := g.held
	if e == nil {
		e, _ = g.castPropRay(28)
	}
	g.toggleFrozen(e)
}

// pushAimed is the physgun secondary action. It shoves the held or aimed body
// away from the camera with a mass-scaled impulse.
func (g *Game) pushAimed() {
	e := g.held
	if e != nil {
		g.held = nil
	} else {
		e, _ = g.castPropRay(28)
	}
	if e == nil {
		g.toast("No target to push", colMuted)
		return
	}
	if e.Frozen {
		g.toast("Target is frozen", colFreeze)
		return
	}
	dir := g.cameraForward()
	mass := e.Body.GetMass()
	if mass <= 0 {
		mass = 1
	}
	e.Body.ApplyLinearImpulseToCenter(bv(
		dir.X*mass*16,
		(dir.Y+0.18)*mass*16,
		dir.Z*mass*16,
	), true)
	e.Body.ApplyAngularImpulse(bv(
		(g.rng.Float32()-0.5)*mass*2.5,
		(g.rng.Float32()-0.5)*mass*2.5,
		(g.rng.Float32()-0.5)*mass*2.5,
	), true)
	g.burst(e.position(), colAccent, 8, 3.5, 0.08)
	g.shake = 0.08
}

func (g *Game) toggleZeroGravity() {
	if g.settings.Gravity <= 0.001 {
		g.setGravity(1)
		g.toast("GRAVITY RESTORED // 1.00x", colGravityTool)
		return
	}
	g.setGravity(0)
	g.toast("ZERO-G ENGAGED", colGravityTool)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ---------------------------------------------------------------------------
// Tool selector menu
// ---------------------------------------------------------------------------

func (g *Game) openToolMenu() {
	if g.state != statePlaying {
		return
	}
	g.toolMenuSel = g.tool
	g.state = stateToolMenu
	g.ignoreMouse = true
}

func (g *Game) closeToolMenu() {
	g.state = statePlaying
	g.justResumed = true
	g.ignoreMouse = true
}

func (g *Game) toolMenuPanel() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	pw := minf(840, w-80)
	ph := minf(624, h-80)
	return rl.Rectangle{X: (w - pw) * 0.5, Y: (h - ph) * 0.5, Width: pw, Height: ph}
}

func (g *Game) toolMenuRow(index int) rl.Rectangle {
	rec := g.toolMenuPanel()
	const rowH, gap = 50.0, 6.0
	return rl.Rectangle{
		X:      rec.X + 26,
		Y:      rec.Y + 94 + float32(index)*(rowH+gap),
		Width:  rec.Width - 52,
		Height: rowH,
	}
}

func (g *Game) toolMenuButtons() (equip, close rl.Rectangle) {
	rec := g.toolMenuPanel()
	y := rec.Y + rec.Height - 54
	return rl.Rectangle{X: rec.X + 26, Y: y, Width: 210, Height: 36},
		rl.Rectangle{X: rec.X + rec.Width - 26 - 130, Y: y, Width: 130, Height: 36}
}

func (g *Game) updateToolMenuInput() {
	if g.menuIgnoreClick {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			g.menuIgnoreClick = false
		}
		return
	}
	mouse := g.uiMouse()
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	for i := 0; i < int(toolCount); i++ {
		if rl.CheckCollisionPointRec(mouse, g.toolMenuRow(i)) {
			g.toolMenuSel = ToolID(i)
			return
		}
	}
	equip, close := g.toolMenuButtons()
	if rl.CheckCollisionPointRec(mouse, equip) {
		g.equipTool(g.toolMenuSel)
		g.closeToolMenu()
		return
	}
	if rl.CheckCollisionPointRec(mouse, close) {
		g.closeToolMenu()
	}
}

func (g *Game) drawToolMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 205))

	rec := g.toolMenuPanel()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.drawTextSpaced("TOOL GUN", rec.X+26, rec.Y+20, 26, 3.0, colText)
	g.drawTextSpaced("SELECT OPERATING MODE", rec.X+27, rec.Y+56, 11, 2.0, colMuted)
	closeText := "C / ESC  CLOSE"
	cw := g.textWidthSpaced(closeText, 11, 1.4)
	g.drawTextSpaced(closeText, rec.X+rec.Width-26-cw, rec.Y+28, 11, 1.4, colMuted)
	rl.DrawLineEx(rl2(rec.X+26, rec.Y+80), rl2(rec.X+rec.Width-26, rec.Y+80), 1, colBorder)

	mouse := g.uiMouse()
	for i := 0; i < int(toolCount); i++ {
		d := toolDefs[i]
		row := g.toolMenuRow(i)
		selected := ToolID(i) == g.toolMenuSel
		equipped := ToolID(i) == g.tool
		hover := rl.CheckCollisionPointRec(mouse, row)

		fill := colPanelSoft
		border := colBorder
		if hover {
			fill = colPanelHover
			border = colBorderHi
		}
		if selected {
			fill = colRowSelected
			border = d.Color
		}
		rl.DrawRectangleRec(row, fill)
		rl.DrawRectangleLinesEx(row, 1, border)
		if selected {
			g.accentBar(rl.Rectangle{X: row.X, Y: row.Y, Width: 3, Height: row.Height}, d.Color)
		}

		icon := rl.Rectangle{X: row.X + 10, Y: row.Y + 9, Width: 32, Height: 32}
		rl.DrawRectangleRec(icon, rgba(0, 0, 0, 90))
		rl.DrawRectangleLinesEx(icon, 1, withAlpha(d.Color, 120))
		g.drawToolIcon(ToolID(i), icon.X+icon.Width*0.5, icon.Y+icon.Height*0.5, 19, d.Color)

		numX := row.X + 54
		g.drawTextSpaced(itoa(i+1), numX, row.Y+9, 12, 1.0, withAlpha(colMuted, 190))
		g.drawTextSpaced(d.Name, numX+20, row.Y+7, 16, 1.8, colText)
		g.drawTextSpaced(d.Tag, numX+20, row.Y+28, 10.5, 1.4, withAlpha(d.Color, 210))

		if equipped {
			g.drawTextSpaced("ACTIVE", row.X+row.Width-78, row.Y+18, 10, 1.4, d.Color)
		}

		px := row.X + row.Width*0.46
		g.drawTextSpaced("LMB  "+d.Primary, px, row.Y+9, 11.5, 1.2, withAlpha(colText, 220))
		g.drawTextSpaced("RMB  "+d.Secondary, px, row.Y+28, 11.5, 1.2, withAlpha(colMuted, 220))
	}

	d := toolDefs[g.toolMenuSel]
	equip, close := g.toolMenuButtons()
	g.drawMenuButton(equip, "EQUIP "+d.Name, false)
	g.drawMenuButton(close, "CLOSE", true)

	hint := "WHEEL PREVIEW   //   1-7 QUICK SELECT   //   " + d.Primary
	g.drawTextSpaced(hint, rec.X+252, rec.Y+rec.Height-43, 10, 1.0, withAlpha(colMuted, 205))
}

// drawToolIcon paints a compact technical glyph for a tool mode.
func (g *Game) drawToolIcon(t ToolID, cx, cy, size float32, c rl.Color) {
	s := size
	switch t {
	case toolPhysgun:
		rl.DrawLineEx(rl2(cx-0.52*s, cy-0.28*s), rl2(cx+0.20*s, cy-0.02*s), 2.4, c)
		rl.DrawLineEx(rl2(cx-0.52*s, cy-0.28*s), rl2(cx-0.10*s, cy+0.40*s), 2.4, c)
		rl.DrawCircleV(rl2(cx+0.26*s, cy+0.02*s), 0.18*s, c)
		rl.DrawCircleLines(int32(cx+0.26*s), int32(cy+0.02*s), 0.32*s, withAlpha(c, 140))
	case toolSpawner:
		g.stroke(rl.Rectangle{X: cx - 0.36*s, Y: cy - 0.36*s, Width: 0.56 * s, Height: 0.56 * s}, c)
		rl.DrawRectangleRec(rl.Rectangle{X: cx - 0.28*s, Y: cy - 0.28*s, Width: 0.40 * s, Height: 0.40 * s}, withAlpha(c, 50))
		rl.DrawLineEx(rl2(cx+0.36*s, cy-0.30*s), rl2(cx+0.36*s, cy+0.10*s), 1.8, c)
		rl.DrawLineEx(rl2(cx+0.16*s, cy-0.10*s), rl2(cx+0.56*s, cy-0.10*s), 1.8, c)
	case toolRemover:
		rl.DrawLineEx(rl2(cx-0.36*s, cy-0.36*s), rl2(cx+0.36*s, cy+0.36*s), 2.6, c)
		rl.DrawLineEx(rl2(cx+0.36*s, cy-0.36*s), rl2(cx-0.36*s, cy+0.36*s), 2.6, c)
	case toolImpact:
		rl.DrawCircleLines(int32(cx), int32(cy), 0.18*s, c)
		rl.DrawCircleLines(int32(cx), int32(cy), 0.34*s, withAlpha(c, 190))
		rl.DrawCircleLines(int32(cx), int32(cy), 0.50*s, withAlpha(c, 110))
	case toolFreeze:
		for i := 0; i < 3; i++ {
			a := float64(i)*math.Pi/3 + math.Pi/2
			dx := float32(math.Cos(a)) * 0.46 * s
			dy := float32(math.Sin(a)) * 0.46 * s
			rl.DrawLineEx(rl2(cx-dx, cy-dy), rl2(cx+dx, cy+dy), 1.8, c)
		}
		rl.DrawCircleV(rl2(cx, cy), 0.08*s, c)
	case toolDuplicator:
		g.stroke(rl.Rectangle{X: cx - 0.44*s, Y: cy - 0.38*s, Width: 0.44 * s, Height: 0.44 * s}, withAlpha(c, 150))
		g.stroke(rl.Rectangle{X: cx - 0.10*s, Y: cy - 0.08*s, Width: 0.48 * s, Height: 0.48 * s}, c)
	case toolGravity:
		rl.DrawLineEx(rl2(cx, cy-0.46*s), rl2(cx, cy+0.30*s), 2.0, c)
		rl.DrawLineEx(rl2(cx, cy+0.40*s), rl2(cx-0.22*s, cy+0.12*s), 2.0, c)
		rl.DrawLineEx(rl2(cx, cy+0.40*s), rl2(cx+0.22*s, cy+0.12*s), 2.0, c)
		rl.DrawLineEx(rl2(cx-0.42*s, cy-0.40*s), rl2(cx+0.42*s, cy-0.40*s), 1.2, withAlpha(c, 140))
	}
}

// ---------------------------------------------------------------------------
// Tool gun viewmodel
// ---------------------------------------------------------------------------

func (g *Game) viewBasis() (right, up, forward rl.Vector3) {
	forward = g.cameraForward()
	right = rlv(float32(math.Cos(float64(g.player.yaw))), 0, float32(math.Sin(float64(g.player.yaw))))
	up = rlv(
		right.Y*forward.Z-right.Z*forward.Y,
		right.Z*forward.X-right.X*forward.Z,
		right.X*forward.Y-right.Y*forward.X,
	)
	return right, up, forward
}

func (g *Game) viewPoint(x, y, z float32) rl.Vector3 {
	r, u, f := g.viewBasis()
	c := g.camera.Position
	return rlv(
		c.X+r.X*x+u.X*y+f.X*z,
		c.Y+r.Y*x+u.Y*y+f.Y*z,
		c.Z+r.Z*x+u.Z*y+f.Z*z,
	)
}

func quatFromBasis(xAxis, yAxis, zAxis rl.Vector3) box3d.Quat {
	m00, m01, m02 := xAxis.X, yAxis.X, zAxis.X
	m10, m11, m12 := xAxis.Y, yAxis.Y, zAxis.Y
	m20, m21, m22 := xAxis.Z, yAxis.Z, zAxis.Z
	trace := m00 + m11 + m22
	var q box3d.Quat
	switch {
	case trace > 0:
		s := float32(math.Sqrt(float64(trace+1))) * 2
		q.S = 0.25 * s
		q.V.X = (m21 - m12) / s
		q.V.Y = (m02 - m20) / s
		q.V.Z = (m10 - m01) / s
	case m00 > m11 && m00 > m22:
		s := float32(math.Sqrt(float64(1+m00-m11-m22))) * 2
		q.S = (m21 - m12) / s
		q.V.X = 0.25 * s
		q.V.Y = (m01 + m10) / s
		q.V.Z = (m02 + m20) / s
	case m11 > m22:
		s := float32(math.Sqrt(float64(1+m11-m00-m22))) * 2
		q.S = (m02 - m20) / s
		q.V.X = (m01 + m10) / s
		q.V.Y = 0.25 * s
		q.V.Z = (m12 + m21) / s
	default:
		s := float32(math.Sqrt(float64(1+m22-m00-m11))) * 2
		q.S = (m10 - m01) / s
		q.V.X = (m02 + m20) / s
		q.V.Y = (m12 + m21) / s
		q.V.Z = 0.25 * s
	}
	return normalizeQuat(q)
}

func (g *Game) drawViewBox(center, half rl.Vector3, c, edge rl.Color) {
	r, u, f := g.viewBasis()
	q := quatFromBasis(r, u, rlv(-f.X, -f.Y, -f.Z))
	world := g.viewPoint(center.X, center.Y, center.Z)
	axis, angle := quatAxisAngle(q)
	rl.DrawModelEx(g.cube, world, axis, angle, rlv(half.X*2, half.Y*2, half.Z*2), c)
	drawBoxEdges(world, q, toBv(half), edge)
}

func (g *Game) muzzleWorld() rl.Vector3 {
	kick := g.toolKick * 0.055
	return g.viewPoint(0.27, -0.165, 0.82-kick)
}

func (g *Game) drawViewmodel() {
	if g.player == nil {
		return
	}
	def := g.toolDef()
	swayX := float32(math.Sin(float64(g.toolBob))) * 0.0045
	swayY := float32(math.Cos(float64(g.toolBob*1.7))) * 0.0035
	kick := g.toolKick * 0.055
	bobY := float32(math.Sin(float64(g.toolBob*2))) * 0.004

	x := 0.27 + swayX
	y := -0.19 + swayY + bobY
	z := 0.55 - kick

	metal := rgb(36, 38, 41)
	metalHi := rgb(56, 59, 63)
	metalDark := rgb(22, 23, 25)
	edge := rgba(130, 136, 142, 95)
	hot := def.Color

	// Receiver, rail, barrel and muzzle shroud.
	g.drawViewBox(rlv(x, y, z), rlv(0.046, 0.046, 0.150), metal, edge)
	g.drawViewBox(rlv(x, y+0.052, z-0.010), rlv(0.026, 0.010, 0.110), metalHi, edge)
	g.drawViewBox(rlv(x, y+0.002, z+0.200), rlv(0.020, 0.020, 0.075), metalDark, edge)
	g.drawViewBox(rlv(x, y+0.002, z+0.278), rlv(0.029, 0.029, 0.014), hot, withAlpha(hot, 180))
	g.drawViewBox(rlv(x, y+0.070, z-0.075), rlv(0.019, 0.012, 0.016), metalDark, edge)

	// Grip and energy cell.
	g.drawViewBox(rlv(x+0.022, y-0.098, z-0.070), rlv(0.028, 0.074, 0.038), metal, edge)
	g.drawViewBox(rlv(x-0.052, y-0.010, z-0.020), rlv(0.014, 0.032, 0.052), hot, withAlpha(hot, 150))
	g.drawViewBox(rlv(x+0.050, y+0.004, z-0.010), rlv(0.007, 0.023, 0.062), withAlpha(hot, 190), withAlpha(hot, 90))

	// Muzzle flash.
	if g.toolKick > 0.15 {
		m := g.muzzleWorld()
		a := uint8(255 * clampf(g.toolKick, 0, 1))
		rl.DrawSphereEx(m, 0.018+0.035*g.toolKick, 5, 7, withAlpha(hot, a))
		r, u, f := g.viewBasis()
		for i := 0; i < 3; i++ {
			ang := float64(i) * math.Pi / 3
			dx := float32(math.Cos(ang))
			dy := float32(math.Sin(ang))
			off := rlv(r.X*dx+u.X*dy, r.Y*dx+u.Y*dy, r.Z*dx+u.Z*dy)
			spread := 0.05 + 0.07*g.toolKick
			rl.DrawLine3D(m, rlv(m.X+off.X*spread+f.X*0.02, m.Y+off.Y*spread+f.Y*0.02, m.Z+off.Z*spread+f.Z*0.02), withAlpha(hot, a))
		}
	}
}
