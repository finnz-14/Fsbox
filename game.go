package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	fixedStep = 1.0 / 60.0
	maxSteps  = 8

	// Wheel-controlled placement range shared by the spawner ghost and the
	// physgun hold point. Keeping it in metres makes the HUD readout honest.
	defaultPlaceDist = 6.0
	minPlaceDist     = 1.4
	maxPlaceDist     = 12.0
	placeDistStep    = 0.4
)

type UIState int

const (
	stateMainMenu UIState = iota
	statePlaying
	statePaused
	stateHelp
	stateSettings
	stateSpawn
	stateToolMenu
	stateMapMenu
)

// toolKeys maps the 1-7 row to the tool gun modes.
var toolKeys = []int32{
	rl.KeyOne, rl.KeyTwo, rl.KeyThree, rl.KeyFour,
	rl.KeyFive, rl.KeySix, rl.KeySeven,
}

type Toast struct {
	Text    string
	Color   rl.Color
	Life    float32
	MaxLife float32
}

type Particle struct {
	Pos     rl.Vector3
	Vel     rl.Vector3
	Color   rl.Color
	Size    float32
	Life    float32
	MaxLife float32
	Gravity float32
}

type Shockwave struct {
	Center    rl.Vector3
	Radius    float32
	MaxRadius float32
	Color     rl.Color
	Life      float32
	MaxLife   float32
}

// Game owns the physics world, the player, every entity and all transient
// effects. The simulation runs at a fixed 60 Hz while rendering follows the
// monitor's frame rate.
type Game struct {
	world       box3d.WorldID
	player      *Player
	entities    []*Entity
	props       []*Entity
	byBody      map[box3d.BodyID]*Entity
	byID        map[int32]*Entity
	nextEntityID int32

	held       *Entity
	holdDist   float32
	holdOffset box3d.Vec3
	spawnDist float32
	aim       *Entity
	aimPoint  rl.Vector3
	aimNormal rl.Vector3

	ghostPos    rl.Vector3
	ghostNormal rl.Vector3
	ghostOK     bool

	selected   int
	propSerial int
	lastSpawn  *Entity

	tool     ToolID
	toolKick float32
	toolBob  float32
	uiTime   float64

	rng  *rand.Rand
	time float64
	acc  float64

	state UIState

	toasts    []*Toast
	particles []*Particle
	waves     []*Shockwave
	shake     float64

	shadowShader rl.Shader
	mainShader   rl.Shader
	shadowMap    rl.RenderTexture2D
	lightVP      rl.Matrix
	sunCam       rl.Camera3D
	hasShadow    bool
	// modelPreview is the offscreen target used by the spawn menu to draw a 3D
	// content model inside the inspection well.
	modelPreview rl.RenderTexture2D
	mods        *ModManager

	cube rl.Model
	font rl.Font
	// fontCustom is true when a real TTF was loaded, in which case Close must
	// release it. The built-in raylib font is owned by the library.
	fontCustom bool

	camera        rl.Camera3D
	cursorLocked  bool
	ignoreMouse   bool
	justResumed   bool
	quitRequested bool

	settings            Settings
	settingsDraft       Settings
	settingsDisplayOpen bool
	currentDisplayMode  int
	settingsDirty       bool
	persistSettings     bool
	settingsReturn      UIState
	draggingSlider      int
	menuIgnoreClick     bool
	spawnAnchor         rl.Vector3
	spawnAnchorOK       bool

	spawnTab      int
	spawnCat      string

	selectedMap string
	spawnSearch   string
	searchActive  bool
	menuToolSel   ToolID
	menuWorldSel  int
	lastCardClick float64
	lastCardIndex int
	lastCardKind  int
	toolMenuSel   ToolID
	menuSpawnN    int

	// ------------------------------------------------------------------
	// Phase 2+ modding runtime state
	// ------------------------------------------------------------------

	// Weapon registry: mods register custom weapons; index into this slice is
	// the weapon ID used by host API calls.
	modWeapons       []*ModWeapon
	modWeaponLast    int32
	activeModWeapon  int32

	// Custom tool registrations from mods.
	modTools        []*ModTool
	modToolLast     int32
	activeModTool   int32
	menuModToolSel  int32
	toolModColor    rl.Color

	// ------------------------------------------------------------------
	// Extended modding runtime state
	// ------------------------------------------------------------------
	modModels       map[int32]rl.Model
	modModelLast    int32
	modModelAnims   map[int32][]rl.ModelAnimation
	modShaders      map[int32]rl.Shader
	modShaderLast   int32
	modUIElements   []ModUIElement
	modUIElementLast int32

	// Camera/player model control.
	cameraMode     int32 // 0 = first person, 1 = third person
	cameraDistance float32
	cameraOffset   rl.Vector3
	playerModel    rl.Model
	playerHasModel bool

	// Active constraints created by mods via box3d joints.
	modConstraints    []*ModConstraint
	modConstraintLast int32

	// Input hooks: key code -> list of event names to fire.
	modInputHooks map[int32][]string
	modMouseHooks map[int32][]string

	// Mod-owned particles (simple point particles rendered by the engine).
	modParticles     []*ModParticle
	modParticleLast  int32

	// Mod projectiles (real dynamic bodies tracked for collision/damage).
	modProjectiles []*ModProjectile

	// Queued 2D UI draw commands and 3D debug draw commands. Mods can call
	// ui.draw_* / render.*_3d from any callback; the engine replays the queue
	// during the draw phase.
	modDrawQueue  []ModDrawCmd
	modDebugQueue []ModDebugCmd
	modDrawActive bool

	// Audio resources owned by mods. Lazily initialised on first use.
	modSounds    map[string]rl.Sound
	modMusic     map[string]rl.Music
	modMusicVol  map[string]float32
	modAudioReady bool

	// Mod hooks that should run during the HUD phase.
	modHUDHooks []string

	// Frame counter and simulation time scale.
	frame     int64
	timeScale float64

	// Rendering overrides exposed to mods.
	sunDir    rl.Vector3
	skyTop    rl.Color
	skyBottom rl.Color
}

// ------------------------------------------------------------------
// Phase 2+ data types
// ------------------------------------------------------------------

// WeaponType identifies the firing behavior of a registered weapon.
type WeaponType int8

const (
	WeaponHitscan WeaponType = iota
	WeaponProjectile
	WeaponMelee
)

func weaponTypeFromString(s string) WeaponType {
	switch s {
	case "projectile", "rocket", "bullet":
		return WeaponProjectile
	case "melee", "swing":
		return WeaponMelee
	default:
		return WeaponHitscan
	}
}

func (t WeaponType) String() string {
	switch t {
	case WeaponProjectile:
		return "projectile"
	case WeaponMelee:
		return "melee"
	default:
		return "hitscan"
	}
}

// ModWeapon is a custom weapon registered by a mod.
type ModWeapon struct {
	ID              int32
	Name            string
	Type            WeaponType
	Damage          float32
	Range           float32
	Ammo            int32
	MaxAmmo         int32
	FireRate        float32 // shots per second
	LastShot        float64
	Recoil          float32
	Spread          float32
	ProjectileSpeed float32
	ProjectileLife  float32
	ProjectileSize  float32
	Color           rl.Color
	Owner           string
}

// ModTool is a custom tool registered by a mod.
type ModTool struct {
	ID            int32
	Name          string
	PrimaryDesc   string
	SecondaryDesc string
	ReloadDesc    string
	Color         rl.Color
	PrimaryHook   string
	SecondaryHook string
	ReloadHook    string
	Owner         string
}

// ModConstraint is an active box3d joint created by a mod.
type ModConstraint struct {
	ID      int32
	Type    string
	EntityA int32
	EntityB int32
	JointID box3d.JointID
	Owner   string
}

// ModParticle is a simple point particle owned by a mod.
type ModParticle struct {
	ID      int32
	Pos     rl.Vector3
	Vel     rl.Vector3
	Life    float32
	MaxLife float32
	Color   rl.Color
	Size    float32
	Gravity float32
}

// ModProjectile tracks a dynamic body fired by a mod weapon.
type ModProjectile struct {
	Entity  *Entity
	Damage  float32
	Life    float32
	MaxLife float32
	Owner   string
	Prev    rl.Vector3
	HitMask uint64
}

// ModDrawCmd is a queued 2D draw operation.
type ModDrawCmd struct {
	Kind      string
	Text      string
	X, Y      float32
	W, H      float32
	X2, Y2    float32
	X3, Y3    float32
	Radius    float32
	Size      float32
	Thickness float32
	Color     rl.Color
	Filled    bool
	Segments  int32
}

// ModDebugCmd is a queued 3D debug draw operation.
type ModDebugCmd struct {
	Kind      string
	From      rl.Vector3
	To        rl.Vector3
	Center    rl.Vector3
	Normal    rl.Vector3
	Radius    float32
	Size      rl.Vector3
	Color     rl.Color
	Thickness float32
	Segments  int32
}

// ModUIElement is an interactive UI element registered by a mod.
type ModUIElement struct {
	ID     int32
	Kind   string // button, panel, slider, text_input
	X, Y   float32
	W, H   float32
	Label  string
	Color  rl.Color
	Value  float32
	Min    float32
	Max    float32
	Owner  string
}

// ------------------------------------------------------------------
// Game constructor
// ------------------------------------------------------------------

func NewGame() *Game {
	g := &Game{
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		byBody:          make(map[box3d.BodyID]*Entity),
		byID:            make(map[int32]*Entity),
		nextEntityID:    1,
		state:           stateMainMenu,
		selected:        0,
		selectedMap:     "default",
		tool:            toolPhysgun,
		menuToolSel:     toolPhysgun,
		toolMenuSel:     toolPhysgun,
		spawnCat:        "ALL",
		spawnDist:       defaultPlaceDist,
		settings:        loadSettings(),
		timeScale:       1,
		activeModWeapon: -1,
		activeModTool:   -1,
		menuModToolSel:  -1,
		sunDir:          rlv(-0.55, 0.82, -0.30),
		skyTop:          colSkyTop,
		skyBottom:       colSkyBottom,
	}
	mergeModContent()
	loadContentSpawnables()
	g.font = loadUIFont(&g.fontCustom)
	g.persistSettings = true
	g.cube = rl.LoadModelFromMesh(rl.GenMeshCube(1, 1, 1))
	g.resetWorld(true, defaultSpawn, 0, -0.12, false)
	g.currentDisplayMode = -1
	g.initShadows()
	mm, err := NewModManager(g)
	if err != nil {
		fmt.Println("[mod] manager error:", err)
	} else {
		g.mods = mm
	}
	g.modInputHooks = make(map[int32][]string)
	g.modMouseHooks = make(map[int32][]string)
	g.modSounds = make(map[string]rl.Sound)
	g.modModels = make(map[int32]rl.Model)
	g.modModelAnims = make(map[int32][]rl.ModelAnimation)
	g.modShaders = make(map[int32]rl.Shader)
	g.cameraDistance = 6
	g.cameraOffset = rlv(0, 1.5, 0)
	g.modMusic = make(map[string]rl.Music)
	g.modMusicVol = make(map[string]float32)
	g.toast("FSBOX ONLINE  //  E SPAWN  //  C TOOLS  //  F1 MANUAL", colAccent)
	if g.mods != nil {
		g.mods.FireEvent("game.start", g.engineInfo())
	}
	return g
}

// loadUIFont prefers a technical DIN-style system face over the tiny built-in
// bitmap font. Everything falls back gracefully when no TTF can be loaded.
func loadUIFont(custom *bool) rl.Font {
	candidates := []string{
		"assets/font.ttf",
		"C:/Windows/Fonts/bahnschrift.ttf",
		"C:/Windows/Fonts/segoeui.ttf",
		"C:/Windows/Fonts/consola.ttf",
		"C:/Windows/Fonts/arial.ttf",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		font := rl.LoadFontEx(path, 64, nil, 0)
		if font.Texture.ID != 0 && font.CharsCount > 0 {
			rl.SetTextureFilter(font.Texture, rl.FilterBilinear)
			*custom = true
			return font
		}
	}
	*custom = false
	return rl.GetFontDefault()
}

func (g *Game) Close() {
	if g.persistSettings && g.settingsDirty {
		_ = g.saveSettings()
		g.settingsDirty = false
	}
	if !g.world.IsNull() {
		g.world.Destroy()
		g.world = box3d.NullWorldID
	}
	if g.shadowMap.ID != 0 {
		rl.UnloadRenderTexture(g.shadowMap)
		g.shadowMap.ID = 0
	}
	if g.modelPreview.ID != 0 {
		rl.UnloadRenderTexture(g.modelPreview)
		g.modelPreview.ID = 0
	}
	if g.mods != nil {
		g.mods.Close()
		g.mods = nil
	}
	if g.shadowShader.ID != 0 {
		rl.UnloadShader(g.shadowShader)
		g.shadowShader.ID = 0
	}
	if g.mainShader.ID != 0 {
		rl.UnloadShader(g.mainShader)
		g.mainShader.ID = 0
	}
	if g.fontCustom && g.font.Texture.ID != 0 {
		rl.UnloadFont(g.font)
		g.fontCustom = false
	}
	os.RemoveAll(".merged_content")
}

// ---------------------------------------------------------------------------
// Frame update
// ---------------------------------------------------------------------------

func (g *Game) Update(dt float64) {
	g.frame++
	g.uiTime += dt
	g.updateUIInput()
	g.updateToolAnimation(float32(dt))
	g.fireInputEvents()
	g.updateUIElements()
	if g.state == stateMainMenu {
		g.updateMainMenuInput()
	}
	if g.state == stateMapMenu {
		g.updateMapMenuInput()
	}

	if g.state == statePlaying {
		g.updateLook()
		g.updateCamera()
		g.updateAim()
		g.updateGhost()
		g.updateActionInput()
	} else {
		g.aim = nil
		g.ghostOK = false
	}

	simDt := dt
	if g.timeScale < 0 {
		g.timeScale = 0
	}
	if g.timeScale > 10 {
		g.timeScale = 10
	}
	simDt = dt * g.timeScale

	// The world keeps simulating in every state except the explicit pause
	// menu and the main/map menus, so props and effects keep moving while
	// overlays are open.
	if g.state != statePaused && g.state != stateMainMenu && g.state != stateMapMenu {
		g.acc += simDt
		steps := 0
		for g.acc >= fixedStep && steps < maxSteps {
			g.tick(float32(fixedStep))
			g.acc -= fixedStep
			steps++
		}
		if steps == maxSteps {
			g.acc = 0
		}
		g.updateParticles(float32(simDt))
		g.updateWaves(float32(simDt))
		g.updateModParticles(float32(simDt))
		g.updateMusic()
		g.updateAnimations(float32(simDt))
		if g.shake > 0 {
			g.shake -= simDt
			if g.shake < 0 {
				g.shake = 0
			}
		}
	}

	g.updateToasts(float32(dt))
	if g.mods != nil {
		g.mods.Tick(g, simDt)
	}
	g.updateCamera()
	g.updateThirdPersonCamera()
	g.syncCursor()
}

func (g *Game) resumePlaying() {
	g.state = statePlaying
	g.justResumed = true
	g.ignoreMouse = true
}

func (g *Game) updateUIInput() {
	// Number keys equip a tool gun mode.
	if g.state == statePlaying || g.state == stateToolMenu {
		for i, key := range toolKeys {
			if i < int(toolCount) && rl.IsKeyPressed(key) {
				g.equipTool(ToolID(i))
				if g.state == stateToolMenu {
					g.closeToolMenu()
				}
			}
		}
	}

	// The wheel is a distance control in the field. Holding Ctrl turns it into
	// a tool selector so quick mode changes stay available without the menu.
	// The spawn menu owns its own wheel handling (prop browsing).
	switch g.state {
	case statePlaying:
		if wheel := rl.GetMouseWheelMove(); wheel != 0 {
			if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
				if wheel > 0 {
					g.cycleTool(-1)
				} else {
					g.cycleTool(1)
				}
			} else {
				g.adjustDistance(wheel)
			}
		}
	case stateToolMenu:
		if wheel := rl.GetMouseWheelMove(); wheel > 0 {
			g.toolMenuSel = ToolID((int(g.toolMenuSel) - 1 + int(toolCount)) % int(toolCount))
		} else if wheel < 0 {
			g.toolMenuSel = ToolID((int(g.toolMenuSel) + 1) % int(toolCount))
		}
	}

	if rl.IsKeyPressed(rl.KeyEscape) {
		switch g.state {
		case statePlaying:
			g.state = statePaused
		case statePaused, stateHelp:
			g.resumePlaying()
		case stateSettings:
			g.closeSettings()
		case stateSpawn:
			if g.searchActive {
				g.searchActive = false
			} else {
				g.closeSpawnMenu()
			}
		case stateToolMenu:
			g.closeToolMenu()
		}
	}
	if rl.IsKeyPressed(rl.KeyF1) || rl.IsKeyPressed(rl.KeyH) {
		switch g.state {
		case statePlaying:
			g.state = stateHelp
		case stateHelp:
			g.resumePlaying()
		case statePaused:
			g.state = stateHelp
		}
	}
	if rl.IsKeyPressed(rl.KeyF2) {
		switch g.state {
		case statePlaying, statePaused, stateSpawn, stateToolMenu:
			g.openSettings()
		case stateSettings:
			g.closeSettings()
		}
	}
	if rl.IsKeyPressed(rl.KeyE) && g.held == nil {
		switch g.state {
		case statePlaying:
			g.openSpawnMenu()
		case stateSpawn:
			if !g.searchActive {
				g.closeSpawnMenu()
			}
		}
	}
	if rl.IsKeyPressed(rl.KeyC) {
		switch g.state {
		case statePlaying:
			g.openToolMenu()
		case stateToolMenu:
			g.closeToolMenu()
		}
	}
	if rl.IsKeyPressed(rl.KeyTab) {
		switch g.state {
		case statePlaying:
			g.openSpawnMenu()
		case stateSpawn:
			if !g.searchActive {
				g.closeSpawnMenu()
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyF5) {
		g.saveGame()
	}
	if rl.IsKeyPressed(rl.KeyF9) {
		g.loadGame()
	}

	if rl.IsKeyPressed(rl.KeyR) && rl.IsKeyDown(rl.KeyLeftControl) {
		yaw, pitch := g.player.yaw, g.player.pitch
		g.resetWorld(true, defaultSpawn, yaw, pitch, g.player.fly)
		g.toast("ARENA RESET", colAccent)
	}
	if rl.IsKeyPressed(rl.KeyDelete) && rl.IsKeyDown(rl.KeyLeftControl) {
		g.removeAllProps()
		g.toast("ALL PROPS CLEARED", colWarn)
	}

	clicked := func(rec rl.Rectangle) bool {
		return rl.CheckCollisionPointRec(g.uiMouse(), rec) && rl.IsMouseButtonPressed(rl.MouseButtonLeft)
	}

	switch g.state {
	case statePaused:
		if clicked(g.pauseResumeRect()) {
			g.resumePlaying()
		}
		if clicked(g.pauseSettingsRect()) {
			g.openSettings()
		}
		if clicked(g.pauseHelpRect()) {
			g.state = stateHelp
		}
		if clicked(g.pauseQuitRect()) {
			g.quitRequested = true
		}
	case stateHelp:
		if clicked(g.resumeButtonRect()) {
			g.resumePlaying()
		}
	case stateSettings:
		g.updateSettingsInput()
	case stateSpawn:
		g.updateSpawnMenuInput()
	case stateToolMenu:
		g.updateToolMenuInput()
	}
}

func (g *Game) updateToolAnimation(dt float32) {
	if g.toolKick > 0 {
		g.toolKick = maxf(0, g.toolKick-dt*4.2)
	}
	speed := float32(0)
	if g.player != nil {
		v := g.player.body.GetLinearVelocity()
		speed = vlen(v.X, 0, v.Z)
	}
	g.toolBob += dt * (1.6 + clampf(speed, 0, 10)*0.28)
}

func (g *Game) updateLook() {
	if g.state != statePlaying || !rl.IsWindowFocused() {
		return
	}
	// Stop the camera from looking around while rotating a held prop.
	if g.held != nil && (rl.IsKeyDown(rl.KeyE) || rl.IsKeyDown(rl.KeyR)) {
		// Consume the mouse delta so it only drives rotation.
		_ = rl.GetMouseDelta()
		return
	}
	if g.ignoreMouse {
		g.ignoreMouse = false
		return
	}
	d := rl.GetMouseDelta()
	sens := float32(0.0026) * g.settings.Sensitivity
	if g.settings.RawInput {
		sens = float32(0.0042) * g.settings.Sensitivity
	}
	g.player.yaw += d.X * sens
	if g.settings.InvertY {
		g.player.pitch += d.Y * sens
	} else {
		g.player.pitch -= d.Y * sens
	}
	g.player.pitch = clampf(g.player.pitch, -1.52, 1.52)

	const twoPi = math.Pi * 2
	for g.player.yaw > math.Pi {
		g.player.yaw -= twoPi
	}
	for g.player.yaw < -math.Pi {
		g.player.yaw += twoPi
	}
}

func (g *Game) updateActionInput() {
	if g.justResumed {
		g.justResumed = false
		return
	}

	if rl.IsKeyPressed(rl.KeyV) {
		g.toggleFly()
	}
	if rl.IsKeyPressed(rl.KeyF) {
		g.freezeAimed()
	}
	if rl.IsKeyPressed(rl.KeyG) {
		g.cycleGravity()
	}
	if rl.IsKeyPressed(rl.KeyLeftBracket) {
		g.nudgeGravity(-0.1)
	}
	if rl.IsKeyPressed(rl.KeyRightBracket) {
		g.nudgeGravity(0.1)
	}
	if rl.IsKeyPressed(rl.KeySpace) && !g.player.fly {
		g.player.jumpBuffer = 8
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		g.primaryAction()
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
		g.secondaryAction()
	}
	// F is kept as the physgun grab/throw alias (E now opens the spawn menu).
	if rl.IsKeyPressed(rl.KeyF) && g.tool == toolPhysgun {
		g.primaryAction()
	}
}

func (g *Game) tick(dt float32) {
	g.time += float64(dt)
	g.player.tick(g, dt)
	if g.held != nil {
		g.updateHeld(dt)
	}
	g.world.Step(dt, 2)
	g.updateProjectiles(dt)
	g.enforceBounds()

	p := g.player.body.GetPosition()
	if p.Y < -6 {
		g.respawnPlayer()
	}
	if g.mods != nil {
		g.mods.FireEvent("game.tick", map[string]any{"dt": dt, "time": g.time})
	}
}

func (g *Game) toggleFly() {
	g.player.fly = !g.player.fly
	if g.player.fly {
		g.player.body.SetGravityScale(0)
		g.player.body.SetLinearVelocity(box3d.Vec3Zero)
		g.toast("FLY MODE  //  SPACE / CTRL ALTITUDE", colEnergy)
	} else {
		g.player.body.SetGravityScale(1)
		g.toast("WALK MODE", colAccent)
	}
	g.player.body.SetAwake(true)
}

// placementDistance is the wheel-controlled range used by the spawner ghost
// and as the fallback drop point when no surface is under the crosshair.
func (g *Game) placementDistance() float32 {
	if g.spawnDist <= 0 {
		return defaultPlaceDist
	}
	return clampf(g.spawnDist, minPlaceDist, maxPlaceDist)
}

// adjustDistance moves the placement/hold point closer or further away. One
// wheel notch is intentionally small so long props can be nudged precisely.
func (g *Game) adjustDistance(steps float32) {
	if steps == 0 {
		return
	}
	if g.spawnDist <= 0 {
		g.spawnDist = defaultPlaceDist
	}
	g.spawnDist = clampf(g.spawnDist+steps*placeDistStep, minPlaceDist, maxPlaceDist)
	if g.held != nil {
		g.holdDist = g.spawnDist
	}
}

// ---------------------------------------------------------------------------
// Camera and cursor
// ---------------------------------------------------------------------------

func (g *Game) updateCamera() {
	p := g.player.body.GetPosition()
	eye := rlv(p.X, p.Y+0.68, p.Z)

	v := g.player.body.GetLinearVelocity()
	speed := vlen(v.X, 0, v.Z)
	if !g.player.fly && g.player.grounded && speed > 0.3 {
		eye.Y += float32(math.Sin(g.time*11)) * 0.018 * clampf(speed/6, 0, 1)
	}
	f := g.cameraForward()
	g.camera.Position = eye
	g.camera.Target = rlv(eye.X+f.X*10, eye.Y+f.Y*10, eye.Z+f.Z*10)
	g.camera.Up = rlv(0, 1, 0)
	fov := g.settings.FOV
	if fov < 30 {
		fov = 72
	}
	g.camera.Fovy = fov
	g.camera.Projection = rl.CameraPerspective
}

func (g *Game) syncCursor() {
	wantLock := g.state == statePlaying
	if wantLock && !g.cursorLocked {
		rl.DisableCursor()
		g.cursorLocked = true
		g.ignoreMouse = true
	} else if !wantLock && g.cursorLocked {
		rl.EnableCursor()
		g.cursorLocked = false
		g.ignoreMouse = false
	}
}

// uiScale keeps the interface readable on large and small windows. It is based
// on the design resolution so panels and text grow or shrink together.
func (g *Game) uiScale() float32 {
	w := float32(rl.GetScreenWidth())
	h := float32(rl.GetScreenHeight())
	s := w / 1280.0
	if h/720.0 < s {
		s = h / 720.0
	}
	if g.settings.UIScale > 0 {
		s *= g.settings.UIScale
	}
	if s < 0.5 {
		s = 0.5
	}
	if s > 2.0 {
		s = 2.0
	}
	return s
}

func (g *Game) uiWidth() float32  { return 1280 }
func (g *Game) uiHeight() float32 { return 720 }

// uiMouse returns the mouse position in the scaled UI coordinate space.
func (g *Game) uiMouse() rl.Vector2 {
	m := rl.GetMousePosition()
	s := g.uiScale()
	return rl.Vector2{X: m.X / s, Y: m.Y / s}
}

func (g *Game) toast(text string, c rl.Color) {
	if len(g.toasts) > 0 {
		t := g.toasts[0]
		if t.Text == text && t.Life > t.MaxLife-0.35 {
			return
		}
	}
	if c.A == 0 {
		c = colText
	}
	g.toasts = append([]*Toast{{
		Text: text, Color: c, Life: 1.8, MaxLife: 1.8,
	}}, g.toasts...)
	if len(g.toasts) > 2 {
		g.toasts = g.toasts[:2]
	}
}

func (g *Game) updateToasts(dt float32) {
	for _, t := range g.toasts {
		t.Life -= dt
	}
	dst := g.toasts[:0]
	for _, t := range g.toasts {
		if t.Life > 0 {
			dst = append(dst, t)
		}
	}
	g.toasts = dst
}

func (g *Game) burst(pos rl.Vector3, c rl.Color, count int, speed, size float32) {
	for i := 0; i < count; i++ {
		dir := normRl(rlv(
			g.rng.Float32()*2-1,
			g.rng.Float32()*1.6-0.3,
			g.rng.Float32()*2-1,
		))
		sp := speed * (0.35 + g.rng.Float32()*0.85)
		life := 0.35 + g.rng.Float32()*0.55
		g.particles = append(g.particles, &Particle{
			Pos:   pos,
			Vel:   rlv(dir.X*sp, dir.Y*sp+2, dir.Z*sp),
			Color: c, Size: size * (0.55 + g.rng.Float32()*0.9),
			Life: life, MaxLife: life, Gravity: 16,
		})
	}
	if len(g.particles) > 500 {
		g.particles = g.particles[len(g.particles)-500:]
	}
}

func (g *Game) updateParticles(dt float32) {
	for _, p := range g.particles {
		p.Life -= dt
		p.Vel.Y -= p.Gravity * dt
		p.Pos.X += p.Vel.X * dt
		p.Pos.Y += p.Vel.Y * dt
		p.Pos.Z += p.Vel.Z * dt
	}
	dst := g.particles[:0]
	for _, p := range g.particles {
		if p.Life > 0 {
			dst = append(dst, p)
		}
	}
	g.particles = dst
}

func (g *Game) updateWaves(dt float32) {
	for _, w := range g.waves {
		w.Life -= dt
		t := 1 - clampf(w.Life/w.MaxLife, 0, 1)
		ease := 1 - (1-t)*(1-t)
		w.Radius = 0.25 + (w.MaxRadius-0.25)*ease
	}
	dst := g.waves[:0]
	for _, w := range g.waves {
		if w.Life > 0 {
			dst = append(dst, w)
		}
	}
	g.waves = dst
}
