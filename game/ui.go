package game

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ---------------------------------------------------------------------------
// Text and panel primitives
// ---------------------------------------------------------------------------

func (g *Game) drawText(text string, x, y, size float32, c rl.Color) {
	rl.DrawTextEx(g.font, text, rl.Vector2{X: x, Y: y}, size, 1.0, c)
}

func (g *Game) drawTextSpaced(text string, x, y, size, spacing float32, c rl.Color) {
	rl.DrawTextEx(g.font, text, rl.Vector2{X: x, Y: y}, size, spacing, c)
}

func (g *Game) textWidth(text string, size float32) float32 {
	return rl.MeasureTextEx(g.font, text, size, 1.0).X
}

func (g *Game) textWidthSpaced(text string, size, spacing float32) float32 {
	return rl.MeasureTextEx(g.font, text, size, spacing).X
}

func (g *Game) textCentered(text string, cx, y, size float32, c rl.Color) {
	g.drawText(text, cx-g.textWidth(text, size)*0.5, y, size, c)
}

func (g *Game) textCenteredSpaced(text string, cx, y, size, spacing float32, c rl.Color) {
	g.drawTextSpaced(text, cx-g.textWidthSpaced(text, size, spacing)*0.5, y, size, spacing, c)
}

// panel is deliberately square: one flat fill and a one-pixel border.
func (g *Game) panel(rec rl.Rectangle, fill, border rl.Color) {
	rl.DrawRectangleRec(rec, fill)
	rl.DrawRectangleLinesEx(rec, 1, border)
}

func (g *Game) panelAccent(rec rl.Rectangle, fill, border rl.Color) {
	g.panel(rec, fill, border)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, border)
}

func (g *Game) accentBar(rec rl.Rectangle, c rl.Color) {
	rl.DrawRectangleRec(rec, c)
}

func (g *Game) stroke(rec rl.Rectangle, c rl.Color) {
	rl.DrawRectangleLinesEx(rec, 1, c)
}

// tagged text: a small coloured tick followed by an uppercase label.
func (g *Game) drawSectionLabel(text string, x, y float32, c rl.Color) {
	rl.DrawRectangle(int32(x), int32(y+4), 10, 2, c)
	g.drawTextSpaced(text, x+17, y, 10.5, 1.8, c)
}

func (g *Game) drawDivider(x0, x1, y float32, c rl.Color) {
	rl.DrawLineEx(rl2(x0, y), rl2(x1, y), 1, c)
}

func (g *Game) drawMenuButton(rec rl.Rectangle, label string, secondary bool) {
	hover := rl.CheckCollisionPointRec(g.uiMouse(), rec)
	fill := colAccent
	border := colAccent
	text := colDark
	if secondary {
		fill = colPanelSoft
		border = colBorderHi
		text = colText
		if hover {
			fill = colPanelHover
			border = colAccent
			text = colAccentHi
		}
	} else if hover {
		fill = colAccentHi
		border = colAccentHi
	}
	rl.DrawRectangleRec(rec, fill)
	rl.DrawRectangleLinesEx(rec, 1, border)
	// Left tick gives every button the same technical cadence.
	tick := text
	if !secondary {
		tick = colDark
	}
	rl.DrawRectangle(int32(rec.X+6), int32(rec.Y+rec.Height*0.5-1), 8, 2, tick)
	g.textCenteredSpaced(label, rec.X+rec.Width*0.5+5, rec.Y+(rec.Height-14)*0.5, 12, 1.6, text)
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func (g *Game) drawKeyRow(x, y float32, key, desc string) {
	const keySize = 11.5
	kw := g.textWidthSpaced(key, keySize, 1.1) + 18
	rec := rl.Rectangle{X: x, Y: y, Width: kw, Height: 22}
	rl.DrawRectangleRec(rec, rgba(255, 255, 255, 10))
	rl.DrawRectangleLinesEx(rec, 1, rgba(200, 204, 208, 52))
	g.drawTextSpaced(key, x+9, y+5, keySize, 1.1, colAccent)
	g.drawTextSpaced(desc, x+kw+14, y+4, 12.5, 0.5, withAlpha(colText, 225))
}

// ---------------------------------------------------------------------------
// HUD
// ---------------------------------------------------------------------------

func (g *Game) drawHUD() {
	w := g.uiWidth()
	h := g.uiHeight()

	g.drawHeader()
	g.drawStats(w)
	if g.state == statePlaying {
		g.drawCrosshair(w, h)
	}
	g.drawToolStatus(w, h)
	g.drawControlHints(w, h)
	g.drawSpawnBuffer(w, h)
}

func (g *Game) drawHeader() {
	rec := rl.Rectangle{X: 18, Y: 18, Width: 334, Height: 66}
	g.panel(rec, colPanel, colBorder)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.drawTextSpaced("FSBOX", rec.X+18, rec.Y+13, 22, 3.6, colText)
	g.drawTextSpaced("FIELD SANDBOX  /  BUILD 01", rec.X+19, rec.Y+42, 10, 1.7, colMuted)

	// Small status cluster on the right edge.
	rl.DrawRectangle(int32(rec.X+rec.Width-15), int32(rec.Y+13), 6, 6, colAccent)
	g.drawTextSpaced("LIVE", rec.X+rec.Width-42, rec.Y+10, 9.5, 1.0, colAccentDim)
}

func (g *Game) drawStats(w float32) {
	rec := rl.Rectangle{X: w - 248, Y: 18, Width: 230, Height: 200}
	g.panel(rec, colPanel, colBorder)
	g.drawSectionLabel("TELEMETRY", rec.X+14, rec.Y+12, colMuted)

	mode := "WALK"
	modeColor := colText
	if g.player != nil && g.player.fly {
		mode = "FLY"
		modeColor = colEnergy
	}
	rows := []struct {
		label string
		value string
		color rl.Color
	}{
		{"FPS", fmt.Sprintf("%d", rl.GetFPS()), colText},
		{"BODIES", fmt.Sprintf("%d", len(g.entities)), colText},
		{"PROPS", fmt.Sprintf("%d", len(g.props)), colText},
		{"FROZEN", fmt.Sprintf("%d", g.frozenCount()), colFreeze},
		{"GRAV", fmt.Sprintf("%.2fx", g.settings.Gravity), colEnergy},
		{"MODE", mode, modeColor},
	}
	toolRowY := rec.Y + 37
	g.drawTextSpaced("TOOL", rec.X+14, toolRowY, 10, 1.2, colMuted)
	tv := g.toolDef()
	valW := g.textWidthSpaced(tv.Name, 11, 1.4)
	g.drawTextSpaced(tv.Name, rec.X+rec.Width-14-valW, toolRowY-1, 11, 1.4, tv.Color)

	for i, row := range rows {
		y := rec.Y + 64 + float32(i)*22
		g.drawTextSpaced(row.label, rec.X+14, y, 10, 1.2, colMuted)
		vw := g.textWidthSpaced(row.value, 12, 1.0)
		g.drawTextSpaced(row.value, rec.X+rec.Width-14-vw, y-1, 12, 1.0, row.color)
	}
}

func (g *Game) drawCrosshair(w, h float32) {
	cx, cy := w*0.5, h*0.5
	c := withAlpha(colText, 190)
	if g.aim != nil || g.held != nil {
		c = g.toolColor()
	}

	const gap = 6.0
	const length = 9.0
	thick := float32(1.3)
	rl.DrawLineEx(rl2(cx-gap-length, cy), rl2(cx-gap, cy), thick, c)
	rl.DrawLineEx(rl2(cx+gap, cy), rl2(cx+gap+length, cy), thick, c)
	rl.DrawLineEx(rl2(cx, cy-gap-length), rl2(cx, cy-gap), thick, c)
	rl.DrawLineEx(rl2(cx, cy+gap), rl2(cx, cy+gap+length), thick, c)
	rl.DrawRectangle(int32(cx)-1, int32(cy)-1, 2, 2, c)

	if g.held != nil {
		rl.DrawCircleLines(int32(cx), int32(cy), 13, withAlpha(g.toolColor(), 190))
		rl.DrawCircleLines(int32(cx), int32(cy), 17, withAlpha(g.toolColor(), 70))
	}
	if g.aim != nil && g.aim != g.held {
		dist := lenRl(rlv(g.aimPoint.X-g.camera.Position.X, g.aimPoint.Y-g.camera.Position.Y, g.aimPoint.Z-g.camera.Position.Z))
		label := strings.ToUpper(g.aim.Name)
		if g.aim.Frozen {
			label += "  [FROZEN]"
		}
		g.textCenteredSpaced(label, cx, cy+25, 11, 1.4, c)
		g.textCenteredSpaced(fmt.Sprintf("%.1f M", dist), cx, cy+41, 10, 1.2, withAlpha(colMuted, 215))
	}
}

func (g *Game) drawToolStatus(w, h float32) {
	def := g.toolDef()
	rec := rl.Rectangle{X: 18, Y: h - 144, Width: 348, Height: 126}
	g.panel(rec, colPanel, colBorder)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, def.Color)

	g.drawTextSpaced(fmt.Sprintf("ACTIVE TOOL  /  %02d", int(g.tool)+1), rec.X+16, rec.Y+12, 10, 1.6, colMuted)
	g.drawTextSpaced(def.Name, rec.X+16, rec.Y+27, 19, 2.6, colText)
	g.drawTextSpaced(def.Tag, rec.X+rec.Width-14-g.textWidthSpaced(def.Tag, 9.5, 1.4), rec.Y+14, 9.5, 1.4, withAlpha(def.Color, 210))

	g.drawTextSpaced("LMB", rec.X+16, rec.Y+60, 9.5, 1.0, def.Color)
	g.drawTextSpaced(def.Primary, rec.X+50, rec.Y+58, 11.5, 0.8, withAlpha(colText, 225))
	g.drawTextSpaced("RMB", rec.X+16, rec.Y+80, 9.5, 1.0, colMuted)
	g.drawTextSpaced(def.Secondary, rec.X+50, rec.Y+78, 11.5, 0.8, withAlpha(colMuted, 225))

	g.drawTextSpaced("RANGE", rec.X+16, rec.Y+102, 9.5, 1.0, def.Color)
	g.drawTextSpaced(fmt.Sprintf("%.1f M", g.placementDistance()), rec.X+50, rec.Y+100, 11, 0.8, withAlpha(colText, 225))
	wheelHint := "WHEEL"
	wh := g.textWidthSpaced(wheelHint, 8.5, 1.0)
	g.drawTextSpaced(wheelHint, rec.X+rec.Width-14-wh, rec.Y+102, 8.5, 1.0, withAlpha(colMuted, 180))

	if g.held != nil {
		label := "FIELD LOCK  //  " + strings.ToUpper(g.held.Name)
		g.drawTextSpaced(label, rec.X+rec.Width+14, rec.Y+rec.Height-16, 10, 1.2, colAccent)
	}
}

func (g *Game) drawControlHints(w, h float32) {
	lines := []string{
		"E  SPAWN    C  TOOLS    1-7  MODE",
		"WHEEL  DISTANCE    CTRL+WHEEL  TOOL",
		"F  FREEZE    LMB  GRAB    V  FLY    F1  MANUAL",
		"HOLD E / R + MOUSE  ROTATE PROP",
		"F5  SAVE    F9  LOAD    F2  SETTINGS",
	}
	x := w - 18
	for i, line := range lines {
		lw := g.textWidthSpaced(line, 10, 1.2)
		g.drawTextSpaced(line, x-lw, h-102+float32(i)*18, 10, 1.2, withAlpha(colMuted, 195))
	}
}

func (g *Game) drawSpawnBuffer(w, h float32) {
	p := presets[g.selected]
	label := "SPAWN BUFFER"
	name := strings.ToUpper(p.Name)
	lw := g.textWidthSpaced(label, 9, 1.4)
	nw := g.textWidthSpaced(name, 11.5, 1.4)
	total := lw + 20 + nw + 30
	rec := rl.Rectangle{X: w*0.5 - total*0.5, Y: h - 50, Width: total, Height: 28}
	rl.DrawRectangleRec(rec, rgba(10, 11, 12, 190))
	rl.DrawRectangleLinesEx(rec, 1, colBorder)
	g.drawTextSpaced(label, rec.X+11, rec.Y+8, 9, 1.4, withAlpha(colMuted, 215))
	rl.DrawCircleV(rl2(rec.X+lw+20, rec.Y+14), 3.5, p.Color)
	g.drawTextSpaced(name, rec.X+lw+29, rec.Y+6, 11.5, 1.4, withAlpha(colText, 225))
	if g.tool != toolSpawner {
		g.drawTextSpaced("EQUIP SPAWNER TO PLACE", rec.X+total+10, rec.Y+8, 9, 1.2, withAlpha(colMuted, 140))
	}
}

// ---------------------------------------------------------------------------
// Pause / help
// ---------------------------------------------------------------------------

func (g *Game) helpPanelRect() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	pw := minf(980, w-40)
	ph := minf(660, h-40)
	return rl.Rectangle{X: w*0.5 - pw*0.5, Y: h*0.5 - ph*0.5, Width: pw, Height: ph}
}

func (g *Game) resumeButtonRect() rl.Rectangle {
	rec := g.helpPanelRect()
	return rl.Rectangle{X: rec.X + (rec.Width-260)*0.5, Y: rec.Y + rec.Height - 56, Width: 260, Height: 38}
}

func (g *Game) pausePanelRect() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	const pw, ph = 520.0, 430.0
	return rl.Rectangle{X: w*0.5 - pw*0.5, Y: h*0.5 - ph*0.5, Width: pw, Height: ph}
}

func (g *Game) pauseResumeRect() rl.Rectangle {
	rec := g.pausePanelRect()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 196, Width: rec.Width - 60, Height: 42}
}

func (g *Game) pauseSettingsRect() rl.Rectangle {
	rec := g.pausePanelRect()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 248, Width: (rec.Width - 66) * 0.5, Height: 38}
}

func (g *Game) pauseHelpRect() rl.Rectangle {
	rec := g.pausePanelRect()
	return rl.Rectangle{X: rec.X + 30 + (rec.Width-66)*0.5 + 6, Y: rec.Y + 248, Width: (rec.Width - 66) * 0.5, Height: 38}
}

func (g *Game) pauseMainMenuRect() rl.Rectangle {
	rec := g.pausePanelRect()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 296, Width: rec.Width - 60, Height: 38}
}

func (g *Game) pauseDisconnectRect() rl.Rectangle {
	rec := g.pausePanelRect()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 342, Width: rec.Width - 60, Height: 38}
}

func (g *Game) pauseQuitRect() rl.Rectangle {
	rec := g.pausePanelRect()
	y := rec.Y + 342
	if g.netMode == netModeClient {
		y = rec.Y + 388
	}
	return rl.Rectangle{X: rec.X + 30, Y: y, Width: rec.Width - 60, Height: 38}
}

func (g *Game) drawHelp() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 205))

	rec := g.helpPanelRect()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.drawTextSpaced("FIELD MANUAL", rec.X+36, rec.Y+22, 26, 3.2, colText)
	g.drawTextSpaced("FSBOX OPERATING PROCEDURE  //  REV 02", rec.X+37, rec.Y+57, 10, 1.6, colMuted)
	closeText := "F1 / ESC  CLOSE"
	cw := g.textWidthSpaced(closeText, 9.5, 1.4)
	g.drawTextSpaced(closeText, rec.X+rec.Width-36-cw, rec.Y+32, 9.5, 1.4, colMuted)
	rl.DrawLineEx(rl2(rec.X+36, rec.Y+80), rl2(rec.X+rec.Width-36, rec.Y+80), 1, colBorder)

	left := rec.X + 36
	right := rec.X + 512
	sectionY := rec.Y + 98
	const rowStep = 26.0

	drawSection := func(title string, x, y float32, rows [][2]string) float32 {
		g.drawSectionLabel(title, x, y, colAccent)
		for i, row := range rows {
			g.drawKeyRow(x, y+20+float32(i)*rowStep, row[0], row[1])
		}
		return y + 20 + float32(len(rows))*rowStep
	}

	endLeft := drawSection("MOVEMENT", left, sectionY, [][2]string{
		{"W A S D", "Walk"},
		{"SHIFT", "Sprint"},
		{"SPACE", "Jump"},
		{"V", "Toggle fly mode"},
		{"SPACE / CTRL", "Fly altitude"},
	})
	drawSection("TOOL GUN", left, endLeft+8, [][2]string{
		{"LMB", "Primary tool action"},
		{"RMB", "Secondary tool action"},
		{"WHEEL", "Hold / spawn distance"},
		{"CTRL + WHEEL", "Cycle tool mode"},
		{"1 - 7", "Equip tool mode"},
		{"C", "Open tool selector"},
		{"F", "Freeze / thaw aimed prop"},
		{"HOLD E / R + MOUSE", "Rotate held prop"},
	})

	endRight := drawSection("SPAWNING", right, sectionY, [][2]string{
		{"E / TAB", "Open spawn menu"},
		{"CLICK", "Select prop card"},
		{"ENTER", "Spawn selection"},
		{"DOUBLE CLICK", "Spawn selection"},
		{"SPAWNER LMB", "Place buffered prop"},
	})
	drawSection("SYSTEM", right, endRight+8, [][2]string{
		{"G", "Cycle gravity"},
		{"[ / ]", "Gravity fine tune"},
		{"F5 / F9", "Save / load scene"},
		{"F2", "Settings"},
		{"CTRL + R", "Reset arena"},
		{"CTRL + DEL", "Clear all props"},
	})

	note := "PHYSGUN grabs, throws and pushes. Hold E or R and move the mouse to rotate a held prop. F freezes or thaws any body. DUPLICATOR copies matter. IMPACT moves everything."
	g.drawTextSpaced(note, rec.X+36, rec.Y+rec.Height-88, 9.5, 1.1, withAlpha(colMuted, 215))
	g.drawMenuButton(g.resumeButtonRect(), "RESUME SESSION", false)
}

func (g *Game) drawPause() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 190))

	rec := g.pausePanelRect()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.drawTextSpaced("SYSTEM HOLD", rec.X+30, rec.Y+24, 24, 3.0, colText)
	g.drawTextSpaced("SIMULATION SUSPENDED", rec.X+31, rec.Y+58, 10, 1.6, colMuted)

	rows := [][2]string{
		{"BODIES", fmt.Sprintf("%d", len(g.entities))},
		{"PROPS", fmt.Sprintf("%d", len(g.props))},
		{"FROZEN", fmt.Sprintf("%d", g.frozenCount())},
		{"GRAVITY", fmt.Sprintf("%.2fx", g.settings.Gravity)},
	}
	for i, row := range rows {
		y := rec.Y + 92 + float32(i)*24
		g.drawTextSpaced(row[0], rec.X+30, y, 10, 1.2, colMuted)
		vw := g.textWidthSpaced(row[1], 11.5, 1.0)
		g.drawTextSpaced(row[1], rec.X+rec.Width-30-vw, y, 11.5, 1.0, colText)
	}
	rl.DrawLineEx(rl2(rec.X+30, rec.Y+186), rl2(rec.X+rec.Width-30, rec.Y+186), 1, colBorder)

	g.drawMenuButton(g.pauseResumeRect(), "RESUME SESSION", false)
	g.drawMenuButton(g.pauseSettingsRect(), "SETTINGS", true)
	g.drawMenuButton(g.pauseHelpRect(), "MANUAL", true)
	g.drawMenuButton(g.pauseMainMenuRect(), "MAIN MENU", true)
	if g.netMode == netModeClient {
		g.drawMenuButton(g.pauseDisconnectRect(), "DISCONNECT", true)
	}
	g.drawMenuButton(g.pauseQuitRect(), "QUIT TO DESKTOP", true)
	g.textCenteredSpaced("ESC  RESUME    F1  MANUAL    F5  SAVE    F9  LOAD", rec.X+rec.Width*0.5, rec.Y+rec.Height-24, 9.5, 1.2, withAlpha(colMuted, 180))
}

// ---------------------------------------------------------------------------
// Toasts
// ---------------------------------------------------------------------------

func (g *Game) drawToasts() {
	w := float32(rl.GetScreenWidth())
	for i, t := range g.toasts {
		elapsed := t.MaxLife - t.Life
		a := float32(1)
		if t.Life < 0.55 {
			a = t.Life / 0.55
		}
		if elapsed < 0.12 {
			a = minf(a, elapsed/0.12)
		}
		a = clampf(a, 0, 1)

		size := float32(11)
		tw := g.textWidthSpaced(t.Text, size, 1.2)
		bw := tw + 44
		bh := float32(28)
		x := (w - bw) * 0.5
		y := 22 + float32(i)*36

		rec := rl.Rectangle{X: x, Y: y, Width: bw, Height: bh}
		rl.DrawRectangleRec(rec, rgba(10, 11, 12, uint8(220*a)))
		rl.DrawRectangleLinesEx(rec, 1, withAlpha(t.Color, uint8(105*a)))
		rl.DrawRectangle(int32(x), int32(y), 3, int32(bh), withAlpha(t.Color, uint8(245*a)))
		g.textCenteredSpaced(t.Text, w*0.5, y+8, size, 1.2, withAlpha(t.Color, uint8(255*a)))
	}
}
