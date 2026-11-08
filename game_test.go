package main

import (
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"

	box3d "github.com/gataengine/box3d-go"
)

// newHeadlessGame builds the simulation without touching raylib's renderer, so
// world, save and physics behaviour can be tested from the command line.
func newHeadlessGame(t *testing.T) *Game {
	t.Helper()

	g := &Game{
		world:    newPhysicsWorld(),
		byBody:   make(map[box3d.BodyID]*Entity),
		rng:      rand.New(rand.NewSource(1)),
		state:    statePlaying,
		settings: defaultSettings(),
	}
	g.player = newPlayer(g.world, defaultSpawn, 0, -0.12, false)
	g.buildArena()
	g.buildStarterProps()
	t.Cleanup(g.Close)
	return g
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	want := len(g.props)
	if want < 20 {
		t.Fatalf("expected a populated starter scene, got %d props", want)
	}

	g.saveGame()
	if _, err := os.Stat(savePath); err != nil {
		t.Fatalf("save file was not written: %v", err)
	}

	g.removeAllProps()
	if len(g.props) != 0 {
		t.Fatalf("expected an empty scene after clearing, got %d props", len(g.props))
	}

	g.loadGame()
	if len(g.props) != want {
		t.Fatalf("load restored %d props, want %d", len(g.props), want)
	}
	if len(g.byBody) != len(g.entities) {
		t.Fatalf("body index has %d entries for %d entities", len(g.byBody), len(g.entities))
	}

	p := g.player.body.GetPosition()
	if math.Abs(float64(p.X-defaultSpawn.X)) > 1e-4 ||
		math.Abs(float64(p.Y-defaultSpawn.Y)) > 1e-4 ||
		math.Abs(float64(p.Z-defaultSpawn.Z)) > 1e-4 {
		t.Fatalf("player position after load = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)",
			p.X, p.Y, p.Z, defaultSpawn.X, defaultSpawn.Y, defaultSpawn.Z)
	}
}

func TestLoadRejectsUnknownVersion(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	want := len(g.props)
	if err := os.WriteFile(savePath, []byte(`{"version":99,"props":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	g.loadGame()
	if len(g.props) != want {
		t.Fatalf("a bad save version changed the scene: %d props, want %d", len(g.props), want)
	}
}

func TestPlayerSettlesOnFloor(t *testing.T) {
	g := newHeadlessGame(t)

	for i := 0; i < 180; i++ {
		g.world.Step(float32(fixedStep), 4)
	}

	p := g.player.body.GetPosition()
	if p.Y < 0.7 || p.Y > 1.05 {
		t.Fatalf("player capsule settled at y=%.3f, expected the floor at about y=0.85", p.Y)
	}
	if math.Abs(float64(p.X)) > 0.1 || math.Abs(float64(p.Z-defaultSpawn.Z)) > 0.5 {
		t.Fatalf("player drifted to (%.3f, %.3f, %.3f)", p.X, p.Y, p.Z)
	}
}

func TestGrabThrowAndSpawn(t *testing.T) {
	g := newHeadlessGame(t)
	g.player.pitch = -0.08

	g.updateAim()
	if g.aim == nil {
		t.Fatal("expected the default view to aim at the starter crate stack")
	}
	g.grab()
	if g.held == nil {
		t.Fatal("grab did not pick up the prop under the crosshair")
	}
	held := g.held

	g.throwHeld()
	if g.held != nil {
		t.Fatal("throw did not release the held prop")
	}
	v := held.Body.GetLinearVelocity()
	if vlen(v.X, v.Y, v.Z) < 10 {
		t.Fatalf("thrown prop speed = %.2f, expected at least 10", vlen(v.X, v.Y, v.Z))
	}

	before := len(g.props)
	g.ghostPos = rlv(0, 1.2, 6)
	g.ghostOK = true
	g.spawnSelected()
	if len(g.props) != before+1 {
		t.Fatalf("spawn added %d props, want 1", len(g.props)-before)
	}
}

func TestNormalizeQuatAndPresetSupport(t *testing.T) {
	q := normalizeQuat(box3d.Quat{V: bv(0, 2, 0), S: 2})
	l := math.Sqrt(float64(q.V.X*q.V.X + q.V.Y*q.V.Y + q.V.Z*q.V.Z + q.S*q.S))
	if math.Abs(l-1) > 1e-5 {
		t.Fatalf("normalized quaternion length = %.6f", l)
	}

	crate := presets[0]
	if got := crate.support(bv(0, 1, 0)); math.Abs(float64(got-0.5)) > 1e-6 {
		t.Fatalf("crate support along +Y = %.3f, want 0.500", got)
	}
	if got := presets[1].support(bv(1, 0, 0)); math.Abs(float64(got-0.5)) > 1e-6 {
		t.Fatalf("ball support along +X = %.3f, want 0.500", got)
	}
}

func TestGravityToolScalesWorldGravity(t *testing.T) {
	g := newHeadlessGame(t)

	g.setGravity(0.5)
	if got := g.world.GetGravity().Y; math.Abs(float64(got+10)) > 1e-4 {
		t.Fatalf("half gravity = %.3f, want -10", got)
	}

	g.setGravity(0)
	if got := g.world.GetGravity().Y; math.Abs(float64(got)) > 1e-4 {
		t.Fatalf("zero gravity = %.3f, want 0", got)
	}

	g.setGravity(-4)
	if got := g.settings.Gravity; got != 0 {
		t.Fatalf("gravity multiplier clamped to %.3f, want 0", got)
	}
	g.setGravity(9)
	if got := g.settings.Gravity; got != 2.5 {
		t.Fatalf("gravity multiplier clamped to %.3f, want 2.5", got)
	}
}

func TestSphereRolls(t *testing.T) {
	g := newHeadlessGame(t)

	e := g.spawnPreset(presets[1], bp(-12, 0.52, -12), box3d.QuatIdentity, colBall)
	if e == nil {
		t.Fatal("could not spawn the test ball")
	}
	e.Body.SetLinearVelocity(bv(6, 0, 0))

	for i := 0; i < 120; i++ {
		g.world.Step(float32(fixedStep), 4)
	}

	q := normalizeQuat(e.Body.GetRotation())
	angle := 2 * float32(math.Acos(float64(clampf(q.S, -1, 1))))
	if angle < 0.25 {
		t.Fatalf("ball rotated %.3f rad, expected visible rolling", angle)
	}
	p := e.Body.GetPosition()
	if p.X <= -12 {
		t.Fatalf("ball did not travel forward: x=%.3f", p.X)
	}
}

func TestSpawnMenuDropsSelectedProp(t *testing.T) {
	g := newHeadlessGame(t)
	g.state = stateSpawn
	g.spawnAnchor = rlv(0, 6, 8)
	g.spawnAnchorOK = true

	before := len(g.props)
	g.spawnFromMenu(7)
	if g.state != stateSpawn {
		t.Fatalf("spawn menu closed after spawning, state=%d", g.state)
	}
	if g.selected != 7 {
		t.Fatalf("selected preset = %d, want 7", g.selected)
	}
	if len(g.props) != before+1 {
		t.Fatalf("spawn menu added %d props, want 1", len(g.props)-before)
	}

	last := g.props[len(g.props)-1]
	if last.Name != presets[7].Name {
		t.Fatalf("spawned %q, want %q", last.Name, presets[7].Name)
	}
	if p := last.position(); math.Abs(float64(p.Y-6)) > 0.25 {
		t.Fatalf("spawned prop y=%.3f, want the menu anchor at 6", p.Y)
	}
}

func TestSettingsLoadClampsValues(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(settingsPath, []byte(`{"gravity_scale":99,"mouse_sensitivity":0,"fov":5,"invert_y":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := loadSettings()
	if s.Gravity != 2.5 || s.Sensitivity != 0 || s.FOV != 60 || !s.InvertY {
		t.Fatalf("settings were not clamped: %+v", s)
	}
}

func TestFreezeLocksAndThawsBody(t *testing.T) {
	g := newHeadlessGame(t)

	e := g.spawnPreset(presets[0], bp(0, 6, 4), box3d.QuatIdentity, colCrate)
	if e == nil {
		t.Fatal("could not spawn the test crate")
	}
	g.setFrozen(e, true)
	if !e.Frozen {
		t.Fatal("entity was not marked frozen")
	}
	if got := e.Body.GetType(); got != box3d.StaticBody {
		t.Fatalf("frozen body type = %v, want StaticBody", got)
	}
	if e.Body.GetLinearVelocity().Y != 0 {
		t.Fatal("frozen body still had velocity")
	}

	// A frozen prop must not react to gravity.
	for i := 0; i < 30; i++ {
		g.world.Step(float32(fixedStep), 4)
	}
	if p := e.position(); math.Abs(float64(p.Y-6)) > 0.01 {
		t.Fatalf("frozen prop drifted to y=%.3f", p.Y)
	}

	g.setFrozen(e, false)
	if e.Frozen {
		t.Fatal("entity stayed frozen after thaw")
	}
	if got := e.Body.GetType(); got != box3d.DynamicBody {
		t.Fatalf("thawed body type = %v, want DynamicBody", got)
	}
	if g.frozenCount() != 0 {
		t.Fatalf("frozen count = %d, want 0", g.frozenCount())
	}
}

func TestDuplicateCopiesShapeAndMaterial(t *testing.T) {
	g := newHeadlessGame(t)

	src := g.spawnPreset(presets[4], bp(-4, 5, -4), quatYaw(0.3), colAnvil)
	if src == nil {
		t.Fatal("could not spawn the source anvil")
	}
	copyPos := bp(-4, 6.5, -4)
	copy := g.spawnCopy(src, copyPos, quatYaw(0.3))
	if copy == nil {
		t.Fatal("spawnCopy returned nil")
	}
	if copy.Kind != src.Kind || copy.Name != src.Name {
		t.Fatalf("copy = %s/%s, want %s/%s", copy.Kind, copy.Name, src.Kind, src.Name)
	}
	if copy.Half != src.Half || copy.Density != src.Density || copy.Color != src.Color {
		t.Fatalf("copy did not preserve shape/material: %+v vs %+v", copy, src)
	}
	if p := copy.position(); math.Abs(float64(p.Y-copyPos.Y)) > 0.05 {
		t.Fatalf("copy spawned at y=%.3f, want %.3f", p.Y, copyPos.Y)
	}
}

func TestRemoveInRadiusOnlyDeletesNearbyProps(t *testing.T) {
	g := newHeadlessGame(t)

	g.removeAllProps()
	near := g.spawnPreset(presets[0], bp(0, 1, 0), box3d.QuatIdentity, colCrate)
	far := g.spawnPreset(presets[0], bp(12, 1, 12), box3d.QuatIdentity, colCrate)
	if near == nil || far == nil {
		t.Fatal("could not spawn test props")
	}

	if n := g.removeInRadius(rlv(0, 1, 0), 2.5); n != 1 {
		t.Fatalf("removeInRadius deleted %d props, want 1", n)
	}
	if len(g.props) != 1 || g.props[0] != far {
		t.Fatalf("wrong props survived: %d remain", len(g.props))
	}
	if g.byBody[far.Body] != far {
		t.Fatal("body index lost the surviving prop")
	}
}

func TestToolCyclingWraps(t *testing.T) {
	g := newHeadlessGame(t)
	g.equipTool(toolPhysgun)
	g.cycleTool(-1)
	if g.tool != toolCount-1 {
		t.Fatalf("cycling back from the first tool gave %d, want %d", g.tool, toolCount-1)
	}
	g.cycleTool(1)
	if g.tool != toolPhysgun {
		t.Fatalf("cycling forward wrapped to %d, want 0", g.tool)
	}
}

func TestWheelDistanceClampsAndDrivesHold(t *testing.T) {
	g := newHeadlessGame(t)
	g.spawnDist = defaultPlaceDist

	g.adjustDistance(1000)
	if got := g.placementDistance(); math.Abs(float64(got-maxPlaceDist)) > 1e-5 {
		t.Fatalf("distance upper clamp = %.2f, want %.2f", got, maxPlaceDist)
	}
	g.adjustDistance(-1000)
	if got := g.placementDistance(); math.Abs(float64(got-minPlaceDist)) > 1e-5 {
		t.Fatalf("distance lower clamp = %.2f, want %.2f", got, minPlaceDist)
	}

	g.spawnDist = defaultPlaceDist
	g.held = g.props[0]
	g.adjustDistance(1)
	want := float32(defaultPlaceDist + placeDistStep)
	if math.Abs(float64(g.holdDist-want)) > 1e-5 {
		t.Fatalf("wheel did not move the hold point: %.2f, want %.2f", g.holdDist, want)
	}
	if math.Abs(float64(g.spawnDist-want)) > 1e-5 {
		t.Fatalf("wheel did not update the placement distance: %.2f, want %.2f", g.spawnDist, want)
	}
	g.held = nil
}

func TestSpawnMenuFiltering(t *testing.T) {
	g := newHeadlessGame(t)

	g.spawnCat = "ROUNDS"
	rounds := g.filteredPresetIndices()
	if len(rounds) != 3 {
		t.Fatalf("ROUNDS category has %d props, want 3", len(rounds))
	}
	for _, i := range rounds {
		if !strings.Contains(strings.ToLower(presets[i].Name), "ball") &&
			!strings.Contains(strings.ToLower(presets[i].Name), "pebble") &&
			!strings.Contains(strings.ToLower(presets[i].Name), "boulder") {
			t.Fatalf("ROUNDS category contains %q", presets[i].Name)
		}
	}

	g.spawnCat = "ALL"
	g.spawnSearch = "bar"
	bar := g.filteredPresetIndices()
	if len(bar) != 1 || presets[bar[0]].Name != "Barrel" {
		t.Fatalf("search for 'bar' returned %v", bar)
	}

	g.spawnSearch = "zzz"
	if got := g.filteredPresetIndices(); len(got) != 0 {
		t.Fatalf("nonsense search returned %v", got)
	}
}

func TestSpawnMenuLaysOutRepeatedSpawns(t *testing.T) {
	g := newHeadlessGame(t)
	g.player.yaw = 0
	g.player.pitch = -0.1
	g.state = statePlaying
	g.openSpawnMenu()
	if g.state != stateSpawn {
		t.Fatalf("spawn menu did not open, state=%d", g.state)
	}

	before := len(g.props)
	g.spawnFromMenu(0)
	g.spawnFromMenu(0)
	if got := len(g.props) - before; got != 2 {
		t.Fatalf("spawned %d props, want 2", got)
	}
	a := g.props[len(g.props)-2].position()
	b := g.props[len(g.props)-1].position()
	dist := vlen(a.X-b.X, 0, a.Z-b.Z)
	if dist < 0.5 {
		t.Fatalf("repeated spawns overlapped: centres %.2f m apart", dist)
	}
	if g.menuSpawnN != 2 {
		t.Fatalf("spawn layout counter = %d, want 2", g.menuSpawnN)
	}
}

func TestSaveLoadPreservesFrozenProps(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	e := g.spawnPreset(presets[0], bp(0, 6, 0), box3d.QuatIdentity, colCrate)
	if e == nil {
		t.Fatal("could not spawn the test crate")
	}
	g.setFrozen(e, true)
	wantFrozen := g.frozenCount()

	g.saveGame()
	g.removeAllProps()
	if g.frozenCount() != 0 {
		t.Fatal("props remained after clearing")
	}
	g.loadGame()
	if got := g.frozenCount(); got != wantFrozen {
		t.Fatalf("load restored %d frozen props, want %d", got, wantFrozen)
	}
}
