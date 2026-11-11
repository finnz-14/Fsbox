//go:build !release

package main

// Development-only smoke test support.
//
// Release builds use `go build -tags release` (or `build.bat release`) and
// compile smoke_release.go instead, so none of this file is shipped. Keeping
// all smoke code here also makes it safe to delete when publishing the game.

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	smokeMode   = flag.Bool("smoke", false, "render a short automated smoke test and exit")
	smokeShot   = flag.String("screenshot", "", "screenshot path used by -smoke (default preview.png)")
	smokeFrames = flag.Int("frames", 24, "number of frames to render in -smoke mode")
	smokeUI     = flag.String("ui", "", "optional menu shown in -smoke mode: settings, spawn, tools or help")
)

func smokeRequested() bool {
	return *smokeMode
}

// runSmokeIfRequested returns (handled, ok). When -smoke was not passed it
// returns (false, true) and the normal game loop runs.
func runSmokeIfRequested(g *Game) (bool, bool) {
	if !*smokeMode {
		return false, true
	}
	return true, runSmoke(g, *smokeShot, *smokeFrames, *smokeUI)
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
	case "mods":
		g.modsScroll = 0
		g.state = stateModsMenu
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
