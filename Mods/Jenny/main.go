package main

import (
	"encoding/json"

	fsbox "fsbox.dev/sdk"
)

var (
	jennyEntity int32
	spawned     bool
)

type spawnResult struct {
	ID int32 `json:"id"`
}

func init() {
	fsbox.On("game.start", func(e fsbox.Event) {
		fsbox.Log("[Jenny] mod ready.")
	})

	fsbox.OnTick(func() {
		if spawned {
			return
		}
		spawned = true

		// Spawn Jenny by preset name — animations come from the preset.
		raw, err := fsbox.Call("entity.spawn", map[string]any{
			"preset_name": "Jenny",
			"position":    map[string]float32{"x": 0, "y": 2, "z": 4},
			"color":       map[string]any{"r": 255, "g": 255, "b": 255, "a": 255},
		})
		if err != nil {
			fsbox.Log("[Jenny] failed to spawn: " + err.Error())
			return
		}

		var result spawnResult
		if err := json.Unmarshal(raw, &result); err != nil {
			fsbox.Log("[Jenny] failed to parse spawn result: " + err.Error())
			return
		}
		jennyEntity = result.ID

		fsbox.Log("[Jenny] spawned at (0, 2, 4). Walk up to her to see the animation GUI.")
		fsbox.Log("[Jenny] Equip the HAND tool (slot 1) to click animations.")
	})
}

func main() {}
