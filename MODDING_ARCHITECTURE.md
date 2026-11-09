# Modding Architecture

Fsbox supports two executable mod formats only: Go source mods (`.go`, isolated worker processes) and WebAssembly (`.wasm`, wazero). No Lua, JS, Python, C#, Java, DLLs, or shared libraries.

## Runtime
- WASM mods are WASI reactors built with `GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared`. They are instantiated with `_initialize`.
- Go mods are pre-built binaries at `ModsGo/<mod>/bin/worker.exe`, launched as isolated processes. Communication is newline-delimited JSON-RPC 2.0 over stdio.
- Manifests: `Mods/<name>.wasm.json`, `Mods/manifest.json`, or `ModsGo/<mod>/mod.json`. Only `wasm` and `go` runtimes are accepted.

## Host ABI
- WASM: `fsbox_call(reqPtr, reqLen, respPtr, respCap) int32` with JSON request `{"cmd":..., "args":{...}}` and response `{"ok":bool,"result":...,"error":...}`.
- WASM events: engine calls `mod_event(ptr,len)` when the guest exports `mod_alloc`/`mod_scratch`; otherwise events are polled via `fsbox_event_poll`.
- Go worker: engine sends `mod.init`, `mod.tick`, `mod.event`, `mod.hook` notifications/calls; worker calls `engine.call` with `{"cmd":...,"args":...}`.
- Compatibility scalar exports (`log`, `toast`, `spawn_box`, etc.) are retained.

## Capability permissions
Manifest `permissions` list. Wildcards supported (`*`, `entity.*`, `entity`). Enforced at every host API boundary. No manifest means all permissions.

## Full command API
Commands are grouped by namespace. All are reachable through `fsbox_call` / `engine.call`:
- `engine.*`: info, log, toast, time scale, pause, reset, save/load, quit, screenshot, key/mouse binding, HUD hooks, api version, permission query.
- `entity.*`: spawn (preset or custom), remove, clear, list, get, exists, transform, color, name, frozen, invulnerable, health, damage, heal, static, velocity, material, bullet, wake/sleep, collision, spawn_copy.
- `physics.*`: gravity get/set, raycast (world and camera), overlap sphere, explode, stats, worker count, constraints (create/list/remove: weld, distance, rope, spring, revolute, hinge, prismatic, slider, spherical, ball, motor, filter, parallel, wheel).
- `player.*`: get, position, rotation, velocity, fly, speed/jump scales, health, damage, heal, respawn, gravity scale.
- `world.*`: sun, sky, time, time scale, pause/resume, reset, gravity, save/load.
- `input.*`: key/mouse state queries, mouse position/delta, wheel, focus, cursor lock.
- `ui.*`: toast, draw text/rect/line/circle/triangle, measure text, screen size, clear.
- `render.*`: 3D line/box/sphere/ray/circle, world/screen conversion, FOV, shadows, screen size.
- `particle.*`: spawn, burst, clear, count.
- `audio.*`: load/play/stop sound and music, volume, ready, unload.
- `weapon.*`: register, unregister, list, select, fire, ammo management.
- `tool.*`: register, unregister, list, select, primary/secondary/reload, active.
- `hook.*`: add, remove, list, fire.
- `resource.*`: sandboxed mod data files (read, write, append, exists, delete, list, data dir).
- `content.*`: list/add/remove spawnable presets.
- `settings.*`: get, set, save.
- `net.*`: local mod message broadcast.
- `camera.*`: shake, get.

## Gameplay systems
- Entity health/damage/death with events.
- Transient projectile bodies with collision damage.
- Custom weapons (hitscan, projectile, melee) with ammo and fire rate.
- Custom tools with primary/secondary/reload hooks.
- box3d constraints (all joint types).
- Mod particles and UI/debug draw queues.
- Audio playback.
- Mod data files sandboxed to `ModData/<mod>/`.

## SDK
- `sdk/go` module `fsbox.dev/sdk` provides a WASM helper (`fsbox.Call`) and exported memory/event stubs.

## Build
- `build.bat` builds WASM mod, Go worker, game, and copies to `bin\sandbox3d.exe` and `sandbox3d.exe`.
