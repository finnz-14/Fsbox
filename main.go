package main

import (
	"flag"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

func main() {
	os.Exit(run())
}

func run() int {
	mapName := flag.String("map", "", "start this map directly instead of showing the main menu")
	flag.Parse()

	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(screenWidth, screenHeight, "FSBOX // 3D physics sandbox")
	rl.InitAudioDevice()
	defer rl.CloseWindow()
	if _, err := os.Stat("fsbox.png"); err == nil {
		if icon := rl.LoadImage("fsbox.png"); icon != nil {
			rl.SetWindowIcon(*icon)
			rl.UnloadImage(icon)
		}
	} else if _, err := os.Stat("fsbox.ico"); err == nil {
		if icon := rl.LoadImage("fsbox.ico"); icon != nil {
			rl.SetWindowIcon(*icon)
			rl.UnloadImage(icon)
		}
	}
	rl.SetWindowMinSize(960, 620)
	rl.SetExitKey(rl.KeyNull)

	if smokeRequested() {
		rl.SetTargetFPS(0)
	} else {
		rl.SetTargetFPS(60)
	}

	g := NewGame()
	defer g.Close()
	if *mapName != "" {
		g.selectedMap = *mapName
		g.resetWorld(true, defaultSpawn, 0, -0.12, false)
		g.state = statePlaying
		g.resumePlaying()
	}
	g.applyDisplayMode()

	if g.state == statePlaying {
		rl.DisableCursor()
		g.cursorLocked = true
	} else {
		rl.EnableCursor()
		g.cursorLocked = false
	}
	g.ignoreMouse = true

	if handled, ok := runSmokeIfRequested(g); handled {
		if !ok {
			return 1
		}
		return 0
	}

	for !rl.WindowShouldClose() && !g.quitRequested {
		dt := float64(rl.GetFrameTime())
		if dt > 0.1 {
			dt = 0.1
		}
		g.Update(dt)
		g.Draw()
	}
	return 0
}
