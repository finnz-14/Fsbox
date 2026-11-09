package main

import (
	"encoding/json"
	"unsafe"

	fsbox "fsbox.dev/sdk"
)

//export mod_init
func modInit() {}

//export mod_event
func modEvent(ptr, size uint32) {
	if size == 0 {
		return
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	var ev struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(buf, &ev); err != nil {
		return
	}
	if ev.Event != "entity.grab" {
		return
	}
	var data struct {
		ID   int32  `json:"id"`
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		return
	}
	// Debug: notify engine.
	_, _ = fsbox.Call("engine.log", map[string]any{"message": "crate grabbed: " + data.Name})
	// Only recolor crates/boxes.
	if data.Kind != "box" {
		return
	}
	// Change to a bright red.
	_, _ = fsbox.Call("entity.set_color", map[string]any{
		"id":    data.ID,
		"color": map[string]any{"r": 220, "g": 60, "b": 60, "a": 255},
	})
}

//export mod_update
func modUpdate() {}

func main() {}
