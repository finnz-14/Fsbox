//go:build !wasip1

package fsbox

import "encoding/json"

// Stub implementations for non-WASM platforms (IDE autocomplete / tests).
// On wasip1 the real implementations in wasm.go are used.

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

type EventHandler func(Event)
type TickHandler func()

func On(event string, h EventHandler) {}
func OnTick(h TickHandler)           {}

func Call(cmd string, args any) (json.RawMessage, error) { return nil, nil }
func Log(msg string)                                     {}
func Toast(msg string, c Color)                           {}
func SetEntityColor(id int32, c Color) error              { return nil }
func SpawnEntity(kind string, pos Vec3, color Color) (int32, error) {
	return 0, nil
}
func RandomColor() Color { return Color{R: 255, G: 255, B: 255, A: 255} }
