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

	panelID     int32
	btnIDs      []int32
	playBtnID   int32
	stopBtnID   int32
	animations  []fsbox.AnimationInfo
)

type eventData struct {
	ID    int32  `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Panel int32  `json:"panel"`
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

		// Auto-open billboard GUI when player approaches Jenny and looks at her.
		player, err := fsbox.GetPlayerInfo()
		if err == nil && jennyEntity > 0 {
			ent, err := fsbox.GetEntity(jennyEntity)
			if err == nil {
				jennyPos := ent.Position
				dx := jennyPos.X - player.Position.X
				dy := jennyPos.Y - player.Position.Y
				dz := jennyPos.Z - player.Position.Z
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
				toJenny := fsbox.Vec3{X: jennyPos.X - player.Position.X, Y: jennyPos.Y - player.Position.Y, Z: jennyPos.Z - player.Position.Z}
				if dist > 0 {
					toJenny = fsbox.Vec3{X: toJenny.X / dist, Y: toJenny.Y / dist, Z: toJenny.Z / dist}
				}
				dot := player.Forward.X*toJenny.X + player.Forward.Y*toJenny.Y + player.Forward.Z*toJenny.Z

				if dist < 6.0 && dot > 0.5 && !guiOpen {
					guiOpen = true
					fsbox.Toast("Jenny GUI - use hand tool to click", fsbox.Color{R: 230, G: 166, B: 66, A: 255})
					buildBillboard(jennyPos)
				} else if (dist > 7.0 || dot < 0.3) && guiOpen {
					guiOpen = false
					fsbox.ClearBillboardPanels()
					panelID = 0
					btnIDs = nil
					fsbox.Toast("Jenny GUI closed", fsbox.Color{R: 160, G: 60, B: 60, A: 255})
				}
			}
		}

		// Toggle GUI with 'E'.
		pressed, _ := fsbox.InputKeyPressed(fsbox.KeyE)
		if pressed {
			guiOpen = !guiOpen
			if guiOpen {
				fsbox.Toast("Jenny GUI - use hand tool to click", fsbox.Color{R: 230, G: 166, B: 66, A: 255})
				// Get current Jenny position for panel placement
				if ent, err := fsbox.GetEntity(jennyEntity); err == nil {
					buildBillboard(ent.Position)
				}
			} else {
				fsbox.ClearBillboardPanels()
				panelID = 0
				btnIDs = nil
				fsbox.Log("[Jenny] GUI closed.")
			}
		}
	})

	fsbox.On("ui.click", func(e fsbox.Event) {
		var d eventData
		if err := json.Unmarshal(e.Data, &d); err != nil {
			return
		}

		// Only handle billboard button clicks
		if d.Kind != "billboard_button" {
			return
		}

		if d.ID == playBtnID {
			if selAnim >= 0 && selAnim < int32(len(animations)) {
				if err := fsbox.PlayAnimationByIndex(jennyEntity, selAnim, animSpeed); err != nil {
					fsbox.Log("[Jenny] play failed: " + err.Error())
				} else {
					fsbox.Log("[Jenny] playing " + strconv.Itoa(int(selAnim)) + " = " + animations[selAnim].Name)
				}
			}
			return
		}

		if d.ID == stopBtnID {
			if err := fsbox.StopAnimation(jennyEntity); err != nil {
				fsbox.Log("[Jenny] stop failed: " + err.Error())
			}
			return
		}

		for i, btnID := range btnIDs {
			if d.ID == btnID {
				selAnim = int32(i)
				fsbox.Log("[Jenny] selected " + strconv.Itoa(i) + " = " + animations[i].Name)
				return
			}
		}
	})
}

// buildBillboard creates the 3D billboard panel floating beside Jenny.
func buildBillboard(jennyPos fsbox.Vec3) {
	fsbox.ClearBillboardPanels()
	panelID = 0
	btnIDs = nil

	// Panel floats 2m to Jenny's right, at head height (y + 1.5).
	px := jennyPos.X + 2.0
	py := jennyPos.Y + 1.5
	pz := jennyPos.Z

	panel, err := fsbox.AddBillboardPanel(px, py, pz, 2.5, 1.6)
	if err != nil {
		fsbox.Log("[Jenny] add billboard panel failed: " + err.Error())
		return
	}
	panelID = panel

	// Title at top of panel (centered).
	fsbox.AddBillboardText(panelID, 0, 0.85, "Jenny", 0.4, fsbox.Color{R: 230, G: 166, B: 66, A: 255})

	// Animation buttons - vertical list on left side of panel.
	btnIDs = nil
	btnW := float32(2.0)
	btnH := float32(0.14)
	gap := float32(0.015)
	startY := float32(0.6)

	for i := int32(0); i < int32(len(animations)); i++ {
		btnY := startY - float32(i)*(btnH+gap)
		if btnY < -0.65 {
			break
		}
		btn, err := fsbox.AddBillboardButton(panelID, -0.85, btnY, btnW, btnH, animations[i].Name, fsbox.Color{R: 40, G: 45, B: 50, A: 230})
		if err != nil {
			fsbox.Log("[Jenny] add button failed: " + err.Error())
			continue
		}
		btnIDs = append(btnIDs, btn)
	}

	// Play button bottom left.
	playBtn, err := fsbox.AddBillboardButton(panelID, -0.9, -0.75, 0.6, 0.08, "PLAY", fsbox.Color{R: 60, G: 120, B: 60, A: 230})
	if err != nil {
		fsbox.Log("[Jenny] add play button failed: " + err.Error())
	}
	playBtnID = playBtn

	// Stop button bottom right.
	stopBtn, err := fsbox.AddBillboardButton(panelID, 0.9, -0.75, 0.6, 0.08, "STOP", fsbox.Color{R: 160, G: 60, B: 60, A: 230})
	if err != nil {
		fsbox.Log("[Jenny] add stop button failed: " + err.Error())
	}
	stopBtnID = stopBtn

	fsbox.Log("[Jenny] billboard built with " + fmt.Sprint(len(animations)) + " animations")
}

func main() {}
