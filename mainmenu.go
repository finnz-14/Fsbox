package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

func (g *Game) mainMenuPanel() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	pw := minf(420, w-80)
	ph := minf(520, h-80)
	return rl.Rectangle{X: (w - pw) * 0.5, Y: (h - ph) * 0.5, Width: pw, Height: ph}
}

func (g *Game) mainMenuButton(index int) rl.Rectangle {
	rec := g.mainMenuPanel()
	bw := rec.Width - 60
	bh := float32(44)
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 120 + float32(index)*(bh+12), Width: bw, Height: bh}
}

func (g *Game) updateMainMenuInput() {
	mouse := g.uiMouse()
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	for i := 0; i < 4; i++ {
		if rl.CheckCollisionPointRec(mouse, g.mainMenuButton(i)) {
			switch i {
			case 0: // Play
				g.selectedMap = "default"
				if len(listMaps()) > 0 {
					g.selectedMap = listMaps()[0]
				}
				g.resetWorld(true, defaultSpawn, 0, -0.12, false)
				g.state = statePlaying
				g.resumePlaying()
			case 1: // Maps
				g.state = stateMapMenu
			case 2: // Settings
				g.openSettings()
			case 3: // Quit
				g.quitRequested = true
			}
			return
		}
	}
}

func (g *Game) drawMainMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 230))
	rec := g.mainMenuPanel()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.textCenteredSpaced("FSBOX", rec.X+rec.Width*0.5, rec.Y+30, 42, 6.0, colAccent)
	g.textCenteredSpaced("3D PHYSICS SANDBOX", rec.X+rec.Width*0.5, rec.Y+80, 12, 2.0, colMuted)

	labels := []string{"PLAY", "MAPS", "SETTINGS", "QUIT"}
	for i, lab := range labels {
		btn := g.mainMenuButton(i)
		hover := rl.CheckCollisionPointRec(g.uiMouse(), btn)
		fill := colPanelSoft
		if hover {
			fill = colPanelHover
		}
		rl.DrawRectangleRec(btn, fill)
		rl.DrawRectangleLinesEx(btn, 1, colBorderHi)
		g.textCentered(lab, btn.X+btn.Width*0.5, btn.Y+(btn.Height-14)*0.5, 14, colText)
	}
}

// ---------------------------------------------------------------------------
// Map menu
// ---------------------------------------------------------------------------

func (g *Game) updateMapMenuInput() {
	mouse := g.uiMouse()
	if rl.IsKeyPressed(rl.KeyEscape) {
		g.state = stateMainMenu
		return
	}
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	maps := listMaps()
	// map entries
	panel := g.mainMenuPanel()
	for i, m := range maps {
		row := rl.Rectangle{X: panel.X + 30, Y: panel.Y + 120 + float32(i)*40, Width: panel.Width - 60, Height: 34}
		if rl.CheckCollisionPointRec(mouse, row) {
			g.selectedMap = m
			return
		}
	}
	// start button
	startBtn := rl.Rectangle{X: panel.X + 30, Y: panel.Y + panel.Height - 54, Width: 140, Height: 36}
	if rl.CheckCollisionPointRec(mouse, startBtn) {
		g.resetWorld(true, defaultSpawn, 0, -0.12, false)
		g.state = statePlaying
		g.resumePlaying()
		return
	}
	// back button
	backBtn := rl.Rectangle{X: panel.X + panel.Width - 170, Y: panel.Y + panel.Height - 54, Width: 140, Height: 36}
	if rl.CheckCollisionPointRec(mouse, backBtn) {
		g.state = stateMainMenu
	}
}

func (g *Game) drawMapMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 230))
	panel := g.mainMenuPanel()
	g.panel(panel, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: panel.X, Y: panel.Y, Width: 3, Height: panel.Height}, colAccent)
	g.textCenteredSpaced("SELECT MAP", panel.X+panel.Width*0.5, panel.Y+30, 22, 3.0, colText)

	maps := listMaps()
	mouse := g.uiMouse()
	for i, m := range maps {
		row := rl.Rectangle{X: panel.X + 30, Y: panel.Y + 120 + float32(i)*40, Width: panel.Width - 60, Height: 34}
		hover := rl.CheckCollisionPointRec(mouse, row)
		fill := colPanelSoft
		if hover {
			fill = colPanelHover
		}
		if m == g.selectedMap {
			fill = colRowSelected
		}
		rl.DrawRectangleRec(row, fill)
		rl.DrawRectangleLinesEx(row, 1, colBorderHi)
		g.drawText(m, row.X+12, row.Y+10, 13, colText)
	}

	// start button
	startBtn := rl.Rectangle{X: panel.X + 30, Y: panel.Y + panel.Height - 54, Width: 140, Height: 36}
	g.drawMenuButton(startBtn, "START", false)
	// back button
	backBtn := rl.Rectangle{X: panel.X + panel.Width - 170, Y: panel.Y + panel.Height - 54, Width: 140, Height: 36}
	g.drawMenuButton(backBtn, "BACK", true)
}