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

Exported mod callbacks:
- `mod_init()`
- `mod_update()` (called every frame)