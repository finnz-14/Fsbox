# Modding Architecture (Incremental)

## Current Status

- WebAssembly mods (`.wasm`) via wazero are supported.
- Native Go source mods (`.go`) are **not yet isolated**; only WASM mods run today.
- A basic host API is exposed under the `fsbox` module namespace.

## Implemented Host API

See `Mods/README.md`.

## Planned (Not Yet Implemented)

- Process-isolated native Go mod runtime (no `plugin` package).
- Versioned RPC protocol between engine and Go mod workers.
- Manifest validation (`mod.json`).
- Capability-based permission enforcement.
- Entity/physics/rendering/UI/networking APIs.
- Multiplayer replication.
- Mod CLI (`game mod ...`).

## Security Model

- WASM mods are sandboxed by wazero.
- Native Go mods must run in separate processes with OS-level restrictions.
- No DLL/shared-library mods.

This document tracks the incremental roadmap.