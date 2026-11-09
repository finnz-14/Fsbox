package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	fsbox "fsbox.dev/sdk"
)

var (
	jennyEntity int32
	uiCreated   bool
	uiPlayBtn   int32
	uiStopBtn   int32
	uiAnimBtns  []int32
	animSpeed   float32 = 1.0
	spawned     bool
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
			// Spawn Jenny by preset name on first tick.
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

		// Create UI once Jenny exists.
		if !uiCreated {
			buildUI()
			uiCreated = true
		}
	})

	fsbox.On("ui.click", func(e fsbox.Event) {
		var d eventData
		if err := json.Unmarshal(e.Data, &d); err != nil {
			return
		}

		// Animation buttons: match by button ID.
		for i, btn := range uiAnimBtns {
			if d.ID == btn {
				if err := fsbox.PlayAnimationByIndex(jennyEntity, int32(i), animSpeed); err != nil {
					fsbox.Log("[Jenny] play failed: " + err.Error())
				} else {
					fsbox.Log("[Jenny] playing animation " + strconv.Itoa(i))
				}
				return
			}
		}

		switch d.ID {
		case uiPlayBtn:
			if len(uiAnimBtns) > 0 {
				if err := fsbox.PlayAnimationByIndex(jennyEntity, 0, animSpeed); err != nil {
					fsbox.Log("[Jenny] play failed: " + err.Error())
				}
			}
		case uiStopBtn:
			if err := fsbox.StopAnimation(jennyEntity); err != nil {
				fsbox.Log("[Jenny] stop failed: " + err.Error())
			}
		}
	})
}

func buildUI() {
	// Title text.
	fsbox.Call("ui.draw_text", map[string]any{
		"text": "Jenny Animations",
		"x":    float32(40),
		"y":    float32(100),
		"size": 18,
		"color": map[string]any{"r": 230, "g": 166, "b": 66, "a": 255},
	})

	// Get animations and create one button per animation.
	anims, err := fsbox.GetAnimations(jennyEntity)
	if err != nil {
		fsbox.Log("[Jenny] get_animations failed: " + err.Error())
		return
	}

	y := float32(140)
	btnW := float32(200)
	btnH := float32(24)
	gap := float32(6)
	for i, anim := range anims {
		btn, _ := fsbox.AddButton(20, y, btnW, btnH, anim.Name, fsbox.Color{R: 40, G: 45, B: 50, A: 255})
		uiAnimBtns = append(uiAnimBtns, btn)
		fsbox.Log("[Jenny] button " + fmt.Sprint(i) + ": " + anim.Name + " (id=" + strconv.Itoa(int(btn)) + ")")
		y += btnH + gap
		if y > 400 {
			break
		}
	}

	// Play button (plays first animation).
	play, _ := fsbox.AddButton(20, y+12, 95, 30, "PLAY", fsbox.Color{R: 60, G: 120, B: 60, A: 255})
	uiPlayBtn = play

	// Stop button.
	stop, _ := fsbox.AddButton(125, y+12, 95, 30, "STOP", fsbox.Color{R: 160, G: 60, B: 60, A: 255})
	uiStopBtn = stop

	fsbox.Log("[Jenny] UI created with " + fmt.Sprint(len(anims)) + " animations")
}

func main() {}
