package main

import (
	"fmt"

	fsbox "fsbox.dev/sdk"
)

//export mod_init
func modInit() {
	fmt.Println("[examplemod] init")
	raw, err := fsbox.Call("entity.spawn", map[string]any{
		"kind": "box",
		"position": map[string]float32{"x": 0, "y": 4, "z": 0},
		"color":    map[string]any{"r": 100, "g": 180, "b": 220, "a": 255},
	})
	fmt.Printf("[examplemod] spawn: %s err=%v\n", string(raw), err)

	info, err := fsbox.Call("engine.info", nil)
	fmt.Printf("[examplemod] engine info: %s err=%v\n", string(info), err)
}

//export mod_update
func modUpdate() {}

//export mod_event
func modEvent(ptr, size uint32) {}

func main() {}
