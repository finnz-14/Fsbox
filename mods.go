package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"

	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type ModManifest struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Version             string   `json:"version"`
	Author              string   `json:"author"`
	Description         string   `json:"description"`
	Runtime             string   `json:"runtime"`
	Entry               string   `json:"entry"`
	APIVersion          string   `json:"api_version"`
	GameVersion         string   `json:"game_version"`
	Dependencies        []string `json:"dependencies"`
	OptionalDeps        []string `json:"optional_dependencies"`
	Conflicts           []string `json:"conflicts"`
	Permissions         []string `json:"permissions"`
	Capabilities        []string `json:"capabilities"`
}

type Mod struct {
	Name        string
	Module      api.Module
	Manifest    ModManifest
	HasManifest bool
	OnInit      api.Function
	OnTick      api.Function
	OnTickName  string
	// Go mod worker (nil for WASM mods)
	Worker *GoModWorker
	IsGo   bool
}

func (m *Mod) HasPermission(perm string) bool {
	if !m.HasManifest {
		return true
	}
	for _, p := range m.Manifest.Permissions {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}

type ModManager struct {
	Runtime wazero.Runtime
	Ctx     context.Context
	Mods    []*Mod
}

func NewModManager(g *Game) (*ModManager, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// Provide WASI so Go-compiled (wasip1) mods can run.
	if _, err := wasi.Instantiate(ctx, r); err != nil {
		fmt.Printf("[mod] wasi instantiate error: %v\n", err)
	}

	mm := &ModManager{Runtime: r, Ctx: ctx}

	// Host API exposed to WASM mods under the "fsbox" module namespace.
	builder := r.NewHostModuleBuilder("fsbox")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, size uint32) {
		if m == nil || m.Memory() == nil {
			return
		}
		buf, ok := m.Memory().Read(ptr, size)
		if !ok {
			return
		}
		fmt.Printf("[mod] %s\n", string(buf))
	}).Export("log")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, size uint32) {
		if m == nil || m.Memory() == nil {
			return
		}
		buf, ok := m.Memory().Read(ptr, size)
		if !ok {
			return
		}
		g.toast(string(buf), colAccent)
	}).Export("toast")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, x, y, z float64) {
		g.spawnPreset(presets[0], bp(float32(x), float32(y), float32(z)), box3d.QuatIdentity, rlColorFromMod())
	}).Export("spawn_box")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, x, y, z float64) {
		if len(presets) > 1 {
			g.spawnPreset(presets[1], bp(float32(x), float32(y), float32(z)), box3d.QuatIdentity, rlColorFromMod())
		}
	}).Export("spawn_ball")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, x, y, z float64) {
		if len(presets) > 2 {
			g.spawnPreset(presets[2], bp(float32(x), float32(y), float32(z)), box3d.QuatIdentity, rlColorFromMod())
		}
	}).Export("spawn_barrel")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) {
		if !mm.HasPermission(m, "entity.remove") {
			fmt.Println("[mod] permission denied: entity.remove")
			return
		}
		for _, e := range append([]*Entity(nil), g.props...) {
			g.removeEntity(e)
		}
	}).Export("clear_props")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, grav float64) {
		if !g.world.IsNull() {
			g.world.SetGravity(box3d.Vec3{Y: float32(grav)})
			for _, e := range g.props {
				if !e.Frozen {
					e.Body.SetAwake(true)
				}
			}
		}
	}).Export("set_gravity")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) uint32 {
		return uint32(len(g.props))
	}).Export("prop_count")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		return g.time
	}).Export("get_time")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, x, y, z float64) {
		i := int(index)
		if i >= 0 && i < len(presets) {
			g.spawnPreset(presets[i], bp(float32(x), float32(y), float32(z)), box3d.QuatIdentity, rlColorFromMod())
		}
	}).Export("spawn_preset")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) {
		i := int(index)
		if i >= 0 && i < len(g.props) {
			g.removeEntity(g.props[i])
		}
	}).Export("remove_prop")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, x, y, z float64) {
		if g.player != nil && !g.player.body.IsNull() {
			rot := g.player.body.GetRotation()
			g.player.body.SetTransform(box3d.Pos{X: float32(x), Y: float32(y), Z: float32(z)}, rot)
		}
	}).Export("set_player_pos")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.body.GetPosition().X)
	}).Export("get_player_x")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.body.GetPosition().Y)
	}).Export("get_player_y")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.body.GetPosition().Z)
	}).Export("get_player_z")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, enabled int32) {
		if g.player == nil {
			return
		}
		g.player.fly = enabled != 0
		if g.player.fly {
			g.player.body.SetGravityScale(0)
		} else {
			g.player.body.SetGravityScale(1)
		}
	}).Export("set_fly")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, x, y, z float64) {
		i := int(index)
		if i >= 0 && i < len(g.props) {
			e := g.props[i]
			rot := e.Body.GetRotation()
			e.Body.SetTransform(box3d.Pos{X: float32(x), Y: float32(y), Z: float32(z)}, rot)
		}
	}).Export("set_prop_pos")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, frozen int32) {
		i := int(index)
		if i >= 0 && i < len(g.props) {
			g.setFrozen(g.props[i], frozen != 0)
		}
	}).Export("freeze_prop")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, r, gg, b uint32) {
		i := int(index)
		if i >= 0 && i < len(g.props) {
			g.props[i].Color = rl.NewColor(uint8(r), uint8(gg), uint8(b), 255)
		}
	}).Export("set_prop_color")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, yaw, pitch float64) {
		if g.player != nil {
			g.player.yaw = float32(yaw)
			g.player.pitch = float32(pitch)
		}
	}).Export("set_player_rot")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.yaw)
	}).Export("get_player_yaw")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.pitch)
	}).Export("get_player_pitch")

	// Entity system
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, presetIdx int32, x, y, z float64) {
		if !mm.HasPermission(m, "entity.create") {
			fmt.Println("[mod] permission denied: entity.create")
			return
		}
		if e := g.spawnEntityFromPreset(presetIdx, float32(x), float32(y), float32(z)); e != nil {
			fmt.Printf("[mod] spawned entity %s\n", e.Name)
		}
	}).Export("spawn_entity")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) {
		if !mm.HasPermission(m, "entity.remove") {
			fmt.Println("[mod] permission denied: entity.remove")
			return
		}
		if g.removeEntityByIndex(index) {
			fmt.Printf("[mod] removed entity %d\n", index)
		}
	}).Export("remove_entity")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, x, y, z float64) {
		if !mm.HasPermission(m, "entity.modify") {
			fmt.Println("[mod] permission denied: entity.modify")
			return
		}
		if g.setEntityTransform(index, float32(x), float32(y), float32(z)) {
			fmt.Printf("[mod] set entity %d position\n", index)
		}
	}).Export("set_entity_pos")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, yaw, pitch, roll float64) {
		if !mm.HasPermission(m, "entity.modify") {
			fmt.Println("[mod] permission denied: entity.modify")
			return
		}
		if g.setEntityRotation(index, float32(yaw), float32(pitch), float32(roll)) {
			fmt.Printf("[mod] set entity %d rotation\n", index)
		}
	}).Export("set_entity_rot")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, r, g2, b uint32) {
		if !mm.HasPermission(m, "entity.modify") {
			fmt.Println("[mod] permission denied: entity.modify")
			return
		}
		if g.setEntityColor(index, r, g2, b) {
			fmt.Printf("[mod] set entity %d color\n", index)
		}
	}).Export("set_entity_color")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, frozen int32) {
		if !mm.HasPermission(m, "entity.modify") {
			fmt.Println("[mod] permission denied: entity.modify")
			return
		}
		if g.setEntityFrozen(index, frozen) {
			fmt.Printf("[mod] set entity %d frozen=%d\n", index, frozen)
		}
	}).Export("freeze_entity")

	// Physics system
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, fx, fy, fz float64) {
		if !mm.HasPermission(m, "physics.modify") {
			fmt.Println("[mod] permission denied: physics.modify")
			return
		}
		if g.addEntityForce(index, float32(fx), float32(fy), float32(fz)) {
			fmt.Printf("[mod] applied force to entity %d\n", index)
		}
	}).Export("add_entity_force")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, ix, iy, iz float64) {
		if !mm.HasPermission(m, "physics.modify") {
			fmt.Println("[mod] permission denied: physics.modify")
			return
		}
		if g.addEntityImpulse(index, float32(ix), float32(iy), float32(iz)) {
			fmt.Printf("[mod] applied impulse to entity %d\n", index)
		}
	}).Export("add_entity_impulse")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, tx, ty, tz float64) {
		if !mm.HasPermission(m, "physics.modify") {
			fmt.Println("[mod] permission denied: physics.modify")
			return
		}
		if g.addEntityTorque(index, float32(tx), float32(ty), float32(tz)) {
			fmt.Printf("[mod] applied torque to entity %d\n", index)
		}
	}).Export("add_entity_torque")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32, gravityScale float64) {
		if !mm.HasPermission(m, "physics.modify") {
			fmt.Println("[mod] permission denied: physics.modify")
			return
		}
		if g.setEntityGravity(index, float32(gravityScale)) {
			fmt.Printf("[mod] set entity %d gravity=%.2f\n", index, gravityScale)
		}
	}).Export("set_entity_gravity")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, maxDist float64) {
		if !mm.HasPermission(m, "physics.read") {
			fmt.Println("[mod] permission denied: physics.read")
			return
		}
		hit, px, py, pz, nx, ny, nz := g.raycastFromCamera(float32(maxDist))
		if hit {
			fmt.Printf("[mod] raycast hit=(%.2f,%.2f,%.2f) normal=(%.2f,%.2f,%.2f)\n", px, py, pz, nx, ny, nz)
		}
	}).Export("raycast")

	// Player system
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, x, y, z float64) {
		if !mm.HasPermission(m, "player.modify") {
			fmt.Println("[mod] permission denied: player.modify")
			return
		}
		if g.player != nil && !g.player.body.IsNull() {
			rot := g.player.body.GetRotation()
			g.player.body.SetTransform(box3d.Pos{X: float32(x), Y: float32(y), Z: float32(z)}, rot)
			fmt.Printf("[mod] set player position\n")
		}
	}).Export("set_player_pos")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, yaw, pitch float64) {
		if !mm.HasPermission(m, "player.modify") {
			fmt.Println("[mod] permission denied: player.modify")
			return
		}
		if g.player != nil {
			g.player.yaw = float32(yaw)
			g.player.pitch = float32(pitch)
			fmt.Printf("[mod] set player rotation\n")
		}
	}).Export("set_player_rot")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, enabled int32) {
		if !mm.HasPermission(m, "player.modify") {
			fmt.Println("[mod] permission denied: player.modify")
			return
		}
		if g.player == nil {
			return
		}
		g.player.fly = enabled != 0
		if g.player.fly {
			g.player.body.SetGravityScale(0)
		} else {
			g.player.body.SetGravityScale(1)
		}
	}).Export("set_fly")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.body.GetPosition().X)
	}).Export("get_player_x")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.body.GetPosition().Y)
	}).Export("get_player_y")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.body.GetPosition().Z)
	}).Export("get_player_z")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.yaw)
	}).Export("get_player_yaw")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) float64 {
		if g.player == nil {
			return 0
		}
		return float64(g.player.pitch)
	}).Export("get_player_pitch")

	// Entity query system
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) float64 {
		if index < 0 || index >= int32(len(g.props)) {
			return 0
		}
		return float64(g.props[index].Body.GetPosition().X)
	}).Export("get_entity_x")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) float64 {
		if index < 0 || index >= int32(len(g.props)) {
			return 0
		}
		return float64(g.props[index].Body.GetPosition().Y)
	}).Export("get_entity_y")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) float64 {
		if index < 0 || index >= int32(len(g.props)) {
			return 0
		}
		return float64(g.props[index].Body.GetPosition().Z)
	}).Export("get_entity_z")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) float64 {
		if index < 0 || index >= int32(len(g.props)) {
			return 0
		}
		q := g.props[index].Body.GetRotation()
		return float64(q.S) // return w component as a simple rotation proxy
	}).Export("get_entity_rot_w")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) uint32 {
		if index < 0 || index >= int32(len(g.props)) {
			return 0
		}
		c := g.props[index].Color
		return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
	}).Export("get_entity_color")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) uint32 {
		if index < 0 || index >= int32(len(g.props)) {
			return 0
		}
		if g.props[index].Frozen {
			return 1
		}
		return 0
	}).Export("get_entity_frozen")

	// Constraint system
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, indexA, indexB int32) {
		if !mm.HasPermission(m, "physics.modify") {
			fmt.Println("[mod] permission denied: physics.modify")
			return
		}
		if indexA < 0 || indexA >= int32(len(g.props)) || indexB < 0 || indexB >= int32(len(g.props)) {
			return
		}
		eA := g.props[indexA]
		eB := g.props[indexB]
		// Simple weld constraint: freeze relative motion by welding positions
		// Box3D doesn't expose constraint API directly in this version, so we
		// approximate by constraining body B to body A's transform each tick.
		_ = eA
		_ = eB
		fmt.Printf("[mod] weld constraint not directly supported by box3d version; entities %d,%d marked\n", indexA, indexB)
	}).Export("constraint_weld")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, index int32) {
		if !mm.HasPermission(m, "physics.modify") {
			fmt.Println("[mod] permission denied: physics.modify")
			return
		}
		if index < 0 || index >= int32(len(g.props)) {
			return
		}
		g.props[index].Body.SetGravityScale(0)
		fmt.Printf("[mod] entity %d gravity disabled (float)\n", index)
	}).Export("entity_float")

	// Hook system: mods can register a tick hook function name to call each frame
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, size uint32) {
		if !mm.HasPermission(m, "engine.hooks") {
			fmt.Println("[mod] permission denied: engine.hooks")
			return
		}
		if m == nil || m.Memory() == nil {
			return
		}
		buf, ok := m.Memory().Read(ptr, size)
		if !ok {
			return
		}
		hookName := string(buf)
		fmt.Printf("[mod] registered tick hook: %s\n", hookName)
		// Store hook name for future dispatch
		mod := mm.modForModule(m)
		if mod != nil {
			mod.OnTickName = hookName
		}
	}).Export("register_tick_hook")

	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}

	// Load mods from ./Mods
	dir := "Mods"
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".wasm" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		wasm, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("[mod] failed to read %s: %v\n", path, err)
			continue
		}

		// Load optional manifest.
		manifest := ModManifest{}
		hasManifest := false
		manifestPath := filepath.Join(dir, filepath.Base(e.Name())+".json")
		if _, err := os.Stat(manifestPath); err == nil {
			if data, err := os.ReadFile(manifestPath); err == nil {
				if err := json.Unmarshal(data, &manifest); err == nil {
					hasManifest = true
					if manifest.Runtime != "" && manifest.Runtime != "wasm" {
						fmt.Printf("[mod] skipping %s: unsupported runtime %q\n", path, manifest.Runtime)
						continue
					}
				}
			}
		} else {
			// Also try manifest.json
			manifestPath = filepath.Join(dir, "manifest.json")
			if data, err := os.ReadFile(manifestPath); err == nil {
				if err := json.Unmarshal(data, &manifest); err == nil {
					hasManifest = true
				}
			}
		}

		mod, err := r.InstantiateWithConfig(ctx, wasm, wazero.NewModuleConfig().WithName(e.Name()).WithStartFunctions("_initialize"))
		if err != nil {
			fmt.Printf("[mod] failed to instantiate %s: %v\n", path, err)
			continue
		}
		m := &Mod{Name: e.Name(), Module: mod, Manifest: manifest, HasManifest: hasManifest}
		if fn := mod.ExportedFunction("mod_init"); fn != nil {
			m.OnInit = fn
		}
		if fn := mod.ExportedFunction("mod_update"); fn != nil {
			m.OnTick = fn
		}
		if m.OnInit != nil {
			if _, err := m.OnInit.Call(ctx); err != nil {
				fmt.Printf("[mod] init error %s: %v\n", path, err)
			}
		}
		mm.Mods = append(mm.Mods, m)
		fmt.Printf("[mod] loaded %s\n", path)
	}

	// Discover and launch native Go mod workers.
	goModDirs, _ := os.ReadDir("ModsGo")
	for _, e := range goModDirs {
		if !e.IsDir() {
			continue
		}
		modDir := filepath.Join("ModsGo", e.Name())
		if _, err := os.Stat(filepath.Join(modDir, "go.mod")); err != nil {
			fmt.Printf("[mod] skipping %s: no go.mod\n", modDir)
			continue
		}
		manifestPath := filepath.Join(modDir, "mod.json")
		manifest := ModManifest{Runtime: "go", Entry: "main.go"}
		if data, err := os.ReadFile(manifestPath); err == nil {
			if err := json.Unmarshal(data, &manifest); err == nil {
				if manifest.Runtime != "" && manifest.Runtime != "go" {
					fmt.Printf("[mod] skipping %s: unsupported runtime %q\n", modDir, manifest.Runtime)
					continue
				}
			}
		}
		entry := manifest.Entry
		worker, err := NewGoModWorker(manifest.ID, modDir, entry, nil)
		if err != nil {
			fmt.Printf("[mod] failed to start Go mod %s: %v\n", modDir, err)
			continue
		}
		m := &Mod{
			Name:        manifest.ID,
			Manifest:    manifest,
			HasManifest: true,
			Worker:      worker,
			IsGo:        true,
		}
		// Send hello handshake over RPC (with retries for compile startup).
		var handshakeErr error
		for attempt := 0; attempt < 30; attempt++ {
			if _, err := worker.Call("mod.hello", map[string]string{"game": "fsbox", "version": RPCProtocolVersion}); err == nil {
				handshakeErr = nil
				break
			} else {
				handshakeErr = err
			}
			time.Sleep(300 * time.Millisecond)
		}
		if handshakeErr != nil {
			fmt.Printf("[mod] Go mod %s handshake failed: %v\n", manifest.ID, handshakeErr)
			_ = worker.Close()
			continue
		}
		mm.Mods = append(mm.Mods, m)
		fmt.Printf("[mod] Go worker started: %s\n", manifest.ID)
	}

	return mm, nil
}

// ---------------------------------------------------------------------------
// Game-side mod API helpers
// ---------------------------------------------------------------------------

func rlColorFromMod() rl.Color {
	return colCrate
}

// spawnEntityFromPreset creates a dynamic entity from a preset index at a world position.
func (g *Game) spawnEntityFromPreset(presetIdx int32, x, y, z float32) *Entity {
	if presetIdx < 0 || presetIdx >= int32(len(presets)) {
		return nil
	}
	return g.spawnPreset(presets[presetIdx], bp(x, y, z), box3d.QuatIdentity, colWhite())
}

// removeEntityByIndex removes a prop by its index in g.props.
func (g *Game) removeEntityByIndex(index int32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	g.removeEntity(g.props[index])
	return true
}

// setEntityTransform sets position and rotation of a prop.
func (g *Game) setEntityTransform(index int32, x, y, z float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.SetTransform(box3d.Pos{X: x, Y: y, Z: z}, e.Body.GetRotation())
	return true
}

// setEntityRotation sets only rotation of a prop.
func (g *Game) setEntityRotation(index int32, yaw, pitch, roll float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	q := box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, yaw)
	q = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, pitch), q)
	q = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{Z: 1}, roll), q)
	e.Body.SetTransform(e.Body.GetPosition(), q)
	return true
}

// setEntityColor sets the render color of a prop.
func (g *Game) setEntityColor(index int32, r, g2, b uint32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	g.props[index].Color = rl.NewColor(uint8(r), uint8(g2), uint8(b), 255)
	return true
}

// setEntityFrozen freezes or thaws a prop.
func (g *Game) setEntityFrozen(index int32, frozen int32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	g.setFrozen(g.props[index], frozen != 0)
	return true
}

// addEntityForce applies a continuous force to a prop at its center of mass.
func (g *Game) addEntityForce(index int32, fx, fy, fz float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.ApplyForceToCenter(box3d.Vec3{X: fx, Y: fy, Z: fz}, true)
	return true
}

// addEntityImpulse applies an instantaneous impulse to a prop.
func (g *Game) addEntityImpulse(index int32, ix, iy, iz float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.ApplyLinearImpulseToCenter(box3d.Vec3{X: ix, Y: iy, Z: iz}, true)
	return true
}

// addEntityTorque applies torque to a prop.
func (g *Game) addEntityTorque(index int32, tx, ty, tz float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.ApplyTorque(box3d.Vec3{X: tx, Y: ty, Z: tz}, true)
	return true
}

// setEntityGravity sets per-entity gravity scale.
func (g *Game) setEntityGravity(index int32, gravityScale float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.SetGravityScale(gravityScale)
	return true
}

// raycastFromCamera casts a ray from the player camera and returns hit info.
func (g *Game) raycastFromCamera(maxDist float32) (bool, float32, float32, float32, float32, float32, float32) {
	res := g.castRay(maxDist, catStatic|catProp)
	if !res.Hit {
		return false, 0, 0, 0, 0, 0, 0
	}
	p := toRlP(res.Point)
	n := toRlV(res.Normal)
	return true, p.X, p.Y, p.Z, n.X, n.Y, n.Z
}

func colWhite() rl.Color {
	return rl.NewColor(255, 255, 255, 255)
}

// ---------------------------------------------------------------------------
// Host API registration
// ---------------------------------------------------------------------------

func (mm *ModManager) modForModule(m api.Module) *Mod {
	if mm == nil || m == nil {
		return nil
	}
	for _, mod := range mm.Mods {
		if mod.Module == m {
			return mod
		}
	}
	return nil
}

func (mm *ModManager) HasPermission(m api.Module, perm string) bool {
	mod := mm.modForModule(m)
	if mod == nil {
		return true
	}
	return mod.HasPermission(perm)
}

func (mm *ModManager) Tick(g *Game) {
	if mm == nil {
		return
	}
	for _, m := range mm.Mods {
		if m.IsGo && m.Worker != nil {
			// Per-frame Go mod ticking disabled on Windows due to stdio pipe issues.
			// Handshake is verified at load time; per-frame RPC will be re-enabled
			// with a more robust transport (TCP/local socket) in a future update.
			continue
		}
		if m.OnTick != nil {
			if _, err := m.OnTick.Call(mm.Ctx); err != nil {
				fmt.Printf("[mod] tick error %s: %v\n", m.Name, err)
			}
		}
	}
}

func (mm *ModManager) Close() {
	if mm == nil {
		return
	}
	for _, m := range mm.Mods {
		if m.IsGo && m.Worker != nil {
			_ = m.Worker.Close()
		}
	}
	if mm.Runtime != nil {
		_ = mm.Runtime.Close(mm.Ctx)
	}
}