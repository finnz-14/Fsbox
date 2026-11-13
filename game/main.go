package game

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

func Run() int {
	// A binary named like the dedicated server defaults to server mode so that
	// double-clicking server/fsbox-server.exe (or running it without flags)
	// starts the authoritative server instead of the game window.
	defaultServer := false
	if exe, err := os.Executable(); err == nil {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe)))
		if strings.Contains(base, "server") {
			defaultServer = true
		}
	}
	mapName := flag.String("map", "", "start this map directly instead of showing the main menu")
	serverMode := flag.Bool("server", defaultServer, "run a dedicated headless server")
	port := flag.Int("port", netDefaultPort, "dedicated server port")
	connectAddr := flag.String("connect", "", "join a dedicated server at host[:port]")
	playerName := flag.String("name", "PLAYER", "player callsign for multiplayer")
	flag.Parse()

	// Dedicated servers never open the visible game window. runDedicatedServer
	// creates a hidden raylib context so content models and shaders still load.
	if *serverMode {
		return runDedicatedServer(*mapName, *port)
	}

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
	if *connectAddr != "" {
		addr := normalizeNetAddr(*connectAddr)
		c, err := dialNetClient(addr, *playerName)
		if err != nil {
			fmt.Printf("connect to %s failed: %v\n", addr, err)
			return 1
		}
		g.startNetClient(c)
	} else if *mapName != "" {
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
