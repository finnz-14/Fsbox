# Mods

Place compiled WebAssembly modules (`.wasm`) in this folder. They are loaded
automatically on startup.

Mods are written in Go and compiled as a WASI reactor:

```sh
cd examplemod
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../Mods/mymod.wasm .
```

Host API module name: `fsbox`
- `log(ptr, size uint32)`
- `toast(ptr, size uint32)`
- `spawn_box(x, y, z float64)`
- `spawn_ball(x, y, z float64)`
- `spawn_barrel(x, y, z float64)`
- `clear_props()`
- `set_gravity(g float64)`
- `prop_count() uint32`
- `get_time() float64`
- `spawn_preset(index int32, x, y, z float64)`
- `remove_prop(index int32)`
- `set_player_pos(x, y, z float64)`
- `get_player_x/y/z() float64`
- `set_fly(enabled int32)`
- `set_prop_pos(index int32, x, y, z float64)`
- `freeze_prop(index int32, frozen int32)`
- `set_prop_color(index int32, r, g, b uint32)`
- `set_player_rot(yaw, pitch float64)`
- `get_player_yaw() float64`
- `get_player_pitch() float64`

Exported mod callbacks:
- `mod_init()`
- `mod_update()` (called every frame)