# Fsbox dedicated server

The dedicated server is the same binary as the game, started with `-server`.
It runs the authoritative physics simulation, loads mods, and streams snapshots
to connected clients.

## Build

```bat
server\build.bat
```

This writes `server\fsbox-server.exe`. Use `server\build.bat release` to exclude
the smoke-test code.

## Run

Run the server from the repository root so it can see `Mods\` and write its
settings next to the game:

```bat
server\fsbox-server.exe -server -port 27015 -map default
```

Flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-server` | off | Run headless and serve clients. |
| `-port` | `27015` | TCP listen port. |
| `-map` | current map | Start this map directly. |

The server owns physics and the world state. Stop it with `Ctrl+C`.

## Join

Start the normal game and use **MULTIPLAYER**, or connect directly:

```bat
fsbox.exe -connect 127.0.0.1:27015 -name PLAYER
```

Clients only send input and world intents; the server validates and applies
them, then broadcasts snapshots at 20 Hz.

## Mods

Mods run on the server. Client mods receive forwarded events and can render
local effects. Use the networking helpers from the SDK:

- `NetStatus` / `NetPlayers` inspect the session.
- `BroadcastToast` and `KickPlayer` are server-only.
- `SendNetEvent` sends custom events in either direction.

See `Devkit\docs.html` (Networking tab) for the full API.
