package game

import (
	"fmt"
	"strings"

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
	const count = 6
	top := rec.Y + 112
	avail := rec.Y + rec.Height - 20 - top
	if avail < 0 {
		avail = 0
	}
	gap := float32(10)
	bh := (avail - gap*float32(count-1)) / float32(count)
	if bh > 44 {
		bh = 44
	}
	if bh < 22 {
		bh = 22
	}
	gap = (avail - bh*float32(count)) / float32(count-1)
	if gap > 12 {
		gap = 12
	}
	if gap < 4 {
		gap = 4
	}
	return rl.Rectangle{
		X:      rec.X + 30,
		Y:      top + float32(index)*(bh+gap),
		Width:  rec.Width - 60,
		Height: bh,
	}
}

func (g *Game) updateMainMenuInput() {
	mouse := g.uiMouse()
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	for i := 0; i < 6; i++ {
		if rl.CheckCollisionPointRec(mouse, g.mainMenuButton(i)) {
			switch i {
			case 0: // Play
				maps := listMaps()
				if len(maps) == 0 {
					g.toast("NO MAPS FOUND // ADD ONE TO Content/Maps", colDanger)
					return
				}
				if g.selectedMap == "" {
					g.selectedMap = maps[0]
				}
				g.resetWorld(true, defaultSpawn, 0, -0.12, false)
				g.state = statePlaying
				g.resumePlaying()
			case 1: // Multiplayer
				g.netStatus = ""
				g.netField = 1
				g.state = stateNetMenu
			case 2: // Maps
				g.state = stateMapMenu
			case 3: // Settings
				g.openSettings()
			case 4: // Addons
				g.modsScroll = 0
				g.state = stateModsMenu
			case 5: // Quit
				g.quitRequested = true
			}
			return
		}
	}
}

func (g *Game) drawMainMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 255))
	rec := g.mainMenuPanel()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.textCenteredSpaced("FSBOX", rec.X+rec.Width*0.5, rec.Y+30, 42, 6.0, colAccent)
	g.textCenteredSpaced("3D PHYSICS SANDBOX", rec.X+rec.Width*0.5, rec.Y+80, 12, 2.0, colMuted)

	labels := []string{"PLAY", "MULTIPLAYER", "MAPS", "SETTINGS", "ADDONS", "QUIT"}
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
		if len(maps) == 0 {
			g.toast("NO MAPS FOUND // ADD ONE TO Content/Maps", colDanger)
			return
		}
		if g.selectedMap == "" {
			g.selectedMap = maps[0]
		}
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
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 255))
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

// ---------------------------------------------------------------------------
// Addons menu
// ---------------------------------------------------------------------------

const (
	modRowH = 92.0
	modGap  = 8.0
)

func (g *Game) modsPanel() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	pw := minf(820, w-80)
	ph := minf(h-80, 660)
	return rl.Rectangle{X: (w - pw) * 0.5, Y: (h - ph) * 0.5, Width: pw, Height: ph}
}

func (g *Game) modsListRect() rl.Rectangle {
	panel := g.modsPanel()
	return rl.Rectangle{
		X:      panel.X + 24,
		Y:      panel.Y + 104,
		Width:  panel.Width - 48,
		Height: panel.Height - 104 - 70,
	}
}

func (g *Game) modsBackRect() rl.Rectangle {
	panel := g.modsPanel()
	return rl.Rectangle{
		X:      panel.X + panel.Width - 170,
		Y:      panel.Y + panel.Height - 54,
		Width:  140,
		Height: 36,
	}
}

func (g *Game) loadedMods() []*Addon {
	if g.addons == nil {
		return nil
	}
	return g.addons.Addons
}

func (g *Game) modsVisibleRows() int {
	list := g.modsListRect()
	n := int((list.Height + modGap) / (modRowH + modGap))
	if n < 1 {
		n = 1
	}
	return n
}

func (g *Game) clampModsScroll() {
	maxScroll := len(g.loadedMods()) - g.modsVisibleRows()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.modsScroll > maxScroll {
		g.modsScroll = maxScroll
	}
	if g.modsScroll < 0 {
		g.modsScroll = 0
	}
}

func (g *Game) updateModsMenuInput() {
	if rl.IsKeyPressed(rl.KeyEscape) {
		g.state = stateMainMenu
		return
	}
	if wheel := rl.GetMouseWheelMove(); wheel != 0 {
		if wheel > 0 {
			g.modsScroll--
		} else {
			g.modsScroll++
		}
		g.clampModsScroll()
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && rl.CheckCollisionPointRec(g.uiMouse(), g.modsBackRect()) {
		g.state = stateMainMenu
	}
}

func (g *Game) drawModsMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 255))

	panel := g.modsPanel()
	g.panel(panel, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: panel.X, Y: panel.Y, Width: 3, Height: panel.Height}, colAccent)

	addons := g.loadedMods()
	g.textCenteredSpaced("ADDONS", panel.X+panel.Width*0.5, panel.Y+26, 26, 4.0, colText)
	g.textCenteredSpaced(
		fmt.Sprintf("%02d LOADED  /  NAME, AUTHOR, VERSION", len(addons)),
		panel.X+panel.Width*0.5, panel.Y+64, 10, 1.4, colMuted,
	)
	g.drawSectionLabel("INSTALLED CONTENT", panel.X+24, panel.Y+84, colMuted)

	list := g.modsListRect()
	g.clampModsScroll()
	if len(addons) == 0 {
		g.textCenteredSpaced("NO MODS LOADED", list.X+list.Width*0.5, list.Y+list.Height*0.5-6, 12, 2.0, colMuted)
	} else {
		mouse := g.uiMouse()
		visible := g.modsVisibleRows()
		for i := 0; i < visible; i++ {
			idx := g.modsScroll + i
			if idx >= len(addons) {
				break
			}
			m := addons[idx]
			row := rl.Rectangle{
				X:      list.X,
				Y:      list.Y + float32(i)*(modRowH+modGap),
				Width:  list.Width,
				Height: modRowH,
			}
			hover := rl.CheckCollisionPointRec(mouse, row)
			fill := colPanelSoft
			border := colBorder
			if hover {
				fill = colPanelHover
				border = colBorderHi
			}
			rl.DrawRectangleRec(row, fill)
			rl.DrawRectangleLinesEx(row, 1, border)
			g.accentBar(rl.Rectangle{X: row.X, Y: row.Y, Width: 2, Height: row.Height}, colAccent)

			name := m.Manifest.Name
			if name == "" {
				name = m.Manifest.ID
			}
			if name == "" {
				name = m.Name
			}
			version := m.Manifest.Version
			if version == "" {
				version = "0.0.0"
			}
			author := m.Manifest.Author
			if author == "" {
				author = "unknown"
			}
			desc := m.Manifest.Description
			if desc == "" {
				desc = "No description provided."
			}

			g.drawTextSpaced(truncateRunes(strings.ToUpper(name), 56), row.X+14, row.Y+12, 15, 1.4, colText)
			verLabel := "v" + version
			vw := g.textWidthSpaced(verLabel, 10, 0.8)
			g.drawTextSpaced(verLabel, row.X+row.Width-14-vw, row.Y+14, 10, 0.8, colAccent)
			g.drawTextSpaced("BY "+strings.ToUpper(author), row.X+14, row.Y+34, 9.5, 1.0, colMuted)
			g.drawWrappedText(truncateRunes(desc, 220), row.X+14, row.Y+52, row.Width-28, 10.5, 0.9, 14, withAlpha(colText, 210))
		}
	}

	g.drawTextSpaced("WHEEL  SCROLL     ESC  BACK", panel.X+24, panel.Y+panel.Height-44, 9.5, 1.0, withAlpha(colMuted, 200))
	g.drawMenuButton(g.modsBackRect(), "BACK", true)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
