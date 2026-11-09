package main

import (
	"encoding/json"
	"math/rand"
	"unsafe"

	fsbox "fsbox.dev/sdk"
)

//go:wasmexport mod_init
func modInit() {}

//go:wasmexport mod_event
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
	// Change to a random color.
	r := uint8(rand.Intn(200) + 55)
	g := uint8(rand.Intn(200) + 55)
	b := uint8(rand.Intn(200) + 55)
	raw, err := fsbox.Call("entity.set_color", map[string]any{
		"id":    data.ID,
		"color": map[string]any{"r": r, "g": g, "b": b, "a": 255},
	})
	_, _ = fsbox.Call("engine.log", map[string]any{"message": "set_color result: " + string(raw) + " err=" + errStr(err)})
}

//go:wasmexport mod_update
func modUpdate() {}

func errStr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func main() {}
