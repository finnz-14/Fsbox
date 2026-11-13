# Fsbox

A first-person 3D physics sandbox written in Go with
[raylib-go](https://github.com/gen2brain/raylib-go) for rendering and
[box3d-go](https://github.com/gataengine/box3d-go) for rigid-body physics.

![Fsbox preview](preview.png)

Fsbox is a complete desktop sandbox rather than a static demo. You walk a
dynamic capsule through a walled yard, manipulate real rigid bodies with a tool
gun, spawn props from a browser-style menu, freeze and duplicate matter, change
world gravity, and save or load the scene. The interface is a restrained
industrial console rather than a cartoon HUD: near-black panels, square corners,
one-pixel borders, amber signals and compact uppercase telemetry.

## Requirements

- Go 1.27 or newer.
- A desktop with OpenGL 3.3 support and a mouse/keyboard.
- No C compiler required. The project uses raylib-go's `purego` backend, so it
  builds and runs with `CGO_ENABLED=0` (the raylib shared library is embedded in
  the Go module).

The current build has been tested on Windows amd64 with Go 1.27.3 and
`CGO_ENABLED=0`.

## Run

From this directory:

```powershell
go run .
```

To build a standalone executable into `bin\`:

```powershell
.\build.bat
.\fsbox.exe
```

`build.bat` sets `CGO_ENABLED=0` and writes `fsbox.exe` at the root.
You can also build manually:

```powershell
go build -o fsbox.exe .
```

Run the automated tests:

```powershell
go test ./...
```

## Multiplayer

Fsbox has an authoritative dedicated server. The server and client share one
binary; the server runs headless with a hidden GL context so content and mods
still load.

Build the dedicated server into `server\`:

```powershell
server\build.bat
server\fsbox-server.exe -server -port 27015 -map default
```

Use `server\build.bat release` to exclude smoke-test code. Run the server from
the repository root so it can load `Mods\` and write its files next to the game.

Join from the game with **MULTIPLAYER** on the main menu, or connect directly:

```powershell
fsbox.exe -connect 127.0.0.1:27015 -name PLAYER
```

Flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-server` | off | Run the authoritative headless server. |
| `-port` | `27015` | TCP port for the dedicated server. |
| `-connect host[:port]` | off | Join a server at this address. |
| `-name` | `PLAYER` | Callsign shown to other players. |
| `-map` | current map | Start a map directly. |

The server simulates at 60 Hz and broadcasts snapshots at 20 Hz. Clients send
input and intents (grab, spawn, delete, tools); the server validates and applies
them. The pause menu has a **DISCONNECT** action while connected. Single-player
behaviour is unchanged.

### Multiplayer mod API

Mods run on the server. The SDK exposes `NetStatus`, `NetPlayers`,
`BroadcastToast`, `SendNetEvent` and `KickPlayer`. Server mods can broadcast
custom events to client mods, and client mods can send events back to the server.
See the **Networking** tab in `Devkit\docs.html` for signatures and examples.

## Controls

| Input | Action |
| --- | --- |
| Mouse | Look |
| W A S D | Move |
| Shift | Sprint |
| Space | Jump (walk mode) |
| V | Toggle walk / fly mode |
| Space / Ctrl | Fly up / down |
| F | Freeze / thaw the held or aimed prop |
| Left mouse | Primary action of the equipped tool |
| Right mouse | Secondary action of the equipped tool |
| E | Physgun grab / throw alias |
| 1 - 7 | Equip a tool mode directly |
| Mouse wheel | Adjust hold / spawn distance |
| Ctrl + Mouse wheel | Cycle tool modes in the field |
| C | Open / close the tool selector |
| Q or Tab | Open / close the spawn menu |
| Enter | Spawn / equip / execute the selected browser entry |
| Double click | Activate a browser card |
| F2 | Open / close the settings menu |
| G | Cycle gravity: Earth, low, zero-G, heavy |
| [ / ] | Lower / raise gravity by 0.10x |
| F5 | Save the scene |
| F9 | Load the scene |
| Ctrl + R | Reset the arena and starter props |
| Ctrl + Delete | Remove every dynamic prop |
| F1 or H | Toggle the field manual |
| Esc | Pause / resume, or close a menu |

### Tool gun modes

The tool gun is the single interaction interface. It is drawn in the lower-right
of the view and changes its accent colour and muzzle flash per mode. The HUD
shows the current wheel-controlled range.

| # | Tool | Left mouse | Right mouse |
| --- | --- | --- | --- |
| 1 | PHYSGUN | Grab or ungrab a body | Shoot (throw) the held body, or push the aimed body |
| 2 | SPAWNER | Spawn the buffered prop at the aim point | Open the spawn menu |
| 3 | REMOVER | Delete the targeted body | Clear a 3.2 m radius |
| 4 | IMPACT | Radial shockwave | Launch the target |
| 5 | FREEZE | Toggle the target body | Toggle every prop |
| 6 | DUPLICATOR | Copy with a forward offset | Copy onto the aimed surface |
| 7 | GRAVITY | Cycle gravity presets | Toggle zero-G |

The global **F** key freezes or thaws whatever you are holding or aiming at, so
anchoring matter no longer depends on a right-click or a specific tool mode.

## Spawn menu

Press **Q** or **Tab** to open the spawn interface. The world keeps simulating
while the overlay is open; only the pause menu stops it. It follows the
structure of a GMod-style browser:

- Left rail: category filter for the current tab, plus a live prop-capacity bar.
- Header tabs: **SPAWN**, **TOOLS**, and **WORLD**.
- Search field: filters props or tools by name, category, or description.
- Card grid: select an entry with a single click or browse with the wheel.
- Inspection panel: preview, mass, friction, bounce, size, or tool instructions.
- Explicit action button: **SPAWN SELECTED**, **EQUIP TOOL**, or **EXECUTE**.

Double-clicking a card or pressing Enter activates it. The browser stays open
after spawning so several props can be placed in one pass; repeated spawns are
laid out around the marker captured when the menu opened instead of stacking on
top of each other. Close the browser with **Q** or **Esc** to resume the frozen
simulation. The WORLD tab exposes save, load, reset, clear, and settings so you
rarely need a hidden hotkey.

## Content packs

Spawnable models are loaded from `Content/Spawnables` using this layout:

```text
Content/
  Spawnables/
    decorate/          category shown in the spawn menu
      crate/           one object folder per spawnable
        crate.glb      .obj, .glb or .gltf
        manifest.json  optional
```

The folder names define the category and the display name. If an object folder
contains more than one model, add a `manifest.json` to choose the main one:

```json
{
  "main": "crate.glb"
}
```

`main` may also be written as `main object`, `mainObject`, `object`, `model`, or
`file`. Optional manifest fields are `name`, `density`, `friction`,
`restitution`/`bounce`, `color` and `half`. The loader also accepts models placed
directly in a category folder, and the older loose `.json` spawnable format at
the root of `Spawnables`.

The game looks for `Content/Spawnables` in the working directory, next to the
executable, and one or two folders above it, so both `go run .` and
`fsbox.exe` find the same content. Loader diagnostics are printed to the
console with a `[content]` prefix.

## Gameplay details

- Ten spawnable shapes: **Crate**, **Ball**, **Barrel**, **Slab**, **Anvil**,
  **Block**, **Pebble**, **Boulder**, **Plank**, and **Column**. Each has its own
  size, mass, friction, restitution, and colour.
- The **spawner tool** shows a translucent placement ghost on the surface under
  the crosshair. Left-click materialises the buffered prop there. The ghost is
  hidden while carrying a body so it cannot obscure the view.
- The **physgun** pulls a grabbed prop toward a stable hold point in front of the
  camera while preserving its angular motion, so objects swing naturally. Large
  props are held further away and below the crosshair. Release to throw along the
  view direction. The mouse wheel moves the hold point closer or further away;
  right-click pushes the held or aimed body without freezing it.
- The **freeze tool** converts a dynamic body to a static one and zeroes its
  velocities. Frozen props ignore gravity and impulses and are outlined in ice
  blue. The global **F** key freezes the held or aimed body from any tool mode,
  and the state is preserved in saves.
- The **duplicator** copies a body with its exact shape, mass, material and
  rotation, offset either forward or along the aimed surface normal.
- The **remover** can dissolve one body or clear a short radius. The **impact**
  tool applies mass-scaled radial impulses with distance falloff, plus a visible
  expanding ring, particles, and camera shake.
- The **settings menu** (**F2**) has live sliders for gravity, mouse sensitivity,
  and field of view, plus toggles for inverted mouse Y, contact shadows, fullscreen,
  and raw mouse input. Mouse sensitivity is shown as a percentage and can go down
  to 10%. The interface scales with the window size. Changes apply immediately and
  are stored in `fsbox_settings.json`.
- The **gravity tool** changes world gravity from **0.00x** (zero-G) to **2.50x**
  (heavy). **G** cycles the useful presets and **[** / **]** fine-tune by 0.10x.
  The HUD always shows the current multiplier.
- The starter yard contains a 14-crate ziggurat, balls, barrels, slabs, an anvil,
  raised platforms, a ramp, steps, a storage bay, pillars, and scattered crates.
- Spheres draw two body-aligned great circles and a surface dot, so a rolling or
  spinning ball is visibly rotating instead of looking like a static disc.
- The player is a dynamic capsule with locked rotation. box3d resolves all
  collisions with the arena and props; the controller only steers velocity.
- The simulation runs at a fixed **60 Hz** step with an accumulator (up to eight
  steps per frame), while rendering runs at the monitor's rate.
- Collision categories keep queries away from the player: static geometry and
  props can be hit by aim/placement rays, but the player capsule cannot.

## Architecture

The root keeps only a thin launcher (`main.go`). All game code lives under
`internal/`, split into packages by responsibility:

```
main.go                    Thin launcher: calls internal/game.Run()
internal/game/             The game itself (one package)
  main.go                  Window setup, game loop, flags, smoke test wiring
  game.go                  Game state, fixed-step update, input routing, camera, toasts and effects
  world.go                 Physics world, collision filters, presets, arena, props, actions and queries
  content.go               Discovers Content/Spawnables, reads manifests and loads model spawnables
  content_test.go          Content discovery, descriptor and map tests
  player.go                Dynamic capsule character controller (walk, sprint, jump, fly)
  tools.go                 Tool gun definitions, tool selection, primary/secondary actions, selector and viewmodel
  render.go                Sky, arena grid, shadows, entities, sphere spin markings, outlines, ghost, beams, particles and waves
  ui.go                    Industrial HUD, crosshair, telemetry, tool status, manual, pause overlay and toasts
  menu.go                  GMod-style spawn browser, world actions and settings panel
  settings.go              Settings values, persistence, clamping and the gravity tool
  save.go                  Versioned JSON save/load of dynamic props, frozen state and player pose
  addons.go                Addon manager: discovery, load/unload, commands, errors and console logging
  addonanim.go             Addon animation playback driving skeletons
  addonapi.go              Addon command router (tool.*, net.*, render.*, ...)
  addonapi_entity.go       Addon entity/tool/weapon APIs and their tests
  addonapi_extra.go        Addon UI, billboard and particle APIs
  skeletalcollision.go     Per-bone skeletal collision proxies (spheres, capsules, hulls)
  visualdepth_entity.go    Rasterises posed models into the visual depth buffer
  visualdepth_entity_test.go  Visual depth occlusion test with a posed mesh
  loading.go               Startup loading screen with progress
  mainmenu.go              Main menu (static background) and multiplayer entry
  netapi.go                Addon-facing networking status/player APIs
  net_alias.go             Aliases into internal/netproto so call sites stay short
  net_client.go            Dedicated-server client: dial, snapshots, prediction
  net_server.go            Dedicated headless server: authoritative loop and broadcasts
  netmenu.go               Multiplayer menu with saved server list
  smoke.go / smoke_release.go  Development-only smoke mode (excluded in release builds)
  *_test.go                Headless tests for physics, tools, save/load, spawning, gravity, UI filters, networking
  Content/                 Embedded default spawnables and maps
internal/netproto/         TCP + JSON wire protocol for dedicated-server multiplayer (types, framing, name sanitize, change detection)
internal/visualdepth/      Half-resolution software depth buffer for posed skeletal models
server/                    Dedicated server packaging (server/build.bat, README)
Addons/                    Addon packages (gitignored; not committed)
Devkit/                    SDK and docs for addon authors
```

| Package | Responsibility |
| --- | --- |
| `main` | Thin launcher only; exits with the game's return code |
| `internal/game` | Everything game-side: simulation, rendering, UI, addons, networking glue |
| `internal/netproto` | Wire protocol types and helpers, free of game state, unit-testable |
| `internal/visualdepth` | Visual depth buffer used for billboard text occlusion, unit-tested |

Windows build helpers: `build.bat` produces `fsbox.exe` at the root, and
`server/build.bat` produces `server/fsbox-server.exe` (run with no flags to
start an authoritative headless server).

The arena is rebuilt on every load; only dynamic props, their frozen state, and
the player pose are stored in `fsbox_save.json` (created the first time you
press F5). Saves are versioned so incompatible files are rejected with a toast
instead of corrupting the scene. Preferences live separately in
`fsbox_settings.json`, which is written when you change something in the
settings menu or close the game.

## Smoke test

For automated render verification:

```powershell
go run . -smoke -screenshot preview.png -frames 60
```

The smoke mode runs the simulation for a number of frames, writes a screenshot,
then samples the image and prints markers such as:

```text
SMOKE_OK screenshot=preview.png bytes=177824
SMOKE_PIXELS size=1280x720 unique_sampled=1652 nonblack=100.0% quadrants=[...]
SMOKE_WORLD props=35 bodies=51 player=(0.00,0.85,12.00) grounded=true yaw=0.000 pitch=-0.120 ... fps=156
```

`SMOKE_OK` means the screenshot was written, `SMOKE_PIXELS` reports sampled
colour variety and non-black coverage, and `SMOKE_WORLD` reports the live prop,
body, player and camera state. The command exits non-zero if the render is
blank or the screenshot cannot be written.

Options:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-smoke` | off | Run the automated smoke test and exit |
| `-screenshot` | `preview.png` | Output path for the smoke screenshot |
| `-frames` | `24` | Number of simulated frames before the screenshot |
| `-ui` | empty | Open `settings`, `spawn`, `tools` or `help` in the smoke screenshot |

For example, to capture the spawn browser, the tool selector and the manual:

```powershell
go run . -smoke -ui spawn -screenshot spawn_menu.png -frames 10
go run . -smoke -ui tools -screenshot tool_menu.png -frames 10
go run . -smoke -ui help -screenshot manual.png -frames 10
```

## Notes and limitations

- The game is designed for a desktop window with mouse lock; there is no touch
  or gamepad support yet.
- The tool gun viewmodel is a separate close-range 3D pass built from simple
  boxes; it is deliberately stylised, not an imported weapon model.
- The arena is intentionally open and small so physics props stay close enough
  to interact with. Props that fall out of bounds are removed automatically.
- The renderer uses simple shapes, generated meshes, flat colours, fake
  elliptical shadows, and 3D wire outlines instead of imported art assets, which
  keeps the project self-contained.

## Credits

- [raylib](https://www.raylib.com/) and
  [raylib-go](https://github.com/gen2brain/raylib-go) for rendering, input and
  window management.
- [box3d-go](https://github.com/gataengine/box3d-go), a pure-Go 3D rigid-body
  physics engine, for collision detection and dynamics.
- [purego](https://github.com/ebitengine/purego) for enabling the CGO-free
  raylib backend.
