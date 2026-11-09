package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	fsbox "fsbox.dev/sdk"
)

var (
	jennyEntity int32
	guiOpen     bool
	scrollOff   float32 = 0
	selAnim     int32 = -1
	animSpeed   float32 = 1.0
	spawned     bool

	uiAll       []int32
	uiPanel     int32
	uiCloseBtn  int32
	uiPlayBtn   int32
	uiStopBtn   int32
	uiAnimBtns  []int32
	animations  []fsbox.AnimationInfo
)

type eventData struct {
	ID    int32  `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

func init() {
	fsbox.On("game.start", func(e fsbox.Event) {
		fsbox.Log("[Jenny] mod ready.")
	})

	fsbox.OnTick(func() {
		if !spawned {
			raw, err := fsbox.Call("entity.spawn", map[string]any{
				"preset_name": "Jenny",
				"position":    map[string]float32{"x": 0, "y": 2, "z": 4},
				"color":       map[string]any{"r": 255, "g": 255, "b": 255, "a": 255},
			})
			if err != nil {
				fsbox.Log("[Jenny] spawn failed: " + err.Error())
				return
			}
			var res struct {
				ID int32 `json:"id"`
			}
			if err := json.Unmarshal(raw, &res); err != nil {
				fsbox.Log("[Jenny] parse failed: " + err.Error())
				return
			}
			jennyEntity = res.ID
			spawned = true
			fsbox.Log("[Jenny] spawned entity " + fmt.Sprint(jennyEntity))
			return
		}

		// Refresh animation list every tick.
		anims, err := fsbox.GetAnimations(jennyEntity)
		if err != nil {
			return
		}
		animations = anims

		// Auto-open GUI when player approaches Jenny and looks at her.
		player, err := fsbox.GetPlayerInfo()
		if err == nil && jennyEntity > 0 {
			// Get Jenny's actual position from the entity.
			eInfo, err := fsbox.GetEntity(jennyEntity)
			if err == nil {
				jennyPos := eInfo.Position
				// Distance check.
				dx := jennyPos.X - player.Position.X
				dy := jennyPos.Y - player.Position.Y
				dz := jennyPos.Z - player.Position.Z
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
				// Dot product: is Jenny in front of the player?
				toJenny := fsbox.Vec3{X: jennyPos.X - player.Position.X, Y: jennyPos.Y - player.Position.Y, Z: jennyPos.Z - player.Position.Z}
				toJenny = fsbox.Vec3{X: toJenny.X / dist, Y: toJenny.Y / dist, Z: toJenny.Z / dist}
				dot := player.Forward.X*toJenny.X + player.Forward.Y*toJenny.Y + player.Forward.Z*toJenny.Z

				if dist < 6.0 && dot > 0.5 && !guiOpen {
					guiOpen = true
					fsbox.SetCursorLocked(false)
					buildUI()
					fsbox.Toast("Jenny GUI opened - click animations to play", fsbox.Color{R: 230, G: 166, B: 66, A: 255})
				} else if (dist > 7.0 || dot < 0.3) && guiOpen {
					guiOpen = false
					fsbox.SetCursorLocked(true)
					cleanupUI()
					fsbox.Toast("Jenny GUI closed", fsbox.Color{R: 160, G: 60, B: 60, A: 255})
				}
			}
		}

		// Toggle GUI with 'E' as well.
		pressed, _ := fsbox.InputKeyPressed(69) // KEY_E
		if pressed {
			guiOpen = !guiOpen
			if guiOpen {
				fsbox.SetCursorLocked(false)
				buildUI()
				fsbox.Log("[Jenny] GUI opened (" + fmt.Sprint(len(animations)) + " animations).")
			} else {
				fsbox.SetCursorLocked(true)
				cleanupUI()
				fsbox.Log("[Jenny] GUI closed.")
			}
		}

		if guiOpen {
			// Mouse wheel scroll.
			wheel, _ := fsbox.InputWheel()
			if wheel != 0 {
				scrollOff -= wheel * 24
				if scrollOff < 0 {
					scrollOff = 0
				}
				maxScroll := float32(len(animations))*26 - 200
				if maxScroll < 0 {
					maxScroll = 0
				}
				if scrollOff > maxScroll {
					scrollOff = maxScroll
				}
				updatePanelPosition()
			}
		}
	})

	fsbox.On("ui.click", func(e fsbox.Event) {
		var d eventData
		if err := json.Unmarshal(e.Data, &d); err != nil {
			return
		}

		if d.ID == uiCloseBtn {
			guiOpen = false
			fsbox.SetCursorLocked(true)
			cleanupUI()
			fsbox.Log("[Jenny] GUI closed.")
			return
		}

		if d.ID == uiPlayBtn {
			if selAnim >= 0 && selAnim < int32(len(animations)) {
				if err := fsbox.PlayAnimationByIndex(jennyEntity, selAnim, animSpeed); err != nil {
					fsbox.Log("[Jenny] play failed: " + err.Error())
				} else {
					fsbox.Log("[Jenny] playing index " + strconv.Itoa(int(selAnim)) + " = " + animations[selAnim].Name)
				}
			}
			return
		}

		if d.ID == uiStopBtn {
			if err := fsbox.StopAnimation(jennyEntity); err != nil {
				fsbox.Log("[Jenny] stop failed: " + err.Error())
			}
			return
		}

		for i, btnID := range uiAnimBtns {
			if d.ID == btnID {
				selAnim = int32(i)
				fsbox.Log("[Jenny] selected index " + strconv.Itoa(i) + " = " + animations[i].Name)
				return
			}
		}
	})
}

// buildUI creates all UI elements once when the GUI opens.
func buildUI() {
	cleanupUI()
	uiAll = uiAll[:0]
	uiAnimBtns = uiAnimBtns[:0]

	screen, err := fsbox.WorldToScreen(fsbox.Vec3{X: 0, Y: 3, Z: 4})
	if err != nil {
		fsbox.Log("[Jenny] world_to_screen failed: " + err.Error())
		return
	}

	// Position panel to the right of Jenny on screen, clamped to screen bounds.
	sw := float32(1280) // default screen width
	sh := float32(720)  // default screen height
	panelW := float32(240)
	panelH := float32(340)

	px := screen.X + 60
	if px+panelW > sw {
		px = screen.X - panelW - 10
	}
	if px < 0 {
		px = 10
	}
	py := screen.Y - panelH/2
	if py < 0 {
		py = 10
	}
	if py+panelH > sh {
		py = sh - panelH - 10
	}

	// Panel background.
	panel, _ := fsbox.AddButton(px, py, panelW, panelH, "", fsbox.Color{R: 20, G: 24, B: 30, A: 230})
	uiPanel = panel
	uiAll = append(uiAll, panel)

	// Close button (top-right of panel).
	closeBtn, _ := fsbox.AddButton(px+panelW-30, py+8, 22, 22, "X", fsbox.Color{R: 160, G: 60, B: 60, A: 255})
	uiCloseBtn = closeBtn
	uiAll = append(uiAll, closeBtn)

	// Title.
	fsbox.Call("ui.draw_text", map[string]any{
		"text": "Jenny Animations",
		"x":    px + 10,
		"y":    py + 14,
		"size": 16,
		"color": map[string]any{"r": 230, "g": 166, "b": 66, "a": 255},
	})

	// Create animation buttons (visible ones based on scroll).
	btnW := float32(200)
	btnH := float32(22)
	gap := float32(4)
	startY := py + 44

	for i := int32(0); i < int32(len(animations)); i++ {
		animY := startY + float32(i)*(btnH+gap) - scrollOff
		if animY+btnH < py+40 || animY > py+panelH-40 {
			continue
		}
		btn, _ := fsbox.AddButton(px+10, animY, btnW, btnH, animations[i].Name, fsbox.Color{R: 40, G: 45, B: 50, A: 255})
		uiAnimBtns = append(uiAnimBtns, btn)
		uiAll = append(uiAll, btn)
	}

	// Play / Stop buttons at bottom.
	playY := py + panelH - 40
	play, _ := fsbox.AddButton(px+10, playY, 105, 30, "PLAY", fsbox.Color{R: 60, G: 120, B: 60, A: 255})
	uiPlayBtn = play
	uiAll = append(uiAll, play)

	stop, _ := fsbox.AddButton(px+panelW-115, playY, 105, 30, "STOP", fsbox.Color{R: 160, G: 60, B: 60, A: 255})
	uiStopBtn = stop
	uiAll = append(uiAll, stop)

	// Selected animation label.
	if selAnim >= 0 && selAnim < int32(len(animations)) {
		fsbox.Call("ui.draw_text", map[string]any{
			"text": "Selected: " + animations[selAnim].Name,
			"x":    px + 10,
			"y":    playY + 34,
			"size": 12,
			"color": map[string]any{"r": 200, "g": 200, "b": 200, "a": 255},
		})
	}

	fsbox.Log("[Jenny] UI built with " + fmt.Sprint(len(animations)) + " animations")
}

// updatePanelPosition moves the panel to follow Jenny's screen position.
func updatePanelPosition() {
	screen, err := fsbox.WorldToScreen(fsbox.Vec3{X: 0, Y: 3, Z: 4})
	if err != nil {
		return
	}

	sw := float32(1280)
	sh := float32(720)
	panelW := float32(240)
	panelH := float32(340)

	px := screen.X + 60
	if px+panelW > sw {
		px = screen.X - panelW - 10
	}
	if px < 0 {
		px = 10
	}
	py := screen.Y - panelH/2
	if py < 0 {
		py = 10
	}
	if py+panelH > sh {
		py = sh - panelH - 10
	}

	if uiPanel != 0 {
		fsbox.Call("ui.set_element", map[string]any{
			"id": uiPanel,
			"x":  &px, "y":  &py,
			"w":  float32Ptr(panelW), "h": float32Ptr(panelH),
		})
	}
	if uiCloseBtn != 0 {
		cx := px + panelW - 30
		cy := py + 8
		fsbox.Call("ui.set_element", map[string]any{"id": uiCloseBtn, "x": &cx, "y": &cy})
	}

	// Update animation button positions.
	btnH := float32(22)
	gap := float32(4)
	startY := py + 44

	for i, btnID := range uiAnimBtns {
		if i >= len(animations) {
			continue
		}
		animY := startY + float32(i)*(btnH+gap) - scrollOff
		if animY+btnH < py+40 || animY > py+panelH-40 {
			fsbox.Call("ui.set_element", map[string]any{"id": btnID, "x": float32Ptr(-9999), "y": float32Ptr(-9999)})
			continue
		}
		bx := px + 10
		by := animY
		fsbox.Call("ui.set_element", map[string]any{"id": btnID, "x": &bx, "y": &by})
	}

	if uiPlayBtn != 0 {
		playY := py + panelH - 40
		fsbox.Call("ui.set_element", map[string]any{"id": uiPlayBtn, "x": &px, "y": &playY})
	}
	if uiStopBtn != 0 {
		playY := py + panelH - 40
		sx := px + panelW - 115
		sy := playY
		fsbox.Call("ui.set_element", map[string]any{"id": uiStopBtn, "x": &sx, "y": &sy})
	}
}

// cleanupUI removes all UI elements created by this mod.
func cleanupUI() {
	for _, id := range uiAll {
		fsbox.Call("ui.remove_element", map[string]any{"id": id})
	}
	uiAll = uiAll[:0]
	uiAnimBtns = uiAnimBtns[:0]
	uiPanel = 0
	uiCloseBtn = 0
	uiPlayBtn = 0
	uiStopBtn = 0
	selAnim = -1
	scrollOff = 0
}

func float32Ptr(v float32) *float32 {
	return &v
}

func main() {}
