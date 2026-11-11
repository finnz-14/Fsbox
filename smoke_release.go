//go:build release

package main

// Release builds contain no smoke-test code or flags. Use `build.bat release`
// (or `go build -tags release`) when publishing the game.

func smokeRequested() bool {
	return false
}

func runSmokeIfRequested(g *Game) (bool, bool) {
	return false, true
}
