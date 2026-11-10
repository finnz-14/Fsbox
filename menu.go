package main

import (
	"fmt"
	"math"
	"strings"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func rl2(x, y float32) rl.Vector2 {
	return rl.Vector2{X: x, Y: y}
}

// ---------------------------------------------------------------------------
// Spawn menu: layout
// ---------------------------------------------------------------------------

const (
	spawnTabProps = iota
	spawnTabTools
	spawnTabWorld
)

var spawnTabs = []string{"SPAWN", "TOOLS", "WORLD"}

var spawnCategories = []string{"ALL", "BLOCKS", "ROUNDS", "CONTAINERS", "HEAVY"}

var worldActions = []struct {
	Name string
	Sub  string
	Desc string
}{
	{"SAVE SCENE", "F5", "Write the current dynamic props and player pose to fsbox_save.json."},
	{"LOAD SCENE", "F9", "Rebuild the arena and restore every prop from the latest save file."},
	{"RESET ARENA", "CTRL+R", "Rebuild the arena and the starter prop layout. The save file is untouched."},
	{"CLEAR PROPS", "CTRL+DEL", "Remove every dynamic prop from the simulation. Frozen props are included."},
	{"SETTINGS", "F2", "Open the system configuration panel for gravity, sensitivity and video."},
}

func presetCategoryName(p ShapePreset) string {
	if p.Category != "" {
		return p.Category
	}
	switch p.Name {
	case "Ball", "Pebble", "Boulder":
		return "ROUNDS"
	case "Barrel", "Column":
		return "CONTAINERS"
	case "Anvil":
		return "HEAVY"
	default:
		return "BLOCKS"
	}
}

func (g *Game) spawnMenuPanel() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	pw := minf(w-64, 1180)
	ph := minf(h-64, 700)
	return rl.Rectangle{X: (w - pw) * 0.5, Y: (h - ph) * 0.5, Width: pw, Height: ph}
}

const (
	spawnRailW   = 210.0
	spawnHeaderH = 60.0
	spawnFooterH = 42.0
	spawnDetailW = 286.0
	spawnPad     = 16.0
)

func (g *Game) spawnRailRect() rl.Rectangle {
	rec := g.spawnMenuPanel()
	return rl.Rectangle{X: rec.X, Y: rec.Y, Width: spawnRailW, Height: rec.Height}
}

func (g *Game) spawnContentX() float32 {
	rec := g.spawnMenuPanel()
	return rec.X + spawnRailW
}

func (g *Game) spawnGridRect() rl.Rectangle {
	rec := g.spawnMenuPanel()
	x := g.spawnContentX() + spawnPad
	y := rec.Y + spawnHeaderH + spawnPad
	w := rec.Width - spawnRailW - spawnDetailW - spawnPad*3
	h := rec.Height - spawnHeaderH - spawnFooterH - spawnPad*2
	return rl.Rectangle{X: x, Y: y, Width: w, Height: h}
}

func (g *Game) spawnDetailRect() rl.Rectangle {
	rec := g.spawnMenuPanel()
	return rl.Rectangle{
		X:      rec.X + rec.Width - spawnDetailW - spawnPad,
		Y:      rec.Y + spawnHeaderH + spawnPad,
		Width:  spawnDetailW,
		Height: rec.Height - spawnHeaderH - spawnFooterH - spawnPad*2,
	}
}

func (g *Game) spawnTabRect(index int) rl.Rectangle {
	rec := g.spawnMenuPanel()
	return rl.Rectangle{
		X:      g.spawnContentX() + spawnPad + float32(index)*100,
		Y:      rec.Y + 15,
		Width:  92,
		Height: 30,
	}
}

func (g *Game) spawnSearchRect() rl.Rectangle {
	rec := g.spawnMenuPanel()
	return rl.Rectangle{X: rec.X + rec.Width - 38 - 8 - 224, Y: rec.Y + 15, Width: 224, Height: 30}
}

func (g *Game) spawnCloseRect() rl.Rectangle {
	rec := g.spawnMenuPanel()
	return rl.Rectangle{X: rec.X + rec.Width - 34, Y: rec.Y + 15, Width: 24, Height: 30}
}

func (g *Game) spawnFooterRect() rl.Rectangle {
	rec := g.spawnMenuPanel()
	return rl.Rectangle{X: g.spawnContentX(), Y: rec.Y + rec.Height - spawnFooterH, Width: rec.Width - spawnRailW, Height: spawnFooterH}
}

func (g *Game) spawnRailRow(index int) rl.Rectangle {
	rail := g.spawnRailRect()
	return rl.Rectangle{
		X:      rail.X + 12,
		Y:      rail.Y + 90 + float32(index)*32,
		Width:  rail.Width - 24,
		Height: 29,
	}
}

func (g *Game) spawnDetailButton() rl.Rectangle {
	d := g.spawnDetailRect()
	return rl.Rectangle{X: d.X + 16, Y: d.Y + d.Height - 54, Width: d.Width - 32, Height: 38}
}

// ---------------------------------------------------------------------------
// Spawn menu: filtering and selection
// ---------------------------------------------------------------------------

func (g *Game) filteredPresetIndices() []int {
	q := strings.ToLower(strings.TrimSpace(g.spawnSearch))
	out := make([]int, 0, len(presets))
	for i, p := range presets {
		if g.spawnCat != "" && g.spawnCat != "ALL" && presetCategoryName(p) != g.spawnCat {
			continue
		}
		if q != "" {
			hay := strings.ToLower(p.Name + " " + presetCategoryName(p) + " " + p.Kind.String())
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}

func (g *Game) filteredToolIndices() []int {
	q := strings.ToLower(strings.TrimSpace(g.spawnSearch))
	out := make([]int, 0, len(toolDefs))
	for i, t := range toolDefs {
		if q != "" {
			hay := strings.ToLower(t.Name + " " + t.Tag + " " + t.Desc)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}

func (g *Game) spawnEntryCount() int {
	switch g.spawnTab {
	case spawnTabProps:
		return len(g.filteredPresetIndices())
	case spawnTabTools:
		return len(g.filteredToolIndices())
	default:
		return len(worldActions)
	}
}

func (g *Game) ensureSpawnSelection() {
	switch g.spawnTab {
	case spawnTabProps:
		list := g.filteredPresetIndices()
		if len(list) == 0 {
			return
		}
		for _, i := range list {
			if i == g.selected {
				return
			}
		}
		g.selected = list[0]
	case spawnTabTools:
		list := g.filteredToolIndices()
		if len(list) == 0 {
			return
		}
		for _, i := range list {
			if ToolID(i) == g.menuToolSel {
				return
			}
		}
		g.menuToolSel = ToolID(list[0])
	}
}

type menuCard struct {
	Index int
	Rec   rl.Rectangle
}

func (g *Game) spawnCards() []menuCard {
	grid := g.spawnGridRect()
	n := g.spawnEntryCount()
	if n == 0 {
		return nil
	}
	const gap = 8.0
	cardW := float32(128)
	cols := int((grid.Width + gap) / (cardW + gap))
	if cols < 2 {
		cols = 2
	}
	if cols > n {
		cols = n
	}
	rows := (n + cols - 1) / cols
	cardH := (grid.Height - float32(rows-1)*gap) / float32(rows)
	cardH = clampf(cardH, 68, 112)
	totalW := float32(cols)*cardW + float32(cols-1)*gap
	x0 := grid.X + (grid.Width-totalW)*0.5
	y0 := grid.Y

	// Indices vary per tab.
	var order []int
	switch g.spawnTab {
	case spawnTabProps:
		order = g.filteredPresetIndices()
	case spawnTabTools:
		order = g.filteredToolIndices()
	default:
		order = make([]int, len(worldActions))
		for i := range worldActions {
			order[i] = i
		}
	}

	cards := make([]menuCard, 0, len(order))
	for slot, idx := range order {
		r := slot / cols
		c := slot % cols
		cards = append(cards, menuCard{
			Index: idx,
			Rec: rl.Rectangle{
				X:      x0 + float32(c)*(cardW+gap),
				Y:      y0 + float32(r)*(cardH+gap),
				Width:  cardW,
				Height: cardH,
			},
		})
	}
	return cards
}

func (g *Game) spawnRowCount() int {
	switch g.spawnTab {
	case spawnTabProps:
		return len(spawnCategories)
	case spawnTabTools:
		return len(toolDefs)
	default:
		return len(worldActions)
	}
}

// ---------------------------------------------------------------------------
// Spawn menu: actions
// ---------------------------------------------------------------------------

func (g *Game) openSpawnMenu() {
	if g.state != statePlaying {
		return
	}
	g.spawnAnchor = g.defaultSpawnPoint()
	g.spawnAnchorOK = true
	g.spawnTab = spawnTabProps
	g.spawnCat = "ALL"
	g.spawnSearch = ""
	g.searchActive = false
	g.menuToolSel = g.tool
	g.menuWorldSel = 0
	g.menuSpawnN = 0
	g.state = stateSpawn
	g.menuIgnoreClick = false
	fmt.Println("SPAWN CATS:", spawnCategories)
	fmt.Println("SPAWN ENTRIES:", g.spawnEntryCount())
}

func (g *Game) closeSpawnMenu() {
	g.searchActive = false
	g.state = statePlaying
	g.justResumed = true
	g.ignoreMouse = true
}

func (g *Game) spawnFromMenu(index int) {
	if g.state != stateSpawn {
		return
	}
	index = clampInt(index, 0, len(presets)-1)
	g.selected = index
	p := presets[index]
	if len(g.props) >= maxProps {
		g.toast("PROP LIMIT REACHED", colWarn)
		return
	}

	pos := g.spawnAnchor
	support := p.support(bv(0, 1, 0))
	if pos.Y < support+0.06 {
		pos.Y = support + 0.06
	}
	// Repeated spawns are laid out in a small grid around the captured drop
	// point instead of stacking perfectly on top of each other. The simulation
	// is frozen while the browser is open, so overlapping bodies would explode
	// apart the moment play resumes.
	spacing := 2*p.support(bv(1, 0, 0)) + 0.25
	col := g.menuSpawnN % 3
	row := g.menuSpawnN / 3
	g.menuSpawnN++
	right := g.rightFlat()
	fwd := g.forwardFlat()
	off := float32(col-1) * spacing
	pos.X += right.X*off - fwd.X*float32(row)*spacing
	pos.Z += right.Z*off - fwd.Z*float32(row)*spacing
	pos.X = clampf(pos.X, -29, 29)
	pos.Z = clampf(pos.Z, -29, 29)

	e := g.spawnPreset(p, toBp(pos), box3d.QuatIdentity, p.Color)
	if e == nil {
		g.toast("SPAWN FAILED", colDanger)
		return
	}
	g.lastSpawn = e
	g.burst(pos, e.Color, 10, 3.5, 0.09)
	g.wakeAround(pos, 1.4)
	// Keep the browser open so several props can be spawned in one pass, the
	// same way a GMod-style spawn menu behaves. The simulation stays frozen
	// until the menu closes, and the capacity bar tracks the count silently.
}

func (g *Game) activateSpawnSelection() {
	switch g.spawnTab {
	case spawnTabProps:
		g.spawnFromMenu(g.selected)
	case spawnTabTools:
		g.equipTool(g.menuToolSel)
		g.closeSpawnMenu()
	case spawnTabWorld:
		g.runWorldAction(g.menuWorldSel)
	}
}

func (g *Game) runWorldAction(index int) {
	switch index {
	case 0:
		g.saveGame()
	case 1:
		g.closeSpawnMenu()
		g.loadGame()
	case 2:
		yaw, pitch := g.player.yaw, g.player.pitch
		fly := g.player.fly
		g.closeSpawnMenu()
		g.resetWorld(true, defaultSpawn, yaw, pitch, fly)
		g.toast("ARENA RESET", colAccent)
	case 3:
		g.removeAllProps()
		g.toast("ALL PROPS CLEARED", colWarn)
	case 4:
		g.openSettings()
	}
}

// ---------------------------------------------------------------------------
// Spawn menu: input
// ---------------------------------------------------------------------------

func (g *Game) updateSpawnMenuInput() {
	if g.menuIgnoreClick {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			g.menuIgnoreClick = false
		}
		return
	}

	if g.searchActive && g.spawnTab != spawnTabWorld {
		for {
			ch := rl.GetCharPressed()
			if ch == 0 {
				break
			}
			if ch >= 32 && ch < 127 && len(g.spawnSearch) < 20 {
				g.spawnSearch += string(rune(ch))
			}
		}
		if rl.IsKeyPressed(rl.KeyBackspace) && len(g.spawnSearch) > 0 {
			g.spawnSearch = g.spawnSearch[:len(g.spawnSearch)-1]
		}
		g.ensureSpawnSelection()
	}

	// Enter activates whatever the details panel is showing.
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		g.activateSpawnSelection()
		return
	}

	if wheel := rl.GetMouseWheelMove(); wheel != 0 && g.spawnTab == spawnTabProps {
		list := g.filteredPresetIndices()
		if len(list) > 0 {
			pos := 0
			for i, idx := range list {
				if idx == g.selected {
					pos = i
					break
				}
			}
			if wheel > 0 {
				pos = (pos - 1 + len(list)) % len(list)
			} else {
				pos = (pos + 1) % len(list)
			}
			g.selected = list[pos]
		}
	}

	mouse := g.uiMouse()
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}

	// Search field focus is explicit so Q keeps closing the menu.
	search := g.spawnSearchRect()
	if g.spawnTab != spawnTabWorld && rl.CheckCollisionPointRec(mouse, search) {
		g.searchActive = true
		return
	}
	g.searchActive = false

	for i := range spawnTabs {
		if rl.CheckCollisionPointRec(mouse, g.spawnTabRect(i)) {
			g.spawnTab = i
			g.spawnSearch = ""
			g.ensureSpawnSelection()
			return
		}
	}
	if rl.CheckCollisionPointRec(mouse, g.spawnCloseRect()) {
		g.closeSpawnMenu()
		return
	}

	for i := 0; i < g.spawnRowCount(); i++ {
		if !rl.CheckCollisionPointRec(mouse, g.spawnRailRow(i)) {
			continue
		}
		switch g.spawnTab {
		case spawnTabProps:
			g.spawnCat = spawnCategories[i]
			g.ensureSpawnSelection()
		case spawnTabTools:
			g.menuToolSel = ToolID(i)
		case spawnTabWorld:
			g.menuWorldSel = i
		}
		return
	}

	for _, card := range g.spawnCards() {
		if !rl.CheckCollisionPointRec(mouse, card.Rec) {
			continue
		}
		switch g.spawnTab {
		case spawnTabProps:
			g.selected = card.Index
		case spawnTabTools:
			g.menuToolSel = ToolID(card.Index)
		case spawnTabWorld:
			g.menuWorldSel = card.Index
		}
		// Double click activates, exactly like a desktop browser.
		if g.lastCardKind == g.spawnTab && g.lastCardIndex == card.Index && g.uiTime-g.lastCardClick < 0.34 {
			g.lastCardClick = 0
			g.activateSpawnSelection()
			return
		}
		g.lastCardClick = g.uiTime
		g.lastCardIndex = card.Index
		g.lastCardKind = g.spawnTab
		return
	}

	if rl.CheckCollisionPointRec(mouse, g.spawnDetailButton()) {
		g.activateSpawnSelection()
	}
}

// ---------------------------------------------------------------------------
// Spawn menu: drawing
// ---------------------------------------------------------------------------

func (g *Game) drawSpawnMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 215))

	g.ensurePreviewThumbCache()
	rec := g.spawnMenuPanel()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.drawSpawnRail()
	g.drawSpawnHeader()
	g.drawSpawnGrid()
	g.drawSpawnDetail()
	g.drawSpawnFooter()
}

func (g *Game) drawSpawnRail() {
	rail := g.spawnRailRect()
	rl.DrawRectangleRec(rail, rgba(17, 18, 20, 250))
	rl.DrawLineEx(rl2(rail.X+rail.Width, rail.Y), rl2(rail.X+rail.Width, rail.Y+rail.Height), 1, colBorder)
	g.accentBar(rl.Rectangle{X: rail.X, Y: rail.Y, Width: 3, Height: rail.Height}, colAccent)

	g.drawTextSpaced("FSBOX", rail.X+16, rail.Y+16, 17, 2.6, colText)
	g.drawTextSpaced("SPAWN INTERFACE", rail.X+17, rail.Y+40, 9.5, 1.5, colMuted)
	g.drawTextSpaced("Q / ESC  CLOSE", rail.X+17, rail.Y+59, 9, 1.1, withAlpha(colMuted, 170))
	rl.DrawLineEx(rl2(rail.X+14, rail.Y+76), rl2(rail.X+rail.Width-14, rail.Y+76), 1, colBorder)

	mouse := g.uiMouse()
	for i := 0; i < g.spawnRowCount(); i++ {
		row := g.spawnRailRow(i)
		hover := rl.CheckCollisionPointRec(mouse, row)
		selected := false
		var label string
		var sub string
		var color = colMuted

		switch g.spawnTab {
		case spawnTabProps:
			selected = g.spawnCat == spawnCategories[i]
			label = spawnCategories[i]
			count := 0
			for _, p := range presets {
				if spawnCategories[i] == "ALL" || presetCategoryName(p) == spawnCategories[i] {
					count++
				}
			}
			sub = fmt.Sprintf("%02d", count)
		case spawnTabTools:
			selected = ToolID(i) == g.menuToolSel
			label = toolDefs[i].Name
			sub = itoa(i + 1)
			color = toolDefs[i].Color
		case spawnTabWorld:
			selected = i == g.menuWorldSel
			label = worldActions[i].Name
			sub = worldActions[i].Sub
		}

		fill := rgba(0, 0, 0, 0)
		border := rgba(0, 0, 0, 0)
		if hover {
			fill = colPanelHover
			border = colBorder
		}
		if selected {
			fill = colRowSelected
			border = colAccent
		}
		rl.DrawRectangleRec(row, fill)
		rl.DrawRectangleLinesEx(row, 1, border)
		if selected {
			g.accentBar(rl.Rectangle{X: row.X, Y: row.Y, Width: 2, Height: row.Height}, colAccent)
		}
		g.drawTextSpaced(label, row.X+10, row.Y+8, 11, 1.2, withAlpha(colText, 235))
		if color != colMuted {
			rl.DrawRectangle(int32(row.X+row.Width-30), int32(row.Y+13), 4, 4, color)
		}
		sw := g.textWidthSpaced(sub, 9.5, 0.8)
		g.drawTextSpaced(sub, row.X+row.Width-9-sw, row.Y+9, 9.5, 0.8, withAlpha(colMuted, 190))
	}

	// Rail footer: capacity readout.
	fy := rail.Y + rail.Height - 44
	g.drawTextSpaced("CAPACITY", rail.X+16, fy, 9.5, 1.2, colMuted)
	frac := float32(len(g.props)) / float32(maxProps)
	bar := rl.Rectangle{X: rail.X + 16, Y: fy + 16, Width: rail.Width - 32, Height: 4}
	rl.DrawRectangleRec(bar, rgba(255, 255, 255, 14))
	rl.DrawRectangleRec(rl.Rectangle{X: bar.X, Y: bar.Y, Width: bar.Width * clampf(frac, 0, 1), Height: bar.Height}, colAccent)
	cap := fmt.Sprintf("%d / %d", len(g.props), maxProps)
	cw := g.textWidthSpaced(cap, 9.5, 1.0)
	g.drawTextSpaced(cap, rail.X+rail.Width-16-cw, fy+26, 9.5, 1.0, withAlpha(colMuted, 190))
}

func (g *Game) drawSpawnHeader() {
	rec := g.spawnMenuPanel()
	header := rl.Rectangle{X: g.spawnContentX(), Y: rec.Y, Width: rec.Width - spawnRailW, Height: spawnHeaderH}
	rl.DrawRectangleRec(header, rgba(13, 14, 16, 252))
	rl.DrawLineEx(rl2(header.X, header.Y+header.Height), rl2(header.X+header.Width, header.Y+header.Height), 1, colBorder)

	mouse := g.uiMouse()
	for i, name := range spawnTabs {
		tab := g.spawnTabRect(i)
		selected := g.spawnTab == i
		hover := rl.CheckCollisionPointRec(mouse, tab)
		if selected {
			rl.DrawRectangleRec(tab, colRowSelected)
			rl.DrawRectangleLinesEx(tab, 1, colAccent)
			g.accentBar(rl.Rectangle{X: tab.X, Y: tab.Y + tab.Height - 2, Width: tab.Width, Height: 2}, colAccent)
		} else if hover {
			rl.DrawRectangleRec(tab, colPanelHover)
		}
		tc := colMuted
		if selected {
			tc = colText
		}
		g.textCenteredSpaced(name, tab.X+tab.Width*0.5, tab.Y+9, 11, 1.8, tc)
	}

	// Search field.
	search := g.spawnSearchRect()
	rl.DrawRectangleRec(search, rgba(0, 0, 0, 120))
	border := colBorder
	if g.searchActive {
		border = colAccent
	}
	rl.DrawRectangleLinesEx(search, 1, border)
	label := g.spawnSearch
	if label == "" {
		label = "SEARCH UNITS"
	}
	lcol := withAlpha(colText, 220)
	if g.spawnSearch == "" {
		lcol = withAlpha(colMuted, 165)
	}
	g.drawTextSpaced("S", search.X+9, search.Y+9, 10.5, 1.2, withAlpha(colAccent, 190))
	g.drawTextSpaced(label, search.X+27, search.Y+9, 10.5, 1.0, lcol)
	if g.searchActive && int(g.uiTime*2)%2 == 0 {
		cx := search.X + 27 + g.textWidthSpaced(g.spawnSearch, 10.5, 1.0) + 2
		rl.DrawRectangle(int32(cx), int32(search.Y+8), 1, 14, colAccent)
	}

	// Close control.
	close := g.spawnCloseRect()
	hover := rl.CheckCollisionPointRec(mouse, close)
	c := withAlpha(colMuted, 200)
	if hover {
		c = colDanger
	}
	g.drawTextSpaced("X", close.X+8, close.Y+8, 12, 1.0, c)
}

func (g *Game) drawSpawnGrid() {
	grid := g.spawnGridRect()
	mouse := g.uiMouse()
	cards := g.spawnCards()

	if len(cards) == 0 {
		g.textCenteredSpaced("NO ENTRIES MATCH THE FILTER", grid.X+grid.Width*0.5, grid.Y+grid.Height*0.5-6, 9, 1.6, withAlpha(colMuted, 200))
		return
	}

	for _, card := range cards {
		hover := rl.CheckCollisionPointRec(mouse, card.Rec)
		selected := false
		var accent = colAccent
		switch g.spawnTab {
		case spawnTabProps:
			selected = card.Index == g.selected
			accent = presets[card.Index].Color
		case spawnTabTools:
			selected = ToolID(card.Index) == g.menuToolSel
			accent = toolDefs[card.Index].Color
		case spawnTabWorld:
			selected = card.Index == g.menuWorldSel
			accent = colEnergy
		}

		fill := colPanelSoft
		border := colBorder
		if hover {
			fill = colPanelHover
			border = colBorderHi
		}
		if selected {
			fill = colRowSelected
			border = accent
		}
		rl.DrawRectangleRec(card.Rec, fill)
		rl.DrawRectangleLinesEx(card.Rec, 1, border)
		if selected {
			g.accentBar(rl.Rectangle{X: card.Rec.X, Y: card.Rec.Y, Width: 2, Height: card.Rec.Height}, accent)
		}

		switch g.spawnTab {
		case spawnTabProps:
			p := presets[card.Index]
			g.drawTextSpaced(presetCategoryName(p), card.Rec.X+8, card.Rec.Y+7, 8.5, 0.8, withAlpha(colMuted, 175))
			thumb := rl.Rectangle{
				X:      card.Rec.X + 8,
				Y:      card.Rec.Y + 18,
				Width:  card.Rec.Width - 16,
				Height: card.Rec.Height - 38,
			}
			if thumb.Height < 12 {
				thumb.Height = 12
			}
			g.drawPresetThumbnail(card.Index, thumb)
			g.textCenteredSpaced(strings.ToUpper(p.Name), card.Rec.X+card.Rec.Width*0.5, card.Rec.Y+card.Rec.Height-20, 10, 1.0, withAlpha(colText, 225))
		case spawnTabTools:
			d := toolDefs[card.Index]
			g.drawTextSpaced(d.Tag, card.Rec.X+8, card.Rec.Y+7, 8.5, 0.8, withAlpha(d.Color, 190))
			g.drawToolIcon(ToolID(card.Index), card.Rec.X+card.Rec.Width*0.5, card.Rec.Y+card.Rec.Height*0.44, minf(card.Rec.Height*0.32, 30), d.Color)
			g.textCenteredSpaced(d.Name, card.Rec.X+card.Rec.Width*0.5, card.Rec.Y+card.Rec.Height-20, 10, 1.0, withAlpha(colText, 225))
		case spawnTabWorld:
			wa := worldActions[card.Index]
			g.drawWorldIcon(card.Index, card.Rec.X+card.Rec.Width*0.5, card.Rec.Y+card.Rec.Height*0.44, minf(card.Rec.Height*0.34, 32), colEnergy)
			g.textCenteredSpaced(wa.Name, card.Rec.X+card.Rec.Width*0.5, card.Rec.Y+card.Rec.Height-20, 9.5, 0.9, withAlpha(colText, 225))
		}
	}
}

func (g *Game) drawSpawnDetail() {
	d := g.spawnDetailRect()
	g.panel(d, colPanelSoft, colBorder)
	g.accentBar(rl.Rectangle{X: d.X, Y: d.Y, Width: d.Width, Height: 2}, colAccent)
	g.drawSectionLabel("INSPECTION", d.X+14, d.Y+13, colMuted)

	// Preview well.
	well := rl.Rectangle{X: d.X + 14, Y: d.Y + 36, Width: d.Width - 28, Height: 124}
	rl.DrawRectangleRec(well, rgba(0, 0, 0, 130))
	for i := 0; i < 4; i++ {
		x := well.X + float32(i+1)*well.Width/5
		rl.DrawLineEx(rl2(x, well.Y+well.Height-8), rl2(x, well.Y+well.Height), 1, withAlpha(colMuted, 38))
	}
	if g.spawnTab == spawnTabProps {
		g.drawPresetPreview3D(presets[g.selected], well)
	}
	rl.DrawRectangleLinesEx(well, 1, colBorder)
	g.drawCornerMarks(well, withAlpha(colAccent, 130))

	nameY := well.Y + well.Height + 12
	var title, subtitle string
	var accent = colAccent

	switch g.spawnTab {
	case spawnTabProps:
		p := presets[g.selected]
		accent = p.Color
		title = strings.ToUpper(p.Name)
		subtitle = presetCategoryName(p) + " / " + strings.ToUpper(p.Kind.String())

		g.drawTextSpaced(title, d.X+14, nameY, 20, 2.4, colText)
		g.drawTextSpaced(subtitle, d.X+15, nameY+27, 9.5, 1.4, withAlpha(accent, 220))

		vol := presetVolume(p)
		mass := vol * p.Density
		rows := [][2]string{
			{"MASS", fmt.Sprintf("%.1f KG", mass)},
			{"FRICTION", fmt.Sprintf("%.2f", p.Friction)},
			{"BOUNCE", fmt.Sprintf("%.2f", p.Restitution)},
			{"SIZE", presetSizeLabel(p)},
		}
		for i, row := range rows {
			y := nameY + 58 + float32(i)*22
			g.drawTextSpaced(row[0], d.X+14, y, 9.5, 1.0, colMuted)
			vw := g.textWidthSpaced(row[1], 10.5, 1.0)
			g.drawTextSpaced(row[1], d.X+d.Width-14-vw, y, 10.5, 1.0, withAlpha(colText, 225))
		}
	case spawnTabTools:
		t := toolDefs[g.menuToolSel]
		accent = t.Color
		g.drawToolIcon(g.menuToolSel, well.X+well.Width*0.5, well.Y+well.Height*0.5, 46, t.Color)
		title = t.Name
		subtitle = t.Tag + " / TOOL MODE"
		g.drawTextSpaced(title, d.X+14, nameY, 20, 2.4, colText)
		g.drawTextSpaced(subtitle, d.X+15, nameY+27, 9.5, 1.4, withAlpha(accent, 220))
		g.drawWrappedText(t.Desc, d.X+14, nameY+58, d.Width-28, 11, 1.0, 17, withAlpha(colMuted, 225))
		g.drawTextSpaced("LMB", d.X+14, nameY+140, 9.5, 1.0, accent)
		g.drawTextSpaced(t.Primary, d.X+48, nameY+139, 10.5, 0.8, withAlpha(colText, 225))
		g.drawTextSpaced("RMB", d.X+14, nameY+162, 9.5, 1.0, colMuted)
		g.drawTextSpaced(t.Secondary, d.X+48, nameY+161, 10.5, 0.8, withAlpha(colMuted, 225))
	case spawnTabWorld:
		accent = colEnergy
		g.drawWorldIcon(g.menuWorldSel, well.X+well.Width*0.5, well.Y+well.Height*0.5, 48, colEnergy)
		wa := worldActions[g.menuWorldSel]
		title = wa.Name
		subtitle = "WORLD CONTROL / " + wa.Sub
		g.drawTextSpaced(title, d.X+14, nameY, 18, 2.0, colText)
		g.drawTextSpaced(subtitle, d.X+15, nameY+26, 9.5, 1.4, withAlpha(accent, 220))
		g.drawWrappedText(wa.Desc, d.X+14, nameY+58, d.Width-28, 11, 1.0, 17, withAlpha(colMuted, 225))
	}

	btn := g.spawnDetailButton()
	label := "SPAWN SELECTED"
	switch g.spawnTab {
	case spawnTabTools:
		label = "EQUIP TOOL"
	case spawnTabWorld:
		label = "EXECUTE"
	}
	g.drawMenuButton(btn, label, false)

	// Context line beneath the preview well.
	anchor := fmt.Sprintf("DROP ZONE  %+.1f  %+.1f  %+.1f", g.spawnAnchor.X, g.spawnAnchor.Y, g.spawnAnchor.Z)
	g.drawTextSpaced(anchor, d.X+14, d.Y+d.Height-76, 9, 0.8, withAlpha(colMuted, 160))
}

const (
	modelPreviewW = 520
	modelPreviewH = 250
	thumbPreviewW = 256
	thumbPreviewH = 192
)

// previewSignature identifies the current preset list so cached thumbnails are
// thrown away if a mod adds or removes spawnables.
func previewSignature() string {
	var b strings.Builder
	for i := range presets {
		p := presets[i]
		b.WriteString(p.Name)
		b.WriteByte('|')
		b.WriteString(p.Category)
		b.WriteByte('|')
		if p.HasModel {
			b.WriteByte('m')
		}
		b.WriteByte(';')
	}
	return b.String()
}

func (g *Game) ensurePreviewThumbCache() {
	sig := previewSignature()
	if sig == g.previewThumbSig && g.previewThumbs != nil {
		return
	}
	for i, rt := range g.previewThumbs {
		if rt.ID != 0 {
			rl.UnloadRenderTexture(rt)
		}
		delete(g.previewThumbs, i)
	}
	g.previewThumbs = make(map[int]rl.RenderTexture2D)
	g.previewThumbSig = sig
}

// presetThumbnail returns the cached 3D thumbnail for a spawnable, rendering it
// once on first use. Thumbnails are static, so caching keeps the spawn grid
// cheap even with many content packs.
func (g *Game) presetThumbnail(index int) rl.RenderTexture2D {
	if index < 0 || index >= len(presets) {
		return rl.RenderTexture2D{}
	}
	if g.previewThumbs == nil {
		g.previewThumbs = make(map[int]rl.RenderTexture2D)
	}
	if rt, ok := g.previewThumbs[index]; ok && rt.ID != 0 {
		return rt
	}
	rt := rl.LoadRenderTexture(thumbPreviewW, thumbPreviewH)
	if rt.ID == 0 {
		return rt
	}
	rl.SetTextureFilter(rt.Texture, rl.FilterBilinear)
	g.renderPresetPreview(presets[index], rt)
	g.previewThumbs[index] = rt
	return rt
}

func (g *Game) drawPresetThumbnail(index int, dest rl.Rectangle) {
	rt := g.presetThumbnail(index)
	if rt.ID == 0 {
		p := presets[index]
		g.drawPresetPreview(p, dest.X+dest.Width*0.5, dest.Y+dest.Height*0.5, minf(dest.Width, dest.Height), 255)
		return
	}
	g.drawRenderTextureFit(rt, dest)
}

func (g *Game) drawRenderTextureFit(rt rl.RenderTexture2D, dest rl.Rectangle) {
	tw := float32(rt.Texture.Width)
	th := float32(rt.Texture.Height)
	if tw <= 0 || th <= 0 {
		return
	}
	scale := minf(dest.Width/tw, dest.Height/th)
	w := tw * scale
	h := th * scale
	r := rl.Rectangle{
		X:      dest.X + (dest.Width-w)*0.5,
		Y:      dest.Y + (dest.Height-h)*0.5,
		Width:  w,
		Height: h,
	}
	// Render textures are stored bottom-up; flip the source rectangle.
	src := rl.Rectangle{X: 0, Y: 0, Width: tw, Height: -th}
	rl.DrawTexturePro(rt.Texture, src, r, rl2(0, 0), 0, rl.White)
}

// drawPresetPreview3D renders the selected preset into the shared inspection
// texture and blits it into the well.
func (g *Game) drawPresetPreview3D(p ShapePreset, well rl.Rectangle) {
	if g.modelPreview.ID == 0 {
		g.modelPreview = rl.LoadRenderTexture(modelPreviewW, modelPreviewH)
		if g.modelPreview.ID == 0 {
			return
		}
		rl.SetTextureFilter(g.modelPreview.Texture, rl.FilterBilinear)
	}
	g.renderPresetPreview(p, g.modelPreview)
	g.drawRenderTextureFit(g.modelPreview, well)
}

// renderPresetPreview draws a preset into an offscreen texture with the same
// framing used by the inspection well. Models and primitive presets share the
// path so every spawnable gets a real 3D thumbnail.
func (g *Game) renderPresetPreview(p ShapePreset, rt rl.RenderTexture2D) {
	// Leave 2D mode so the projection push/pop stays balanced; BeginTextureMode
	// overwrites the projection and EndTextureMode does not restore it.
	rl.EndMode2D()
	rl.BeginTextureMode(rt)
	rl.ClearBackground(rgba(8, 9, 10, 255))

	// Frame the model from its bounding box so packs authored at wildly
	// different scales still fill the well sensibly.
	r := presetFrameRadius(p)
	dir := rlv(0.55, 0.24, 1.0)
	dl := float32(math.Sqrt(float64(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)))
	dist := r * 3.1
	cam := rl.Camera3D{
		Position:   rlv(dir.X/dl*dist, dir.Y/dl*dist, dir.Z/dl*dist),
		Target:     rlv(0, 0, 0),
		Up:         rlv(0, 1, 0),
		Fovy:       40,
		Projection: rl.CameraPerspective,
	}
	rl.BeginMode3D(cam)
	rl.DisableBackfaceCulling()
	if p.HasModel {
		rl.DrawModelEx(p.Model, rlv(0, 0, 0), rlv(0, 1, 0), 0, rlv(1, 1, 1), rl.White)
	} else {
		drawPresetMesh3D(p)
	}
	rl.EnableBackfaceCulling()
	rl.EndMode3D()
	rl.EndTextureMode()

	// BeginTextureMode replaces the 2D projection, so restore the UI camera.
	rl.BeginMode2D(rl.Camera2D{Zoom: g.uiScale()})
}

func presetFrameRadius(p ShapePreset) float32 {
	var r float32
	switch p.Kind {
	case PropSphere:
		r = p.Radius
	case PropCylinder:
		r = float32(math.Sqrt(float64(p.Radius*p.Radius + p.Height*p.Height*0.25)))
	default:
		r = float32(math.Sqrt(float64(p.Half.X*p.Half.X + p.Half.Y*p.Half.Y + p.Half.Z*p.Half.Z)))
	}
	if r < 0.2 {
		r = 0.2
	}
	return r
}

func drawPresetMesh3D(p ShapePreset) {
	edge := withAlpha(rgb(8, 9, 10), 200)
	hi := withAlpha(rgb(255, 255, 255), 45)
	switch p.Kind {
	case PropSphere:
		rl.DrawSphere(rlv(0, 0, 0), p.Radius, p.Color)
		rl.DrawSphereWires(rlv(0, 0, 0), p.Radius*1.005, 8, 16, edge)
		rl.DrawSphere(rlv(-p.Radius*0.35, p.Radius*0.35, -p.Radius*0.25), p.Radius*0.22, hi)
	case PropCylinder:
		h := p.Height
		base := rlv(0, -h*0.5, 0)
		rl.DrawCylinder(base, p.Radius, p.Radius, h, 24, p.Color)
		rl.DrawCylinderWires(base, p.Radius, p.Radius, h, 24, edge)
		rl.DrawSphere(rlv(-p.Radius*0.3, h*0.2, -p.Radius*0.25), maxf(0.02, p.Radius*0.18), hi)
	default:
		size := rlv(p.Half.X*2, p.Half.Y*2, p.Half.Z*2)
		rl.DrawCubeV(rlv(0, 0, 0), size, p.Color)
		rl.DrawCubeWiresV(rlv(0, 0, 0), size, edge)
	}
}

func (g *Game) drawSpawnFooter() {
	f := g.spawnFooterRect()
	rl.DrawRectangleRec(f, rgba(13, 14, 16, 252))
	rl.DrawLineEx(rl2(f.X, f.Y), rl2(f.X+f.Width, f.Y), 1, colBorder)

	hints := "ENTER / DOUBLE CLICK  ACTIVATE     WHEEL  BROWSE     Q / ESC  CLOSE"
	g.drawTextSpaced(hints, f.X+16, f.Y+14, 9.5, 1.0, withAlpha(colMuted, 205))

	status := fmt.Sprintf("%02d ENTRIES", g.spawnEntryCount())
	sw := g.textWidthSpaced(status, 9.5, 1.2)
	g.drawTextSpaced(status, f.X+f.Width-16-sw, f.Y+14, 9.5, 1.2, withAlpha(colAccent, 190))
}

func (g *Game) drawCornerMarks(rec rl.Rectangle, c rl.Color) {
	const n = 7.0
	// Top-left
	rl.DrawLineEx(rl2(rec.X, rec.Y), rl2(rec.X+n, rec.Y), 1, c)
	rl.DrawLineEx(rl2(rec.X, rec.Y), rl2(rec.X, rec.Y+n), 1, c)
	// Top-right
	rl.DrawLineEx(rl2(rec.X+rec.Width-n, rec.Y), rl2(rec.X+rec.Width, rec.Y), 1, c)
	rl.DrawLineEx(rl2(rec.X+rec.Width, rec.Y), rl2(rec.X+rec.Width, rec.Y+n), 1, c)
	// Bottom-left
	rl.DrawLineEx(rl2(rec.X, rec.Y+rec.Height-n), rl2(rec.X, rec.Y+rec.Height), 1, c)
	rl.DrawLineEx(rl2(rec.X, rec.Y+rec.Height), rl2(rec.X+n, rec.Y+rec.Height), 1, c)
	// Bottom-right
	rl.DrawLineEx(rl2(rec.X+rec.Width-n, rec.Y+rec.Height), rl2(rec.X+rec.Width, rec.Y+rec.Height), 1, c)
	rl.DrawLineEx(rl2(rec.X+rec.Width, rec.Y+rec.Height-n), rl2(rec.X+rec.Width, rec.Y+rec.Height), 1, c)
}

func (g *Game) drawJaggedLine(x0, x1, y float32, c rl.Color) {
	rl.DrawLineEx(rl2(x0, y), rl2(x1, y), 1, c)
}

func (g *Game) drawWrappedText(text string, x, y, width, size, spacing, lineStep float32, c rl.Color) {
	words := strings.Fields(text)
	line := ""
	for _, w := range words {
		test := w
		if line != "" {
			test = line + " " + w
		}
		if g.textWidthSpaced(test, size, spacing) > width && line != "" {
			g.drawTextSpaced(line, x, y, size, spacing, c)
			y += lineStep
			line = w
			continue
		}
		line = test
	}
	if line != "" {
		g.drawTextSpaced(line, x, y, size, spacing, c)
	}
}

// ---------------------------------------------------------------------------
// Proportional 2D prop preview
// ---------------------------------------------------------------------------

func (g *Game) drawPresetPreview(p ShapePreset, cx, cy, maxSize float32, alpha uint8) {
	c := withAlpha(p.Color, alpha)
	edge := withAlpha(rgb(6, 7, 8), alpha)
	light := withAlpha(rgb(255, 255, 255), uint8(float32(alpha)*0.24))

	var w, h float32
	switch p.Kind {
	case PropBox:
		w, h = 2*maxf(p.Half.X, p.Half.Z), 2*p.Half.Y
	case PropSphere:
		w, h = p.Radius*2, p.Radius*2
	case PropCylinder:
		w, h = p.Radius*2, p.Height
	}
	if w < 0.05 {
		w = 1
	}
	if h < 0.05 {
		h = 1
	}
	scale := maxSize / maxf(w, h)
	w = clampf(w*scale, 10, maxSize)
	h = clampf(h*scale, 10, maxSize)

	switch p.Kind {
	case PropSphere:
		cv := rl.Vector2{X: cx, Y: cy}
		rl.DrawCircleV(cv, w*0.5, c)
		rl.DrawRing(cv, w*0.5-1, w*0.5, 0, 360, 32, edge)
		rl.DrawCircleV(rl2(cx-w*0.16, cy-h*0.18), maxf(2, w*0.11), light)
		rl.DrawLineEx(rl2(cx-w*0.3, cy+h*0.28), rl2(cx+w*0.3, cy-h*0.28), 1, withAlpha(edge, uint8(float32(alpha)*0.7)))
	case PropCylinder:
		rec := rl.Rectangle{X: cx - w*0.5, Y: cy - h*0.5, Width: w, Height: h}
		rl.DrawRectangleRec(rec, c)
		rl.DrawRectangleLinesEx(rec, 1, edge)
		rl.DrawLineEx(rl2(cx-w*0.5, cy-h*0.16), rl2(cx+w*0.5, cy-h*0.16), 1, edge)
		rl.DrawLineEx(rl2(cx-w*0.5, cy+h*0.16), rl2(cx+w*0.5, cy+h*0.16), 1, edge)
		rl.DrawCircleV(rl2(cx, cy-h*0.32), maxf(2, w*0.12), light)
	default:
		rec := rl.Rectangle{X: cx - w*0.5, Y: cy - h*0.5, Width: w, Height: h}
		rl.DrawRectangleRec(rec, c)
		rl.DrawRectangleLinesEx(rec, 1, edge)
		rl.DrawLineEx(rl2(cx-w*0.5, cy), rl2(cx+w*0.5, cy), 1, edge)
		rl.DrawLineEx(rl2(cx, cy-h*0.5), rl2(cx, cy+h*0.5), 1, edge)
		rl.DrawLineEx(rl2(cx-w*0.32, cy-h*0.34), rl2(cx+w*0.32, cy-h*0.34), 1, light)
	}
}

func presetVolume(p ShapePreset) float32 {
	switch p.Kind {
	case PropBox:
		return 8 * p.Half.X * p.Half.Y * p.Half.Z
	case PropSphere:
		return 4.0 / 3.0 * 3.14159 * p.Radius * p.Radius * p.Radius
	case PropCylinder:
		return 3.14159 * p.Radius * p.Radius * p.Height
	}
	return 1
}

func presetSizeLabel(p ShapePreset) string {
	switch p.Kind {
	case PropBox:
		return fmt.Sprintf("%.2f x %.2f", p.Half.X*2, p.Half.Y*2)
	case PropSphere:
		return fmt.Sprintf("D %.2f", p.Radius*2)
	case PropCylinder:
		return fmt.Sprintf("D %.2f H %.2f", p.Radius*2, p.Height)
	}
	return "-"
}

func (g *Game) drawWorldIcon(index int, cx, cy, size float32, c rl.Color) {
	s := size * 0.5
	switch index {
	case 0: // save
		g.stroke(rl.Rectangle{X: cx - s*0.7, Y: cy - s*0.8, Width: s * 1.4, Height: s * 1.6}, c)
		rl.DrawRectangleRec(rl.Rectangle{X: cx - s*0.35, Y: cy - s*0.8, Width: s * 0.7, Height: s * 0.5}, withAlpha(c, 90))
		rl.DrawRectangleRec(rl.Rectangle{X: cx - s*0.45, Y: cy + s*0.15, Width: s * 0.9, Height: s * 0.55}, withAlpha(c, 60))
	case 1: // load
		g.stroke(rl.Rectangle{X: cx - s*0.7, Y: cy - s*0.2, Width: s * 1.4, Height: s * 1.0}, c)
		rl.DrawLineEx(rl2(cx, cy-s*0.85), rl2(cx, cy+s*0.05), 2, c)
		rl.DrawLineEx(rl2(cx, cy+s*0.1), rl2(cx-s*0.22, cy-s*0.15), 2, c)
		rl.DrawLineEx(rl2(cx, cy+s*0.1), rl2(cx+s*0.22, cy-s*0.15), 2, c)
	case 2: // reset
		rl.DrawCircleLines(int32(cx), int32(cy), s*0.7, c)
		rl.DrawLineEx(rl2(cx+s*0.55, cy-s*0.55), rl2(cx+s*0.78, cy-s*0.22), 2, c)
		rl.DrawLineEx(rl2(cx+s*0.55, cy-s*0.55), rl2(cx+s*0.15, cy-s*0.45), 2, c)
	case 3: // clear
		g.stroke(rl.Rectangle{X: cx - s*0.7, Y: cy - s*0.6, Width: s * 1.4, Height: s * 1.3}, withAlpha(c, 120))
		rl.DrawLineEx(rl2(cx-s*0.55, cy-s*0.45), rl2(cx+s*0.55, cy+s*0.55), 2, c)
		rl.DrawLineEx(rl2(cx+s*0.55, cy-s*0.45), rl2(cx-s*0.55, cy+s*0.55), 2, c)
	case 4: // settings
		rl.DrawCircleLines(int32(cx), int32(cy), s*0.38, c)
		for i := 0; i < 6; i++ {
			a := float64(i) * 3.14159 / 3
			x0 := cx + float32(mathCos(a))*s*0.58
			y0 := cy + float32(mathSin(a))*s*0.58
			x1 := cx + float32(mathCos(a))*s*0.78
			y1 := cy + float32(mathSin(a))*s*0.78
			rl.DrawLineEx(rl2(x0, y0), rl2(x1, y1), 2, c)
		}
	}
}

// ---------------------------------------------------------------------------
// Settings panel
// ---------------------------------------------------------------------------

func (g *Game) settingsPanel() rl.Rectangle {
	w := g.uiWidth()
	h := g.uiHeight()
	pw := minf(740, w-40)
	ph := minf(640, h-40)
	return rl.Rectangle{X: (w - pw) * 0.5, Y: (h - ph) * 0.5, Width: pw, Height: ph}
}

func (g *Game) settingsSliderRect(index int) rl.Rectangle {
	rec := g.settingsPanel()
	return rl.Rectangle{
		X:      rec.X + 30,
		Y:      rec.Y + 156 + float32(index)*54,
		Width:  rec.Width - 60,
		Height: 3,
	}
}

func (g *Game) settingsToggleRect(index int) rl.Rectangle {
	rec := g.settingsPanel()
	y := rec.Y + 128 + float32(3+index)*54
	return rl.Rectangle{X: rec.X + rec.Width - 30 - 58, Y: y - 3, Width: 58, Height: 24}
}

func (g *Game) settingsDisplayBoxRect() rl.Rectangle {
	rec := g.settingsPanel()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 390, Width: rec.Width - 60, Height: 32}
}

func (g *Game) settingsDisplayModeRow(index int) rl.Rectangle {
	rec := g.settingsPanel()
	return rl.Rectangle{
		X:      rec.X + 30,
		Y:      rec.Y + 426 + float32(index)*28,
		Width:  rec.Width - 60,
		Height: 26,
	}
}
func (g *Game) settingsButtons() (reset, back rl.Rectangle) {
	rec := g.settingsPanel()
	const bw, bh, gap float32 = 150, 38, 10
	total := bw*3 + gap*2
	x0 := rec.X + (rec.Width-total)*0.5
	y := rec.Y + rec.Height - 52
	return rl.Rectangle{X: x0, Y: y, Width: bw, Height: bh},
		rl.Rectangle{X: x0 + 2*(bw+gap), Y: y, Width: bw, Height: bh}
}

func (g *Game) settingsApplyRect() rl.Rectangle {
	rec := g.settingsPanel()
	const bw, bh, gap float32 = 150, 38, 10
	total := bw*3 + gap*2
	x0 := rec.X + (rec.Width-total)*0.5
	y := rec.Y + rec.Height - 52
	return rl.Rectangle{X: x0 + bw + gap, Y: y, Width: bw, Height: bh}
}

func (g *Game) openSettings() {
	if g.state != statePlaying && g.state != statePaused && g.state != stateSpawn && g.state != stateToolMenu {
		return
	}
	g.settingsReturn = g.state
	g.state = stateSettings
	g.settingsDraft = g.settings
	g.settingsDisplayOpen = false
	g.draggingSlider = -1
	g.searchActive = false
	g.menuIgnoreClick = true
}

func (g *Game) closeSettings() {
	if g.settingsDirty {
		_ = g.saveSettings()
		g.settingsDirty = false
	}
	g.draggingSlider = -1
	if g.settingsReturn == statePlaying {
		g.state = statePlaying
		g.justResumed = true
		g.ignoreMouse = true
		return
	}
	g.state = g.settingsReturn
	g.menuIgnoreClick = true
}

func (g *Game) setSliderFromMouse(index int, mouseX float32) {
	tr := g.settingsSliderRect(index)
	t := clampf((mouseX-tr.X)/tr.Width, 0, 1)
	switch index {
	case 0:
		g.settingsDraft.Gravity = clampf(lerpf(0, 2.5, t), 0, 2.5)
	case 1:
		g.settingsDraft.Sensitivity = clampf(lerpf(0, 1, t), 0, 1)
	case 2:
		g.settingsDraft.FOV = clampf(lerpf(60, 105, t), 60, 105)
	case 3:
		g.settingsDraft.UIScale = clampf(lerpf(0.75, 1.5, t), 0.75, 1.5)
	}
}

func (g *Game) updateSettingsInput() {
	if g.menuIgnoreClick {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			g.menuIgnoreClick = false
		}
		return
	}

	mouse := g.uiMouse()
	pressed := rl.IsMouseButtonPressed(rl.MouseButtonLeft)
	down := rl.IsMouseButtonDown(rl.MouseButtonLeft)

	if pressed {
		for i := 0; i < 3; i++ {
			tr := g.settingsSliderRect(i)
			hit := rl.Rectangle{X: tr.X - 8, Y: tr.Y - 14, Width: tr.Width + 16, Height: tr.Height + 28}
			if rl.CheckCollisionPointRec(mouse, hit) {
				g.draggingSlider = i
				g.setSliderFromMouse(i, mouse.X)
			}
		}
		for i := 0; i < 3; i++ {
			if rl.CheckCollisionPointRec(mouse, g.settingsToggleRect(i)) {
				switch i {
				case 0:
					g.settingsDraft.InvertY = !g.settingsDraft.InvertY
				case 1:
					g.settingsDraft.Shadows = !g.settingsDraft.Shadows
				case 2:
					g.settingsDraft.RawInput = !g.settingsDraft.RawInput
				}
				g.settingsDirty = true
			}
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			if rl.CheckCollisionPointRec(mouse, g.settingsDisplayBoxRect()) {
				g.settingsDisplayOpen = !g.settingsDisplayOpen
			} else if g.settingsDisplayOpen {
				selected := false
				for i := 0; i < 2; i++ {
					if rl.CheckCollisionPointRec(mouse, g.settingsDisplayModeRow(i)) {
						g.settingsDraft.DisplayMode = i
						g.settingsDisplayOpen = false
						selected = true
						break
					}
				}
				if !selected {
					g.settingsDisplayOpen = false
				}
			}
		}
		if rl.CheckCollisionPointRec(mouse, g.settingsApplyRect()) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			g.settings = g.settingsDraft
			g.settings.clampValues()
			g.settingsDirty = true
			g.applyGravity()
			g.applyDisplayMode()
			_ = g.saveSettings()
		}

		reset, back := g.settingsButtons()
		if rl.CheckCollisionPointRec(mouse, reset) {
			g.settingsDraft = defaultSettings()
		}
		if rl.CheckCollisionPointRec(mouse, back) {
			g.closeSettings()
			return
		}
	}

	if g.draggingSlider >= 0 {
		if down {
			g.setSliderFromMouse(g.draggingSlider, mouse.X)
		} else {
			g.draggingSlider = -1
		}
	}
}

func (g *Game) drawSettings() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 210))

	rec := g.settingsPanel()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.drawTextSpaced("SETTINGS", rec.X+30, rec.Y+22, 24, 3.0, colText)
	g.drawTextSpaced("SYSTEM CONFIGURATION  /  CHANGES APPLY LIVE", rec.X+31, rec.Y+57, 9.5, 1.5, colMuted)
	closeText := "F2 / ESC  CLOSE"
	cw := g.textWidthSpaced(closeText, 9.5, 1.2)
	g.drawTextSpaced(closeText, rec.X+rec.Width-30-cw, rec.Y+32, 9.5, 1.2, colMuted)
	rl.DrawLineEx(rl2(rec.X+30, rec.Y+80), rl2(rec.X+rec.Width-30, rec.Y+80), 1, colBorder)

	labels := []string{"WORLD GRAVITY", "MOUSE SENSITIVITY", "FIELD OF VIEW", "UI SCALE"}
	values := []string{
		fmt.Sprintf("%.2f X", g.settingsDraft.Gravity),
		fmt.Sprintf("%.0f%%", g.settingsDraft.Sensitivity*100),
		fmt.Sprintf("%.0f DEG", g.settingsDraft.FOV),
		fmt.Sprintf("%.0f%%", g.settingsDraft.UIScale*100),
	}
	t := []float32{
		g.settingsDraft.Gravity / 2.5,
		g.settingsDraft.Sensitivity,
		(g.settingsDraft.FOV - 60) / (105 - 60),
		(g.settingsDraft.UIScale - 0.75) / (1.5 - 0.75),
	}
	for i := 0; i < 4; i++ {
		y := rec.Y + 128 + float32(i)*54
		tr := g.settingsSliderRect(i)
		g.drawTextSpaced(labels[i], rec.X+30, y, 10.5, 1.4, colText)
		vw := g.textWidthSpaced(values[i], 10.5, 1.2)
		g.drawTextSpaced(values[i], tr.X+tr.Width-vw, y-1, 10.5, 1.2, colAccent)

		rl.DrawRectangleRec(tr, rgba(255, 255, 255, 16))
		frac := clampf(t[i], 0, 1)
		if fill := tr.Width * frac; fill > 0 {
			rl.DrawRectangleRec(rl.Rectangle{X: tr.X, Y: tr.Y, Width: fill, Height: tr.Height}, colAccent)
		}
		knobX := tr.X + tr.Width*frac
		rl.DrawRectangleRec(rl.Rectangle{X: knobX - 3, Y: tr.Y - 6, Width: 6, Height: 15}, colAccentHi)
	}

	toggles := []struct {
		label string
		on    bool
	}{
		{"INVERT MOUSE Y", g.settings.InvertY},
		{"CONTACT SHADOWS", g.settings.Shadows},
		{"RAW MOUSE INPUT", g.settings.RawInput},
	}
	for i, tog := range toggles {
		y := rec.Y + 128 + float32(3+i)*54
		g.drawTextSpaced(tog.label, rec.X+30, y, 10.5, 1.4, colText)
		box := g.settingsToggleRect(i)
		rl.DrawRectangleRec(box, rgba(0, 0, 0, 110))
		rl.DrawRectangleLinesEx(box, 1, colBorderHi)
		if tog.on {
			rl.DrawRectangleRec(rl.Rectangle{X: box.X + 3, Y: box.Y + 3, Width: box.Width - 6, Height: box.Height - 6}, withAlpha(colAccent, 210))
			g.drawTextSpaced("ON", box.X+box.Width+8, y, 9.5, 1.1, colAccent)
		} else {
			g.drawTextSpaced("OFF", box.X+box.Width+8, y, 9.5, 1.1, colMuted)
		}
	}

	g.drawTextSpaced("DISPLAY MODE", rec.X+30, rec.Y+376, 10.5, 1.4, colText)
	box := g.settingsDisplayBoxRect()
	g.drawMenuButton(box, g.displayModeName(), true)
	if g.settingsDisplayOpen {
		displayModes := []string{"FULLSCREEN", "WINDOWED FULLSCREEN"}
		for i, name := range displayModes {
			row := g.settingsDisplayModeRow(i)
			selected := g.settingsDraft.DisplayMode == i
			fill := colPanelSoft
			border := colBorder
			if selected {
				fill = colRowSelected
				border = colAccent
			}
			rl.DrawRectangleRec(row, fill)
			rl.DrawRectangleLinesEx(row, 1, border)
			if selected {
				g.accentBar(rl.Rectangle{X: row.X, Y: row.Y, Width: 3, Height: row.Height}, colAccent)
			}
			g.drawTextSpaced(name, row.X+12, row.Y+6, 10.5, 1.2, withAlpha(colText, 225))
		}
	}

	g.drawMenuButton(g.settingsApplyRect(), "APPLY", false)

	reset, back := g.settingsButtons()
	g.drawMenuButton(reset, "RESET DEFAULTS", true)
	g.drawMenuButton(back, "BACK", false)
	g.textCenteredSpaced("SETTINGS ARE STORED IN fsbox_settings.json", rec.X+rec.Width*0.5, rec.Y+rec.Height-80, 9.5, 1.0, withAlpha(colMuted, 160))
}

// Small local wrappers keep the world icon code readable.
func mathCos(a float64) float64 { return math.Cos(a) }
func mathSin(a float64) float64 { return math.Sin(a) }
