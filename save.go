package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	savePath    = "fsbox_save.json"
	saveVersion = 1
)

type vec3Save struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type quatSave struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	W float32 `json:"w"`
}

type colorSave struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

type playerSave struct {
	Position vec3Save `json:"position"`
	Yaw      float32  `json:"yaw"`
	Pitch    float32  `json:"pitch"`
	Fly      bool     `json:"fly"`
}

type propSave struct {
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Position    vec3Save  `json:"position"`
	Rotation    quatSave  `json:"rotation"`
	Color       colorSave `json:"color"`
	Half        vec3Save  `json:"half,omitempty"`
	Radius      float32   `json:"radius,omitempty"`
	Height      float32   `json:"height,omitempty"`
	Density     float32   `json:"density"`
	Friction    float32   `json:"friction"`
	Restitution float32   `json:"restitution"`
	Frozen      bool      `json:"frozen,omitempty"`
}

type saveFile struct {
	Version  int        `json:"version"`
	SavedAt  string     `json:"saved_at"`
	Selected int        `json:"selected"`
	Player   playerSave `json:"player"`
	Props    []propSave `json:"props"`
}

func (g *Game) saveGame() {
	p := g.player.body.GetPosition()
	sf := saveFile{
		Version:  saveVersion,
		SavedAt:  time.Now().UTC().Format(time.RFC3339),
		Selected: g.selected,
		Player: playerSave{
			Position: vec3Save{X: p.X, Y: p.Y, Z: p.Z},
			Yaw:      g.player.yaw,
			Pitch:    g.player.pitch,
			Fly:      g.player.fly,
		},
		Props: make([]propSave, 0, len(g.props)),
	}

	for _, e := range g.props {
		bp := e.Body.GetPosition()
		q := e.Body.GetRotation()
		ps := propSave{
			Kind:     e.Kind.String(),
			Name:     e.Name,
			Position: vec3Save{X: bp.X, Y: bp.Y, Z: bp.Z},
			Rotation: quatSave{X: q.V.X, Y: q.V.Y, Z: q.V.Z, W: q.S},
			Color:    colorSave{R: e.Color.R, G: e.Color.G, B: e.Color.B, A: e.Color.A},
			Half:     vec3Save{X: e.Half.X, Y: e.Half.Y, Z: e.Half.Z},
			Radius:   e.Radius,
			Height:   e.Height,
			Density:  e.Density,
			Friction: e.Friction,
			// Keep the JSON future-proof and explicit.
			Restitution: e.Restitution,
			Frozen:      e.Frozen,
		}
		sf.Props = append(sf.Props, ps)
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		g.toast("Save failed: "+err.Error(), colDanger)
		return
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		g.toast("Save failed: "+err.Error(), colDanger)
		return
	}
	g.toast(fmt.Sprintf("Saved %d props to %s", len(sf.Props), savePath), colAccent)
}

func (g *Game) loadGame() {
	data, err := os.ReadFile(savePath)
	if err != nil {
		g.toast("No save found at "+savePath, colWarn)
		return
	}

	var sf saveFile
	if err := json.Unmarshal(data, &sf); err != nil {
		g.toast("Load failed: "+err.Error(), colDanger)
		return
	}
	if sf.Version != saveVersion {
		g.toast(fmt.Sprintf("Unsupported save version %d", sf.Version), colDanger)
		return
	}

	spawn := defaultSpawn
	p := sf.Player.Position
	if p.X > -28 && p.X < 28 && p.Z > -28 && p.Z < 28 && p.Y > 0.4 && p.Y < 40 {
		spawn = bp(p.X, p.Y, p.Z)
	}
	yaw := sf.Player.Yaw
	pitch := clampf(sf.Player.Pitch, -1.52, 1.52)

	g.resetWorld(false, spawn, yaw, pitch, sf.Player.Fly)

	count := 0
	limit := len(sf.Props)
	if limit > maxProps {
		limit = maxProps
	}
	for i := 0; i < limit; i++ {
		if g.spawnSaved(sf.Props[i]) {
			count++
		}
	}
	g.selected = clampInt(sf.Selected, 0, len(presets)-1)
	g.toast(fmt.Sprintf("Loaded %d props from %s", count, savePath), colAccent)
}

func (g *Game) spawnSaved(ps propSave) bool {
	kind, ok := propKindFromString(ps.Kind)
	if !ok {
		return false
	}
	if ps.Position.X < -40 || ps.Position.X > 40 || ps.Position.Z < -40 || ps.Position.Z > 40 {
		return false
	}
	if ps.Position.Y < -4 || ps.Position.Y > 60 {
		return false
	}

	pos := bp(ps.Position.X, ps.Position.Y, ps.Position.Z)
	rot := normalizeQuat(box3d.Quat{
		V: bv(ps.Rotation.X, ps.Rotation.Y, ps.Rotation.Z),
		S: ps.Rotation.W,
	})

	c := rl.Color{R: ps.Color.R, G: ps.Color.G, B: ps.Color.B, A: ps.Color.A}
	if c.A == 0 {
		c.A = 255
	}

	fallback := presetForKind(kind)
	name := ps.Name
	if name == "" {
		name = fallback.Name
	}
	density := ps.Density
	if density <= 0 {
		density = fallback.Density
	}
	friction := ps.Friction
	if friction <= 0 {
		friction = fallback.Friction
	}

	var e *Entity
	switch kind {
	case PropBox:
		half := bv(ps.Half.X, ps.Half.Y, ps.Half.Z)
		if half.X <= 0.01 || half.Y <= 0.01 || half.Z <= 0.01 {
			half = fallback.Half
		}
		// Restore content models (and their hull colliders) by matching the
		// saved prop name against a loaded content preset.
		var contentPreset *ShapePreset
		for i := range presets {
			if presets[i].HasModel && presets[i].Name == name {
				cp := presets[i]
				contentPreset = &cp
				break
			}
		}
		if contentPreset != nil {
			if len(contentPreset.Hulls) > 0 {
				e = g.addHullsEntity(false, pos, rot, half, c, name, density, friction, ps.Restitution, contentPreset.Hulls, contentPreset.Model, true)
			} else if contentPreset.Hull != nil {
				e = g.addHullEntity(false, pos, rot, half, c, name, density, friction, ps.Restitution, contentPreset.Hull, contentPreset.Model, true)
			} else {
				e = g.addBoxEntity(false, pos, rot, half, c, name, density, friction, ps.Restitution)
				e.Model = contentPreset.Model
				e.HasModel = true
			}
		} else {
			e = g.addBoxEntity(false, pos, rot, half, c, name, density, friction, ps.Restitution)
		}
	case PropSphere:
		radius := ps.Radius
		if radius <= 0.01 {
			radius = fallback.Radius
		}
		e = g.addSphereEntity(pos, rot, radius, c, name, density, friction, ps.Restitution)
	case PropCylinder:
		radius := ps.Radius
		if radius <= 0.01 {
			radius = fallback.Radius
		}
		height := ps.Height
		if height <= 0.01 {
			height = fallback.Height
		}
		e = g.addCylinderEntity(pos, rot, radius, height, c, name, density, friction, ps.Restitution)
	}
	if e == nil {
		return false
	}
	if ps.Frozen {
		g.setFrozen(e, true)
	}
	return true
}

func presetForKind(kind PropKind) ShapePreset {
	for _, p := range presets {
		if p.Kind == kind {
			return p
		}
	}
	return presets[0]
}

func normalizeQuat(q box3d.Quat) box3d.Quat {
	l := float32(math.Sqrt(float64(q.V.X*q.V.X + q.V.Y*q.V.Y + q.V.Z*q.V.Z + q.S*q.S)))
	if l < 1e-5 {
		return box3d.QuatIdentity
	}
	return box3d.Quat{V: bv(q.V.X/l, q.V.Y/l, q.V.Z/l), S: q.S / l}
}
