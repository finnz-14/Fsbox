//go:build wasip1

package fsbox

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"unsafe"
)

//go:wasmimport fsbox fsbox_call
func hostCall(reqPtr, reqLen, respPtr, respCap uint32) int32

var reqBuf [1 << 20]byte
var respBuf [1 << 20]byte

func reqPtr() uint32 { return uint32(uintptr(unsafe.Pointer(&reqBuf[0]))) }
func respPtr() uint32 { return uint32(uintptr(unsafe.Pointer(&respBuf[0]))) }

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

type Vec3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type Entity struct {
	ID           int32  `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Static       bool   `json:"static"`
	Frozen       bool   `json:"frozen"`
	Invulnerable bool   `json:"invulnerable"`
	Transient    bool   `json:"transient"`
	Health       float32 `json:"health"`
	MaxHealth    float32 `json:"max_health"`
	Position     Vec3   `json:"position"`
	Color        Color  `json:"color"`
	Half         Vec3   `json:"half"`
	Radius       float32 `json:"radius"`
	Height       float32 `json:"height"`
	Owner        string `json:"owner"`
}

type Event struct {
	Name   string
	Data   json.RawMessage
	Entity Entity
}

// ---------------------------------------------------------------------------
// Event / tick handlers
// ---------------------------------------------------------------------------

type EventHandler func(Event)
type TickHandler func()

var eventHandlers = map[string][]EventHandler{}
var tickHandlers []TickHandler

func On(event string, h EventHandler) {
	eventHandlers[event] = append(eventHandlers[event], h)
}

func OnTick(h TickHandler) {
	tickHandlers = append(tickHandlers, h)
}

// ---------------------------------------------------------------------------
// Generic command call
// ---------------------------------------------------------------------------

func Call(cmd string, args any) (json.RawMessage, error) {
	req := map[string]any{"cmd": cmd, "args": args}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if len(reqBytes) > len(reqBuf) {
		return nil, fmt.Errorf("request too large")
	}
	copy(reqBuf[:], reqBytes)
	written := hostCall(reqPtr(), uint32(len(reqBytes)), respPtr(), uint32(len(respBuf)))
	if written < 0 {
		return nil, fmt.Errorf("host call failed")
	}
	if int(written) > len(respBuf) {
		return nil, fmt.Errorf("response too large: %d", written)
	}
	var resp struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(respBuf[:written], &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Result, nil
}

// ---------------------------------------------------------------------------
// Typed helpers
// ---------------------------------------------------------------------------

func Log(msg string) {
	_, _ = Call("engine.log", map[string]any{"message": msg})
}

func Toast(msg string, c Color) {
	_, _ = Call("ui.toast", map[string]any{"message": msg, "color": c})
}

func SetEntityColor(id int32, c Color) error {
	_, err := Call("entity.set_color", map[string]any{"id": id, "color": c})
	return err
}

func SpawnEntity(kind string, pos Vec3, color Color) (int32, error) {
	raw, err := Call("entity.spawn", map[string]any{"kind": kind, "position": pos, "color": color})
	if err != nil {
		return 0, err
	}
	var info struct {
		ID int32 `json:"id"`
	}
	_ = json.Unmarshal(raw, &info)
	return info.ID, nil
}

func RandomColor() Color {
	return Color{
		R: uint8(rand.Intn(200) + 55),
		G: uint8(rand.Intn(200) + 55),
		B: uint8(rand.Intn(200) + 55),
		A: 255,
	}
}

// ---------------------------------------------------------------------------
// Exported entry points (called by the engine)
// ---------------------------------------------------------------------------

//go:wasmexport mod_init
func ModInit() {}

//go:wasmexport mod_update
func ModUpdate() {
	for _, h := range tickHandlers {
		h()
	}
}

//go:wasmexport mod_event
func ModEvent(ptr, size uint32) {
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
	e := Event{Name: ev.Event, Data: ev.Data}
	var ent Entity
	if err := json.Unmarshal(ev.Data, &ent); err == nil && ent.ID != 0 {
		e.Entity = ent
	} else {
		var wrap struct {
			Entity json.RawMessage `json:"entity"`
		}
		if json.Unmarshal(ev.Data, &wrap) == nil && len(wrap.Entity) > 0 {
			_ = json.Unmarshal(wrap.Entity, &ent)
			e.Entity = ent
		}
	}
	if hs, ok := eventHandlers[ev.Event]; ok {
		for _, h := range hs {
			h(e)
		}
	}
}

//go:wasmexport mod_alloc
func ModAlloc(size uint32) uint32 { return respPtr() }

//go:wasmexport mod_scratch
func ModScratch(size uint32) uint32 { return respPtr() }
