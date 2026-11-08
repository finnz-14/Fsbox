# Modding Framework Implementation Plan

## Scope
Two executable formats only: Go source mods (`.go`) and WebAssembly (`.wasm`). No Lua, JS, Python, C#, Java, or native DLLs.

## Phase 1 — Core Mod Infrastructure (this session)
1. Mod manifest schema (`mod.json`) + validation (runtime field, permissions, dependencies).
2. Capability-based permission enforcement at host API boundaries.
3. Native Go mod worker process launcher (no `plugin` package; isolated process).
4. JSON-RPC bridge between engine and Go mod worker.
5. Entity API: create, remove, modify position/rotation/color/freeze.
6. Physics API: gravity, set mass/center of mass, add force/impulse/torque.
7. Player API: position, rotation, fly, get transform.
8. Example Go mod + example WASM mod.
9. Rebuild pipeline in `build.bat`.

## Phase 2 — Gameplay Systems
1. Weapon system: registration, hitscan/projectile/melee, ammo, recoil.
2. Tool system: primary/secondary/reload actions, target tracing.
3. Constraints: weld, rope, spring, hinge, ball socket, slider.
4. Input hooks: key bindings, interception.
5. UI hooks: HUD panels, spawn menu integration.
6. Rendering hooks: materials, particles, post-processing.

## Phase 3 — Multiplayer & Security
1. Server-authoritative entity replication.
2. Mod compatibility verification + version sync.
3. OS-level restrictions for Go mod workers (filesystem, network, process limits).
4. Execution time/fuel budgets for WASM.

## Deliverable Checklist
- [x] WASM mod runtime (wazero) — already working
- [x] Manifest schema + validation
- [x] Permission system
- [ ] Native Go mod worker + RPC
- [ ] Entity/physics/player host API
- [ ] Example Go mod
- [ ] Example WASM mod
- [ ] `build.bat` updated
