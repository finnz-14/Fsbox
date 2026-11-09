package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

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
	smoke := flag.Bool("smoke", false, "render a short automated smoke test and exit")
	shot := flag.String("screenshot", "", "screenshot path used by -smoke (default preview.png)")
	frames := flag.Int("frames", 24, "number of frames to render in -smoke mode")
	ui := flag.String("ui", "", "optional menu shown in -smoke mode: settings, spawn, tools or help")
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

	if *smoke {
		rl.SetTargetFPS(0)
	} else {
		rl.SetTargetFPS(60)
	}

	g := NewGame()
	defer g.Close()
	g.applyDisplayMode()

	rl.DisableCursor()
	g.cursorLocked = true
	g.ignoreMouse = true

	if *smoke {
		if !runSmoke(g, *shot, *frames, *ui) {
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

func runSmoke(g *Game, shot string, frames int, ui string) bool {
	if frames < 4 {
		frames = 4
	}
	if shot == "" {
		shot = "preview.png"
	}

	switch ui {
	case "settings":
		g.openSettings()
	case "spawn":
		g.openSpawnMenu()
	case "tools":
		g.openToolMenu()
	case "help":
		g.state = stateHelp
	}

	// raylib joins the screenshot name to the working directory it captured
	// during InitWindow, so turn absolute paths into equivalent relative ones.
	shotName := shot
	if filepath.IsAbs(shot) {
		if wd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(wd, shot); err == nil {
				shotName = rel
			}
		}
	}

	for i := 0; i < frames; i++ {
		g.Update(fixedStep)
		g.Draw()
		if i == frames-2 {
			rl.TakeScreenshot(shotName)
		}
	}

	info, err := os.Stat(shot)
	if err != nil {
		fmt.Printf("SMOKE_FAIL screenshot=%s err=%v\n", shot, err)
		return false
	}
	fmt.Printf("SMOKE_OK screenshot=%s bytes=%d\n", shot, info.Size())

	stats, err := analyzeScreenshot(shot)
	if err != nil {
		fmt.Printf("SMOKE_PIXELS error=%v\n", err)
		return false
	}
	fmt.Printf("SMOKE_PIXELS size=%dx%d unique_sampled=%d nonblack=%.1f%% quadrants=%v\n",
		stats.Width, stats.Height, stats.Unique, stats.NonBlack*100, stats.Quadrants)
	if stats.Unique < 12 || stats.NonBlack < 0.10 {
		fmt.Println("SMOKE_WARN screenshot appears blank or nearly blank")
		return false
	}

	p := g.player.body.GetPosition()
	fmt.Printf("SMOKE_WORLD props=%d bodies=%d player=(%.2f,%.2f,%.2f) grounded=%v yaw=%.3f pitch=%.3f eye=(%.2f,%.2f,%.2f) target=(%.2f,%.2f,%.2f) fps=%d\n",
		len(g.props), len(g.entities), p.X, p.Y, p.Z, g.player.grounded, g.player.yaw, g.player.pitch,
		g.camera.Position.X, g.camera.Position.Y, g.camera.Position.Z,
		g.camera.Target.X, g.camera.Target.Y, g.camera.Target.Z, rl.GetFPS())
	return true
}

type shotStats struct {
	Width     int
	Height    int
	Unique    int
	NonBlack  float64
	Quadrants [4][3]uint8
}

func analyzeScreenshot(path string) (shotStats, error) {
	var out shotStats

	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return out, err
	}

	b := img.Bounds()
	out.Width = b.Dx()
	out.Height = b.Dy()
	midX := b.Min.X + b.Dx()/2
	midY := b.Min.Y + b.Dy()/2

	uniq := make(map[uint32]struct{}, 4096)
	var sums [4][3]uint64
	var counts [4]uint64

	total := 0
	nonBlack := 0
	for y := b.Min.Y; y < b.Max.Y; y += 5 {
		for x := b.Min.X; x < b.Max.X; x += 5 {
			r, gg, bb, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(gg >> 8)
			b8 := uint8(bb >> 8)

			key := uint32(r8)<<16 | uint32(g8)<<8 | uint32(b8)
			uniq[key] = struct{}{}
			total++
			if int(r8)+int(g8)+int(b8) > 24 {
				nonBlack++
			}

			q := 0
			if x >= midX {
				q += 1
			}
			if y >= midY {
				q += 2
			}
			sums[q][0] += uint64(r8)
			sums[q][1] += uint64(g8)
			sums[q][2] += uint64(b8)
			counts[q]++
		}
	}

	out.Unique = len(uniq)
	if total > 0 {
		out.NonBlack = float64(nonBlack) / float64(total)
	}
	for i := 0; i < 4; i++ {
		if counts[i] == 0 {
			continue
		}
		out.Quadrants[i][0] = uint8(sums[i][0] / counts[i])
		out.Quadrants[i][1] = uint8(sums[i][1] / counts[i])
		out.Quadrants[i][2] = uint8(sums[i][2] / counts[i])
	}
	return out, nil
}
