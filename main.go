// fsbox is a 3D physics sandbox built on raylib and box3d.
// The game itself lives in game/; this file is a thin launcher.
// The Content/ folder at the repository/executable root holds the game's
// embedded resource files; it is embedded here so game/ can use it without
// depending on a Go package inside the resource folder.
package main

import (
	"embed"
	"os"

	"sandbox3d/game"
)

//go:embed Content/Spawnables
var embeddedSpawnables embed.FS

//go:embed Content/Maps
var embeddedMaps embed.FS

func main() {
	game.SetEmbeddedAssets(embeddedSpawnables, embeddedMaps)
	os.Exit(game.Run())
}
