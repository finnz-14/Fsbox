// Jenny is a small example mod that shows how a content model, skeletal
// animations and a world-space billboard GUI can be composed from the SDK.
//
// It spawns one Jenny in front of the player when a map is running, then keeps
// a panel floating beside her. Aim the hand tool at a row to select a clip and
// press PLAY or STOP. H shows/hides the panel. The mouse cursor is never
// unlocked, so the normal engine menus keep working.
package main

import (
	"encoding/json"
	"fmt"
	"math"

	fsbox "fsbox.dev/sdk"
)

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

const (
	panelWidth  = float32(4.2)
	panelHeight = float32(2.9)

	rowCount   = 8
	rowHeight  = float32(0.20)
	rowSpacing = float32(0.24)
	firstRowY  = float32(0.92)

	panelOffsetX = float32(2.6)
	panelOffsetY = float32(1.0)
	spawnDist    = float32(5.0)
	interactDist = float32(12.0)

	animSpeed = float32(1.0)
)

var (
	jennyEntity int32
	spawned     bool
	nextTry     float64

	guiOpen   bool
	panelID   int32
	rowIDs    []int32
	playBtnID int32
	stopBtnID int32
	statusID  int32

	animations []fsbox.AnimationInfo
	scroll     int32
	selAnim    int32 = -1
	playing    int32 = -1
)

var (
	colAccent = fsbox.Color{R: 230, G: 166, B: 66, A: 255}
	colText   = fsbox.Color{R: 226, G: 226, B: 222, A: 255}
	colDark   = fsbox.Color{R: 10, G: 11, B: 12, A: 255}
	colMuted  = fsbox.Color{R: 150, G: 155, B: 160, A: 255}
	colRow    = fsbox.Color{R: 34, G: 37, B: 42, A: 235}
	colPlay   = fsbox.Color{R: 70, G: 130, B: 74, A: 245}
	colStop   = fsbox.Color{R: 168, G: 62, B: 62, A: 245}
)

type eventData struct {
	ID    int32  `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Panel int32  `json:"panel"`
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

func init() {
	fsbox.On("game.start", func(e fsbox.Event) {
		fsbox.Log("[Jenny] mod ready.")
	})

	// A reset destroys every entity, including Jenny. Forget the old state so
	// the next in-world tick spawns a fresh copy.
	fsbox.On("world.reset", func(e fsbox.Event) {
		forgetWorld()
	})

	fsbox.OnTick(tick)

	fsbox.On("ui.click", func(e fsbox.Event) {
		var d eventData
		if err := json.Unmarshal(e.Data, &d); err != nil {
			return
		}
		if d.Kind != "billboard_button" {
			return
		}
		onButton(d.ID)
	})
}

func forgetWorld() {
	hidePanel()
	jennyEntity = 0
	spawned = false
	nextTry = 0
	guiOpen = false
	animations = nil
	scroll = 0
	selAnim = -1
	playing = -1
}

// inWorld reports whether the engine is showing a loaded map. Menus and the
// pause screen must not spawn anything; every other state still simulates.
func inWorld(state string) bool {
	switch state {
	case "menu", "map_menu", "paused":
		return false
	}
	return true
}

func tick() {
	info, err := fsbox.GetEngineInfo()
	if err != nil {
		return
	}
	if !inWorld(info.State) {
		return
	}

	if !spawned {
		trySpawn(info)
		if !spawned {
			return
		}
	}

	ent, err := fsbox.GetEntity(jennyEntity)
	if err != nil || ent.ID == 0 {
		// Something removed Jenny (map change, delete tool, load). Rebuild on
		// the next tick instead of leaving a panel pointing at nothing.
		forgetWorld()
		return
	}

	if pressed, _ := fsbox.InputKeyPressed(fsbox.KeyH); pressed {
		togglePanel()
	}

	player, _ := fsbox.GetPlayerInfo()
	show := guiOpen && panelShouldShow(ent.Position, player)

	if show && panelID == 0 {
		buildPanel()
	}
	if !show && panelID != 0 {
		hidePanel()
	}
	if panelID == 0 {
		return
	}

	// The panel lives beside Jenny in world space and always faces the camera.
	_ = fsbox.MoveBillboardPanel(
		panelID,
		ent.Position.X+panelOffsetX,
		ent.Position.Y+panelOffsetY,
		ent.Position.Z,
	)

	// Only take the wheel while actually playing and in range, so the spawn
	// and tool menus keep their own scrolling.
	if guiOpen && info.State == "playing" && panelShouldShow(ent.Position, player) {
		handleScroll()
	}
}

func trySpawn(info fsbox.EngineInfo) {
	if info.Time < nextTry {
		return
	}
	nextTry = info.Time + 1.0

	player, err := fsbox.GetPlayerInfo()
	if err != nil {
		return
	}

	// Use a horizontal forward vector so looking at the floor does not bury
	// Jenny underground.
	fx, fz := player.Forward.X, player.Forward.Z
	l := float32(math.Sqrt(float64(fx*fx + fz*fz)))
	if l < 1e-4 {
		fx, fz = 0, -1
	} else {
		fx /= l
		fz /= l
	}

	pos := fsbox.Vec3{
		X: player.Position.X + fx*spawnDist,
		Y: player.Position.Y + 0.1,
		Z: player.Position.Z + fz*spawnDist,
	}
	id, err := fsbox.SpawnPreset("Jenny", pos, fsbox.Color{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		fsbox.Log("[Jenny] spawn failed: " + err.Error())
		return
	}
	if id == 0 {
		return
	}

	jennyEntity = id
	spawned = true
	guiOpen = true
	animations, _ = fsbox.GetAnimations(jennyEntity)
	scroll = 0
	selAnim = -1
	playing = -1
	buildPanel()
	fsbox.Log(fmt.Sprintf("[Jenny] spawned entity %d with %d animations", id, len(animations)))
}

// panelShouldShow decides whether the browser should exist right now. The panel
// is world-space, but keeping it alive after the player has walked away or
// turned their back leaves an invisible panel consuming input. The mod removes
// it and rebuilds it when the player comes back into view. Hysteresis stops the
// panel flickering while strafing near the edge of the interaction range.
func panelShouldShow(entPos fsbox.Vec3, player fsbox.PlayerInfo) bool {
	px := entPos.X + panelOffsetX
	py := entPos.Y + panelOffsetY
	pz := entPos.Z

	dx := px - player.Position.X
	dy := py - (player.Position.Y + 1.5)
	dz := pz - player.Position.Z
	dist2 := dx*dx + dy*dy + dz*dz

	maxDist := float32(interactDist)
	dotMin := float32(0.35)
	if panelID != 0 {
		maxDist += 2.5
		dotMin = 0.10
	}
	if dist2 > maxDist*maxDist {
		return false
	}

	l := float32(math.Sqrt(float64(dist2)))
	if l < 0.001 {
		return true
	}
	fx, fy, fz := player.Forward.X, player.Forward.Y, player.Forward.Z
	fl := float32(math.Sqrt(float64(fx*fx + fy*fy + fz*fz)))
	if fl < 1e-4 {
		return true
	}
	dot := (dx*fx + dy*fy + dz*fz) / (l * fl)
	return dot >= dotMin
}

// ---------------------------------------------------------------------------
// Panel
// ---------------------------------------------------------------------------

func buildPanel() {
	ent, err := fsbox.GetEntity(jennyEntity)
	if err != nil {
		return
	}
	if len(animations) == 0 {
		animations, _ = fsbox.GetAnimations(jennyEntity)
	}

	_ = fsbox.ClearBillboardPanels()
	rowIDs = nil
	playBtnID = 0
	stopBtnID = 0
	statusID = 0

	px := ent.Position.X + panelOffsetX
	py := ent.Position.Y + panelOffsetY
	pz := ent.Position.Z

	panel, err := fsbox.AddBillboardPanel(px, py, pz, panelWidth, panelHeight)
	if err != nil {
		fsbox.Log("[Jenny] add panel failed: " + err.Error())
		return
	}
	panelID = panel

	_, _ = fsbox.AddBillboardTextWorld(panelID, 0, 1.30, "JENNY", 0.24, colAccent)
	_, _ = fsbox.AddBillboardTextWorld(panelID, 0, 1.10, "ANIMATION BROWSER", 0.11, colMuted)

	statusID, _ = fsbox.AddBillboardTextWorld(panelID, 0, -0.96, "AIM AT LIST + WHEEL TO SCROLL  |  SELECT THEN PLAY", 0.09, colMuted)

	for i := 0; i < rowCount; i++ {
		y := firstRowY - float32(i)*rowSpacing
		id, err := fsbox.AddBillboardButtonWorld(
			panelID, 0, y, 3.8, rowHeight, "", colRow, colText, 0.11,
		)
		if err != nil {
			fsbox.Log("[Jenny] add row failed: " + err.Error())
			continue
		}
		rowIDs = append(rowIDs, id)
	}

	playBtnID, _ = fsbox.AddBillboardButtonWorld(
		panelID, -1.0, -1.24, 1.6, 0.30, "PLAY", colPlay, colText, 0.13,
	)
	stopBtnID, _ = fsbox.AddBillboardButtonWorld(
		panelID, 1.0, -1.24, 1.6, 0.30, "STOP", colStop, colText, 0.13,
	)

	refreshRows()
	fsbox.Log(fmt.Sprintf("[Jenny] panel built with %d animations", len(animations)))
}

func togglePanel() {
	guiOpen = !guiOpen
	if !guiOpen {
		hidePanel()
	}
}

// hidePanel removes the world-space panel but keeps guiOpen intact, so the
// browser can be rebuilt when the player returns or looks back at Jenny.
func hidePanel() {
	if panelID != 0 {
		_ = fsbox.RemoveBillboardPanel(panelID)
	}
	panelID = 0
	rowIDs = nil
	playBtnID = 0
	stopBtnID = 0
	statusID = 0
}

func refreshRows() {
	maxScroll := int32(len(animations) - rowCount)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	for i, id := range rowIDs {
		idx := scroll + int32(i)
		if idx >= 0 && int(idx) < len(animations) {
			name := animations[idx].Name
			label := fmt.Sprintf("%2d  %s", idx+1, name)
			bg := colRow
			text := colText
			if idx == selAnim {
				bg = colAccent
				text = colDark
			}
			if idx == playing {
				label = "> " + label
				if idx != selAnim {
					text = colAccent
				}
			}
			_ = fsbox.UpdateBillboardElement(id, fsbox.BillboardElementUpdate{
				Label: &label, Color: &bg, TextColor: &text,
			})
			continue
		}
		empty := ""
		bg := fsbox.Color{R: 20, G: 22, B: 26, A: 1}
		text := fsbox.Color{R: 20, G: 22, B: 26, A: 1}
		_ = fsbox.UpdateBillboardElement(id, fsbox.BillboardElementUpdate{
			Label: &empty, Color: &bg, TextColor: &text,
		})
	}
}

func setStatus(text string, c fsbox.Color) {
	if statusID == 0 {
		return
	}
	_ = fsbox.UpdateBillboardElement(statusID, fsbox.BillboardElementUpdate{
		Label: &text, TextColor: &c,
	})
}

// ---------------------------------------------------------------------------
// Interaction
// ---------------------------------------------------------------------------

func handleScroll() {
	// Only scroll while the crosshair is actually on the panel. Otherwise the
	// wheel belongs to whatever the player is looking at (or another menu).
	hover, err := fsbox.GetBillboardHover()
	if err != nil || hover.Panel != panelID {
		return
	}
	w, err := fsbox.InputWheel()
	if err != nil || w == 0 {
		return
	}
	maxScroll := int32(len(animations) - rowCount)
	if maxScroll < 0 {
		maxScroll = 0
	}
	old := scroll
	if w > 0 {
		scroll--
	} else {
		scroll++
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll != old {
		refreshRows()
	}
}

func onButton(id int32) {
	if id == playBtnID {
		playSelected()
		return
	}
	if id == stopBtnID {
		stopSelected()
		return
	}
	for row, rowID := range rowIDs {
		if rowID != id {
			continue
		}
		idx := scroll + int32(row)
		if idx < 0 || int(idx) >= len(animations) {
			return
		}
		selAnim = idx
		refreshRows()
		setStatus("SELECTED: "+animations[idx].Name, colText)
		return
	}
}

func playSelected() {
	if jennyEntity == 0 || len(animations) == 0 {
		return
	}
	if selAnim < 0 || int(selAnim) >= len(animations) {
		selAnim = scroll
		if int(selAnim) >= len(animations) {
			selAnim = 0
		}
	}
	if err := fsbox.PlayAnimationByIndex(jennyEntity, selAnim, animSpeed); err != nil {
		fsbox.Log("[Jenny] play failed: " + err.Error())
		return
	}
	playing = selAnim
	refreshRows()
	setStatus("NOW PLAYING: "+animations[selAnim].Name, colAccent)
	fsbox.Log("[Jenny] playing " + fmt.Sprint(selAnim) + " = " + animations[selAnim].Name)
}

func stopSelected() {
	if jennyEntity == 0 {
		return
	}
	if err := fsbox.StopAnimation(jennyEntity); err != nil {
		fsbox.Log("[Jenny] stop failed: " + err.Error())
		return
	}
	playing = -1
	refreshRows()
	setStatus("STOPPED", colMuted)
}

func main() {}
