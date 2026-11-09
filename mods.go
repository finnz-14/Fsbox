package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Name           string
	Module         api.Module
	Manifest       ModManifest
	HasManifest    bool
	OnInit         api.Function
	OnTick         api.Function
	OnTickName     string
	Worker         *GoModWorker
	IsGo           bool

	// WASM optional exports
	ModEvent       api.Function
	ModAlloc       api.Function
	ModScratch     api.Function

	// Per-mod event queue for mods that cannot receive push callbacks.
	PendingEvents [][]byte

	// Hook registrations owned by this mod: event -> hook names.
	Hooks map[string][]string
}

func (m *Mod) HasPermission(perm string) bool {
	if !m.HasManifest {
		return true
	}
	for _, p := range m.Manifest.Permissions {
		if p == "*" || p == perm {
			return true
		}
		if strings.HasSuffix(p, ".*") && strings.HasPrefix(perm, strings.TrimSuffix(p, ".*")+".") {
			return true
		}
		if p != "" && !strings.Contains(p, ".") && strings.HasPrefix(perm, p+".") {
			return true
		}
	}
	return false
}

type ModManager struct {
	Runtime     wazero.Runtime
	Ctx         context.Context
	Mods        []*Mod
	workerCalls chan workerCall
	Hooks       map[string][]hookReg
}

type workerCall struct {
	owner string
	cmd   string
	args  json.RawMessage
	resp  chan json.RawMessage
}

type hookReg struct {
	owner string
	hook  string
}

func NewModManager(g *Game) (*ModManager, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	if _, err := wasi.Instantiate(ctx, r); err != nil {
		fmt.Printf("[mod] wasi instantiate error: %v\n", err)
	}

	mm := &ModManager{
		Runtime:     r,
		Ctx:         ctx,
		workerCalls: make(chan workerCall, 256),
		Hooks:       make(map[string][]hookReg),
	}

	builder := r.NewHostModuleBuilder("fsbox")

	// Generic JSON command ABI. This is the primary, versioned entry point.
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, reqPtr, reqLen, respPtr, respCap uint32) int32 {
		if m == nil || m.Memory() == nil {
			return 0
		}
		buf, ok := m.Memory().Read(reqPtr, reqLen)
		if !ok {
			return 0
		}
		var req struct {
			Cmd  string          `json:"cmd"`
			Args json.RawMessage `json:"args"`
		}
		if err := json.Unmarshal(buf, &req); err != nil {
			resp := map[string]any{"ok": false, "error": "invalid request: " + err.Error()}
			return writeGuestResponse(m, respPtr, respCap, resp)
		}
		mod := mm.modForModule(m)
		owner := ""
		if mod != nil {
			owner = mod.Name
			if mod.Manifest.ID != "" {
				owner = mod.Manifest.ID
			}
		}
		if mod != nil && !mod.HasPermission(permissionForCommand(req.Cmd)) {
			resp := map[string]any{"ok": false, "error": "permission denied: " + permissionForCommand(req.Cmd)}
			return writeGuestResponse(m, respPtr, respCap, resp)
		}
		result, err := g.ModCall(owner, req.Cmd, req.Args)
		resp := map[string]any{"ok": true, "result": result}
		if err != nil {
			resp = map[string]any{"ok": false, "error": err.Error()}
		}
		return writeGuestResponse(m, respPtr, respCap, resp)
	}).Export("fsbox_call")

	// Event polling for mods without mod_event export.
	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, cap uint32) int32 {
		if m == nil || m.Memory() == nil {
			return 0
		}
		mod := mm.modForModule(m)
		if mod == nil || len(mod.PendingEvents) == 0 {
			return 0
		}
		ev := mod.PendingEvents[0]
		mod.PendingEvents = mod.PendingEvents[1:]
		if uint32(len(ev)) > cap {
			// put back and report required size
			mod.PendingEvents = append([][]byte{ev}, mod.PendingEvents...)
			return int32(len(ev))
		}
		if !m.Memory().Write(ptr, ev) {
			return 0
		}
		return int32(len(ev))
	}).Export("fsbox_event_poll")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module) int32 {
		return 2
	}).Export("fsbox_api_version")

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, size uint32) int32 {
		if m == nil || m.Memory() == nil {
			return 1
		}
		buf, ok := m.Memory().Read(ptr, size)
		if !ok {
			return 1
		}
		perm := string(buf)
		mod := mm.modForModule(m)
		if mod == nil {
			return 1
		}
		if mod.HasPermission(perm) {
			return 1
		}
		return 0
	}).Export("fsbox_has_permission")

	// Backwards-compatible scalar helpers (non-duplicated).
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

	builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, m api.Module, ptr, size uint32) {
		if m == nil || m.Memory() == nil {
			return
		}
		buf, ok := m.Memory().Read(ptr, size)
		if !ok {
			return
		}
		g.toast(string(buf), colAccent)
	}).Export("notify")

	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}

	// Load WASM mods.
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

		manifest := ModManifest{}
		hasManifest := false
		manifestPath := filepath.Join(dir, e.Name()+".json")
		if _, err := os.Stat(manifestPath); err == nil {
			if data, err := os.ReadFile(manifestPath); err == nil {
				if err := json.Unmarshal(data, &manifest); err == nil {
					hasManifest = true
				}
			}
		} else {
			manifestPath = filepath.Join(dir, "manifest.json")
			if data, err := os.ReadFile(manifestPath); err == nil {
				if err := json.Unmarshal(data, &manifest); err == nil {
					hasManifest = true
				}
			}
		}
		if hasManifest && manifest.Runtime != "" && manifest.Runtime != "wasm" {
			fmt.Printf("[mod] skipping %s: unsupported runtime %q\n", path, manifest.Runtime)
			continue
		}
		if manifest.Runtime == "" {
			manifest.Runtime = "wasm"
		}

		mod, err := r.InstantiateWithConfig(ctx, wasm, wazero.NewModuleConfig().WithName(e.Name()).WithStartFunctions("_initialize"))
		if err != nil {
			fmt.Printf("[mod] failed to instantiate %s: %v\n", path, err)
			continue
		}
		m := &Mod{Name: e.Name(), Module: mod, Manifest: manifest, HasManifest: hasManifest, Hooks: make(map[string][]string)}
		if fn := mod.ExportedFunction("mod_init"); fn != nil {
			m.OnInit = fn
		}
		if fn := mod.ExportedFunction("mod_update"); fn != nil {
			m.OnTick = fn
		}
		if fn := mod.ExportedFunction("mod_event"); fn != nil {
			m.ModEvent = fn
		}
		if fn := mod.ExportedFunction("mod_alloc"); fn != nil {
			m.ModAlloc = fn
		}
		if fn := mod.ExportedFunction("mod_scratch"); fn != nil {
			m.ModScratch = fn
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
		if manifest.ID == "" {
			manifest.ID = e.Name()
		}
		if manifest.Runtime == "" {
			manifest.Runtime = "go"
		}
		if manifest.Entry == "" {
			manifest.Entry = "main.go"
		}

		worker, err := NewGoModWorker(manifest.ID, modDir, manifest.Entry, nil)
		if err != nil {
			fmt.Printf("[mod] failed to start Go mod %s: %v\n", modDir, err)
			continue
		}
		worker.OnRequest = func(method string, params json.RawMessage) (any, error) {
			if method != "engine.call" {
				return nil, fmt.Errorf("unknown method %q", method)
			}
			var a struct {
				Cmd  string          `json:"cmd"`
				Args json.RawMessage `json:"args"`
			}
			if err := json.Unmarshal(params, &a); err != nil {
				return nil, err
			}
			ch := make(chan json.RawMessage, 1)
			mm.workerCalls <- workerCall{owner: manifest.ID, cmd: a.Cmd, args: a.Args, resp: ch}
			select {
			case raw := <-ch:
				return json.RawMessage(raw), nil
			case <-time.After(10 * time.Second):
				return nil, fmt.Errorf("engine call timeout")
			}
		}

		m := &Mod{
			Name:        manifest.ID,
			Manifest:    manifest,
			HasManifest: true,
			Worker:      worker,
			IsGo:        true,
			Hooks:       make(map[string][]string),
		}

		params := map[string]any{
			"game":        "fsbox",
			"version":     RPCProtocolVersion,
			"api_version": ModAPIVersion,
		}
		resCh := make(chan struct {
			msg *RPCMessage
			err error
		}, 1)
		go func() {
			msg, err := worker.Call("mod.init", params)
			resCh <- struct {
				msg *RPCMessage
				err error
			}{msg, err}
		}()

		start := time.Now()
		handshakeErr := error(nil)
		done := false
		for !done && time.Since(start) < 10*time.Second {
			select {
			case r := <-resCh:
				handshakeErr = r.err
				done = true
			case req := <-mm.workerCalls:
				result, err := g.ModCall(req.owner, req.cmd, req.args)
				resp := map[string]any{"ok": true, "result": result}
				if err != nil {
					resp = map[string]any{"ok": false, "error": err.Error()}
				}
				raw, _ := json.Marshal(resp)
				req.resp <- raw
			default:
				time.Sleep(10 * time.Millisecond)
			}
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

func writeGuestResponse(m api.Module, respPtr, respCap uint32, resp map[string]any) int32 {
	data, err := json.Marshal(resp)
	if err != nil {
		return 0
	}
	if respCap == 0 {
		return int32(len(data))
	}
	n := len(data)
	if n > int(respCap) {
		n = int(respCap)
	}
	if n > 0 {
		if ok := m.Memory().Write(respPtr, data[:n]); !ok {
			return 0
		}
	}
	return int32(len(data))
}

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

func (mm *ModManager) findModByName(name string) *Mod {
	if mm == nil {
		return nil
	}
	for _, m := range mm.Mods {
		if m.Name == name || (m.Manifest.ID != "" && m.Manifest.ID == name) {
			return m
		}
	}
	return nil
}

func (mm *ModManager) OwnerHasPermission(owner, perm string) bool {
	m := mm.findModByName(owner)
	if m == nil {
		return true
	}
	return m.HasPermission(perm)
}

func (mm *ModManager) AddHook(owner, event, hook string) bool {
	if event == "" || hook == "" {
		return false
	}
	m := mm.findModByName(owner)
	if m == nil {
		return false
	}
	if !m.HasPermission("engine.hooks") {
		return false
	}
	if m.Hooks == nil {
		m.Hooks = make(map[string][]string)
	}
	m.Hooks[event] = appendUnique(m.Hooks[event], hook)
	mm.Hooks[event] = append(mm.Hooks[event], hookReg{owner: owner, hook: hook})
	return true
}

func (mm *ModManager) RemoveHook(owner, event, hook string) bool {
	m := mm.findModByName(owner)
	if m != nil && m.Hooks != nil {
		m.Hooks[event] = removeString(m.Hooks[event], hook)
	}
	regs := mm.Hooks[event]
	out := regs[:0]
	for _, r := range regs {
		if r.owner == owner && r.hook == hook {
			continue
		}
		out = append(out, r)
	}
	mm.Hooks[event] = out
	return true
}

func (mm *ModManager) ListHooks(owner string) map[string][]string {
	out := make(map[string][]string)
	m := mm.findModByName(owner)
	if m != nil && m.Hooks != nil {
		for k, v := range m.Hooks {
			out[k] = append([]string(nil), v...)
		}
	}
	return out
}

func (mm *ModManager) deliverWASMEvent(m *Mod, raw []byte) {
	if m == nil || m.Module == nil {
		return
	}
	if m.ModEvent != nil {
		ptr := uint32(0)
		if m.ModAlloc != nil {
			res, err := m.ModAlloc.Call(mm.Ctx, uint64(len(raw)))
			if err == nil && len(res) > 0 {
				ptr = uint32(res[0])
			}
		} else if m.ModScratch != nil {
			res, err := m.ModScratch.Call(mm.Ctx, uint64(len(raw)))
			if err == nil && len(res) > 0 {
				ptr = uint32(res[0])
			}
		}
		if ptr != 0 {
			if ok := m.Module.Memory().Write(ptr, raw); ok {
				if _, err := m.ModEvent.Call(mm.Ctx, uint64(ptr), uint64(len(raw))); err != nil {
					fmt.Printf("[mod] event error %s: %v\n", m.Name, err)
				}
				return
			}
		}
	}
	// Queue for polling.
	m.PendingEvents = append(m.PendingEvents, raw)
}

func (mm *ModManager) FireEvent(event string, data any) {
	if mm == nil {
		return
	}
	payload := map[string]any{"event": event, "data": data}
	raw, _ := json.Marshal(payload)
	for _, m := range mm.Mods {
		if m.IsGo && m.Worker != nil {
			m.Worker.Notify("mod.event", payload)
			continue
		}
		mm.deliverWASMEvent(m, raw)
	}
	// Registered hooks: invoke specific exported functions.
	if regs, ok := mm.Hooks[event]; ok {
		for _, r := range regs {
			m := mm.findModByName(r.owner)
			if m == nil {
				continue
			}
			if m.IsGo && m.Worker != nil {
				m.Worker.Notify("mod.hook", map[string]any{"event": event, "hook": r.hook, "data": data})
				continue
			}
			if m.Module != nil {
				fn := m.Module.ExportedFunction(r.hook)
				if fn == nil {
					continue
				}
				if _, err := fn.Call(mm.Ctx); err != nil {
					fmt.Printf("[mod] hook error %s/%s: %v\n", m.Name, r.hook, err)
				}
			}
		}
	}
}

func (mm *ModManager) Tick(g *Game, simDt float64) {
	if mm == nil {
		return
	}
	mm.DrainWorkerCalls(g)
	for _, m := range mm.Mods {
		if m.IsGo && m.Worker != nil {
			m.Worker.Notify("mod.tick", map[string]any{"dt": simDt, "time": g.time})
			continue
		}
		if m.OnTick != nil {
			if _, err := m.OnTick.Call(mm.Ctx); err != nil {
				fmt.Printf("[mod] tick error %s: %v\n", m.Name, err)
			}
		}
	}
	mm.DrainWorkerCalls(g)
}

func (mm *ModManager) DrainWorkerCalls(g *Game) {
	for {
		select {
		case req := <-mm.workerCalls:
			result, err := g.ModCall(req.owner, req.cmd, req.args)
			resp := map[string]any{"ok": true, "result": result}
			if err != nil {
				resp = map[string]any{"ok": false, "error": err.Error()}
			}
			raw, _ := json.Marshal(resp)
			req.resp <- raw
		default:
			return
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

// Permission mapping for commands.
func permissionForCommand(cmd string) string {
	switch {
	case strings.HasPrefix(cmd, "engine."):
		if cmd == "engine.info" || cmd == "engine.get_time" || cmd == "engine.api_version" || cmd == "engine.has_permission" {
			return "engine.read"
		}
		return "engine.control"
	case strings.HasPrefix(cmd, "entity."):
		if strings.HasPrefix(cmd, "entity.get") || strings.HasPrefix(cmd, "entity.list") || strings.HasPrefix(cmd, "entity.exists") {
			return "entity.read"
		}
		if strings.HasPrefix(cmd, "entity.spawn") || cmd == "entity.clear" {
			return "entity.create"
		}
		if strings.HasPrefix(cmd, "entity.remove") {
			return "entity.remove"
		}
		return "entity.modify"
	case strings.HasPrefix(cmd, "physics."):
		if cmd == "physics.stats" || cmd == "physics.get_gravity" || cmd == "physics.raycast" || cmd == "physics.raycast_camera" || cmd == "physics.overlap_sphere" {
			return "physics.read"
		}
		return "physics.modify"
	case strings.HasPrefix(cmd, "player."):
		if cmd == "player.get" {
			return "player.read"
		}
		return "player.modify"
	case strings.HasPrefix(cmd, "world."):
		if cmd == "world.get_time" || cmd == "world.get_sun" {
			return "world.read"
		}
		return "world.modify"
	case strings.HasPrefix(cmd, "input."):
		return "input.read"
	case strings.HasPrefix(cmd, "ui."):
		return "ui.draw"
	case strings.HasPrefix(cmd, "render."):
		return "render.draw"
	case strings.HasPrefix(cmd, "particle."):
		return "particle"
	case strings.HasPrefix(cmd, "audio."):
		return "audio"
	case strings.HasPrefix(cmd, "weapon."):
		return "weapon"
	case strings.HasPrefix(cmd, "tool."):
		return "tool"
	case strings.HasPrefix(cmd, "hook."):
		return "engine.hooks"
	case strings.HasPrefix(cmd, "resource."):
		return "resource"
	case strings.HasPrefix(cmd, "content."):
		return "content"
	case strings.HasPrefix(cmd, "settings."):
		return "settings"
	case strings.HasPrefix(cmd, "save") || strings.HasPrefix(cmd, "world.save") || strings.HasPrefix(cmd, "world.load") || strings.HasPrefix(cmd, "engine.save") || strings.HasPrefix(cmd, "engine.load"):
		return "save"
	case strings.HasPrefix(cmd, "net."):
		return "net"
	case strings.HasPrefix(cmd, "camera."):
		return "camera"
	default:
		return "engine.read"
	}
}

// ---------------------------------------------------------------------------
// Game-side mod API helpers (backwards compat)
// ---------------------------------------------------------------------------

func rlColorFromMod() rl.Color {
	return colCrate
}

func (g *Game) spawnEntityFromPreset(presetIdx int32, x, y, z float32) *Entity {
	if presetIdx < 0 || presetIdx >= int32(len(presets)) {
		return nil
	}
	return g.spawnPreset(presets[presetIdx], bp(x, y, z), box3d.QuatIdentity, colWhite())
}

func (g *Game) removeEntityByIndex(index int32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	g.removeEntity(g.props[index])
	return true
}

func (g *Game) setEntityTransform(index int32, x, y, z float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.SetTransform(box3d.Pos{X: x, Y: y, Z: z}, e.Body.GetRotation())
	return true
}

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

func (g *Game) setEntityColor(index int32, r, g2, b uint32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	g.props[index].Color = rl.NewColor(uint8(r), uint8(g2), uint8(b), 255)
	return true
}

func (g *Game) setEntityFrozen(index int32, frozen int32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	g.setFrozen(g.props[index], frozen != 0)
	return true
}

func (g *Game) addEntityForce(index int32, fx, fy, fz float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.ApplyForceToCenter(box3d.Vec3{X: fx, Y: fy, Z: fz}, true)
	return true
}

func (g *Game) addEntityImpulse(index int32, ix, iy, iz float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.ApplyLinearImpulseToCenter(box3d.Vec3{X: ix, Y: iy, Z: iz}, true)
	return true
}

func (g *Game) addEntityTorque(index int32, tx, ty, tz float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.ApplyTorque(box3d.Vec3{X: tx, Y: ty, Z: tz}, true)
	return true
}

func (g *Game) setEntityGravity(index int32, gravityScale float32) bool {
	if index < 0 || index >= int32(len(g.props)) {
		return false
	}
	e := g.props[index]
	e.Body.SetGravityScale(gravityScale)
	return true
}

func (g *Game) raycastFromCamera(maxDist float32) (bool, float32, float32, float32, float32, float32, float32) {
	res := g.castRay(maxDist, catStatic|catProp)
	if !res.Hit {
		return false, 0, 0, 0, 0, 0, 0
	}
	p := toRlP(res.Point)
	n := toRlV(res.Normal)
	return true, p.X, p.Y, p.Z, n.X, n.Y, n.Z
}
