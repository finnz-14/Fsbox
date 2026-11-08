package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"

	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Mod struct {
	Name    string
	Module  api.Module
	OnInit  api.Function
	OnTick  api.Function
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

	// Host API exposed to WASM mods under the "fsbox" module namespace.
	builder := r.NewHostModuleBuilder("fsbox")
	mm := &ModManager{Runtime: r, Ctx: ctx}

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
		mod, err := r.InstantiateWithConfig(ctx, wasm, wazero.NewModuleConfig().WithName(e.Name()).WithStartFunctions("_initialize"))
		if err != nil {
			fmt.Printf("[mod] failed to instantiate %s: %v\n", path, err)
			continue
		}
		m := &Mod{Name: e.Name(), Module: mod}
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

	return mm, nil
}

func rlColorFromMod() rl.Color {
	return colCrate
}

func (mm *ModManager) Tick(g *Game) {
	if mm == nil {
		return
	}
	for _, m := range mm.Mods {
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
	if mm.Runtime != nil {
		_ = mm.Runtime.Close(mm.Ctx)
	}
}