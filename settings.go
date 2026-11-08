package main

import (
	"encoding/json"
	"fmt"
	"os"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	settingsPath = "fsbox_settings.json"
	// baseGravity is the world gravity at 1.00x. The gravity slider and the
	// gravity tool scale this value, so 0.00x is true zero-G.
	baseGravity = float32(-20)
)

// Settings holds player preferences. Values are clamped when loaded so a hand
// edited or corrupted JSON file can never break the simulation.
type Settings struct {
	Gravity     float32 `json:"gravity_scale"`
	Sensitivity float32 `json:"mouse_sensitivity"`
	FOV         float32 `json:"fov"`
	UIScale     float32 `json:"ui_scale"`
	InvertY     bool    `json:"invert_y"`
	Shadows     bool    `json:"shadows"`
	DisplayMode int     `json:"display_mode"`
	RawInput    bool    `json:"raw_input"`
}

func defaultSettings() Settings {
	return Settings{
		Gravity:     1.0,
		Sensitivity: 1.0,
		FOV:         72,
		UIScale:     1.0,
		DisplayMode: 1, // Windowed fullscreen by default (borderless, alt-tab friendly)
		InvertY:     false,
		Shadows:     true,
	}
}

func (s *Settings) clampValues() {
	s.Gravity = clampf(s.Gravity, 0, 2.5)
	s.Sensitivity = clampf(s.Sensitivity, 0, 1)
	s.FOV = clampf(s.FOV, 60, 105)
	s.UIScale = clampf(s.UIScale, 0.75, 1.5)
	s.DisplayMode = clampInt(s.DisplayMode, 0, 1)
}

func loadSettings() Settings {
	s := defaultSettings()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultSettings()
	}
	s.clampValues()
	return s
}

func (g *Game) saveSettings() error {
	g.settings.clampValues()
	data, err := json.MarshalIndent(g.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0o644)
}

// applyGravity pushes the gravity multiplier into the physics world and wakes
// every prop so a change is felt immediately instead of after the next touch.
func (g *Game) applyGravity() {
	if g.world.IsNull() {
		return
	}
	g.world.SetGravity(box3d.Vec3{Y: baseGravity * g.settings.Gravity})
	for _, e := range g.props {
		if e.Frozen {
			continue
		}
		e.Body.SetAwake(true)
	}
}

// setGravity is the single entry point for every gravity control.
func (g *Game) setGravity(mult float32) {
	g.settings.Gravity = clampf(mult, 0, 2.5)
	g.settingsDirty = true
	g.applyGravity()
}

func gravityLabel(mult float32) string {
	switch {
	case mult <= 0.001:
		return "ZERO-G"
	case mult < 0.7:
		return "LOW GRAVITY"
	case mult <= 1.15:
		return "EARTH GRAVITY"
	default:
		return "HEAVY GRAVITY"
	}
}

var gravityCycle = []float32{1.0, 0.4, 0.0, 1.6}

// cycleGravity steps through the useful presets: earth, moon, zero-G, heavy.
func (g *Game) cycleGravity() {
	cur := g.settings.Gravity
	bestIdx := 0
	best := float32(1e9)
	for i, v := range gravityCycle {
		if d := absf(cur - v); d < best {
			best = d
			bestIdx = i
		}
	}
	next := gravityCycle[(bestIdx+1)%len(gravityCycle)]
	g.setGravity(next)
	g.toast(fmt.Sprintf("%s  %.2fx   (G cycle, [ ] fine-tune)", gravityLabel(next), next), colEnergy)
}

func (g *Game) nudgeGravity(delta float32) {
	g.setGravity(g.settings.Gravity + delta)
	g.toast(fmt.Sprintf("Gravity %.2fx", g.settings.Gravity), colEnergy)
}

func (g *Game) setDisplayMode(mode int) {
	g.settings.DisplayMode = clampInt(mode, 0, 1)
	g.settingsDirty = true
	g.applyDisplayMode()
}

func (g *Game) applyDisplayMode() {
	if g.currentDisplayMode == g.settings.DisplayMode {
		return
	}
	// Restore previous mode.
	switch g.currentDisplayMode {
	case 0:
		rl.ToggleFullscreen()
	case 1:
		g.restoreWindowed()
	default:
		if g.currentDisplayMode == 2 {
			rl.ToggleFullscreen()
		} else {
			g.restoreWindowed()
		}
	}

	switch g.settings.DisplayMode {
	case 0:
		saveWindowRect()
		rl.ToggleFullscreen()
	case 1:
		g.applyBorderlessWindowed()
	}
	g.currentDisplayMode = g.settings.DisplayMode
}

func (g *Game) displayModeName() string {
	switch g.settings.DisplayMode {
	case 0:
		return "FULLSCREEN"
	default:
		return "WINDOWED FULLSCREEN"
	}
}
