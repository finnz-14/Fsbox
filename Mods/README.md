# Mods

Place compiled WebAssembly modules (`.wasm`) in this folder. They are loaded
automatically on startup.

Mods are written in Go and compiled as a WASI reactor:

```sh
cd examplemod
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../Mods/mymod.wasm .
```

Native Go mods live in `ModsGo/<name>` and are built to `bin/worker.exe`.

## Host API
Primary entry point is the generic JSON command ABI:
- WASM: `fsbox_call(reqPtr, reqLen, respPtr, respCap) int32`
- Request: `{"cmd":"namespace.command","args":{...}}`
- Response: `{"ok":true,"result":...}` or `{"ok":false,"error":"..."}`

Events are delivered through `mod_event(ptr,len)` when `mod_alloc`/`mod_scratch` are exported, otherwise via `fsbox_event_poll`.

Legacy scalar exports (`log`, `toast`, `spawn_box`, `spawn_ball`, `spawn_barrel`, `clear_props`, `set_gravity`, `prop_count`, `get_time`, `spawn_preset`, `remove_prop`, `set_player_pos`, `get_player_x/y/z`, `set_fly`, `set_prop_pos`, `freeze_prop`, `set_prop_color`, `set_player_rot`, `get_player_yaw/pitch`, `spawn_entity`, `remove_entity`, `set_entity_pos/rot/color`, `freeze_entity`, `add_entity_force/impulse/torque`, `set_entity_gravity`, `raycast`, `constraint_weld`, `entity_float`, `register_tick_hook`) remain available.

## Permissions
Manifest `permissions` may include `*`, exact names, or prefixes (`entity.*`, `entity`). Unknown commands and missing permissions are rejected.

## SDK
See `sdk/go`. Module: `fsbox.dev/sdk`.
