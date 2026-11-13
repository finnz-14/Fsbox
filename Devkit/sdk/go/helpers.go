package fsbox

import "encoding/json"

// ---------------------------------------------------------------------------
// Keyboard key constants (raylib key codes)
// Use these instead of raw numbers like 69 for 'E'.
// ---------------------------------------------------------------------------

const (
	KeyNull         = 0
	KeyApostrophe   = 39
	KeyComma        = 44
	KeyMinus        = 45
	KeyPeriod       = 46
	KeySlash        = 47
	Key0            = 48
	Key1            = 49
	Key2            = 50
	Key3            = 51
	Key4            = 52
	Key5            = 53
	Key6            = 54
	Key7            = 55
	Key8            = 56
	Key9            = 57
	KeySemicolon    = 59
	KeyEqual        = 61
	KeyA            = 65
	KeyB            = 66
	KeyC            = 67
	KeyD            = 68
	KeyE            = 69
	KeyF            = 70
	KeyG            = 71
	KeyH            = 72
	KeyI            = 73
	KeyJ            = 74
	KeyK            = 75
	KeyL            = 76
	KeyM            = 77
	KeyN            = 78
	KeyO            = 79
	KeyP            = 80
	KeyQ            = 81
	KeyR            = 82
	KeyS            = 83
	KeyT            = 84
	KeyU            = 85
	KeyV            = 86
	KeyW            = 87
	KeyX            = 88
	KeyY            = 89
	KeyZ            = 90
	KeyLeftBracket  = 91
	KeyBackslash    = 92
	KeyRightBracket = 93
	KeyGrave        = 96
	KeySpace        = 32
	KeyEscape       = 256
	KeyEnter        = 257
	KeyTab          = 258
	KeyBackspace    = 259
	KeyInsert       = 260
	KeyDelete       = 261
	KeyRight        = 262
	KeyLeft         = 263
	KeyDown         = 264
	KeyUp           = 265
	KeyPageUp       = 266
	KeyPageDown     = 267
	KeyHome         = 268
	KeyEnd          = 269
	KeyCapsLock     = 280
	KeyScrollLock   = 281
	KeyNumLock      = 282
	KeyPrintScreen  = 283
	KeyPause        = 284
	KeyF1           = 290
	KeyF2           = 291
	KeyF3           = 292
	KeyF4           = 293
	KeyF5           = 294
	KeyF6           = 295
	KeyF7           = 296
	KeyF8           = 297
	KeyF9           = 298
	KeyF10          = 299
	KeyF11          = 300
	KeyF12          = 301
	KeyLeftShift    = 340
	KeyLeftControl  = 341
	KeyLeftAlt      = 342
	KeyLeftSuper    = 343
	KeyRightShift   = 344
	KeyRightControl = 345
	KeyRightAlt     = 346
	KeyRightSuper   = 347
	KeyMenu         = 348
)

// ---------------------------------------------------------------------------
// Mouse button constants
// ---------------------------------------------------------------------------

const (
	MouseButtonLeft    = 0
	MouseButtonRight   = 1
	MouseButtonMiddle  = 2
	MouseButtonSide    = 3
	MouseButtonExtra   = 4
	MouseButtonForward = 5
	MouseButtonBack    = 6
)

// ---------------------------------------------------------------------------
// Config (root-level persistent file I/O)
// ---------------------------------------------------------------------------

func ConfigRead(path string) (string, error) {
	raw, err := Call("config.read_text", map[string]any{"path": path})
	if err != nil {
		return "", err
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s, nil
}

func ConfigWrite(path, content string) error {
	_, err := Call("config.write_text", map[string]any{"path": path, "content": content})
	return err
}

func ConfigAppend(path, content string) error {
	_, err := Call("config.write_text", map[string]any{"path": path, "content": content, "append": true})
	return err
}

func ConfigExists(path string) (bool, error) {
	raw, err := Call("config.exists", map[string]any{"path": path})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

func ConfigDelete(path string) error {
	_, err := Call("config.delete", map[string]any{"path": path})
	return err
}

func ConfigList(path string) ([]string, error) {
	raw, err := Call("config.list", map[string]any{"path": path})
	if err != nil {
		return nil, err
	}
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func ConfigMkdir(path string) error {
	_, err := Call("config.mkdir", map[string]any{"path": path})
	return err
}

// ---------------------------------------------------------------------------
// 3D models
// ---------------------------------------------------------------------------

func LoadModel(path string) (int32, error) {
	raw, err := Call("model.load", map[string]any{"path": path})
	if err != nil {
		return 0, err
	}
	var h int32
	_ = json.Unmarshal(raw, &h)
	return h, nil
}

func UnloadModel(h int32) error {
	_, err := Call("model.unload", map[string]any{"model": h})
	return err
}

func ListModelFiles(dir string) ([]string, error) {
	raw, err := Call("model.list_files", map[string]any{"path": dir})
	if err != nil {
		return nil, err
	}
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func SpawnModelEntity(model int32, pos Vec3, color Color) (int32, error) {
	raw, err := Call("entity.spawn", map[string]any{"model": model, "position": pos, "color": color})
	if err != nil {
		return 0, err
	}
	var e struct {
		ID int32 `json:"id"`
	}
	_ = json.Unmarshal(raw, &e)
	return e.ID, nil
}

// SpawnPreset spawns a named content preset (the same list the spawn menu
// shows) and returns the new entity ID.
func SpawnPreset(name string, pos Vec3, color Color) (int32, error) {
	raw, err := Call("entity.spawn", map[string]any{
		"preset_name": name, "position": pos, "color": color,
	})
	if err != nil {
		return 0, err
	}
	var e struct {
		ID int32 `json:"id"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return 0, err
	}
	return e.ID, nil
}

func SetEntityModel(id, model int32) error {
	_, err := Call("entity.set_model", map[string]any{"id": id, "model": model})
	return err
}

// ---------------------------------------------------------------------------
// Shaders
// ---------------------------------------------------------------------------

func LoadShader(vertex, fragment string) (int32, error) {
	raw, err := Call("shader.load", map[string]any{"vertex": vertex, "fragment": fragment})
	if err != nil {
		return 0, err
	}
	var h int32
	_ = json.Unmarshal(raw, &h)
	return h, nil
}

func UnloadShader(h int32) error {
	_, err := Call("shader.unload", map[string]any{"shader": h})
	return err
}

// ---------------------------------------------------------------------------
// Interactive UI elements
// ---------------------------------------------------------------------------

type UIElement struct {
	ID    int32
	Kind  string
	X, Y  float32
	W, H  float32
	Label string
	Color Color
	Value float32
	Min   float32
	Max   float32
}

func AddButton(x, y, w, h float32, label string, color Color) (int32, error) {
	raw, err := Call("ui.add_element", map[string]any{
		"kind": "button", "x": x, "y": y, "w": w, "h": h, "label": label, "color": color,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

func AddPanel(x, y, w, h float32, color Color) (int32, error) {
	raw, err := Call("ui.add_element", map[string]any{
		"kind": "panel", "x": x, "y": y, "w": w, "h": h, "color": color,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

func AddSlider(x, y, w, h float32, min, max, value float32, color Color) (int32, error) {
	raw, err := Call("ui.add_element", map[string]any{
		"kind": "slider", "x": x, "y": y, "w": w, "h": h, "min": min, "max": max, "value": value, "color": color,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

func RemoveUIElement(id int32) error {
	_, err := Call("ui.remove_element", map[string]any{"id": id})
	return err
}

func SetUIElement(id int32, x, y, w, h *float32, label *string, color *Color, value *float32) error {
	args := map[string]any{"id": id}
	if x != nil {
		args["x"] = *x
	}
	if y != nil {
		args["y"] = *y
	}
	if w != nil {
		args["w"] = *w
	}
	if h != nil {
		args["h"] = *h
	}
	if label != nil {
		args["label"] = *label
	}
	if color != nil {
		args["color"] = *color
	}
	if value != nil {
		args["value"] = *value
	}
	_, err := Call("ui.set_element", args)
	return err
}

// ---------------------------------------------------------------------------
// 3D Billboard panels
// ---------------------------------------------------------------------------

func AddBillboardPanel(x, y, z, width, height float32) (int32, error) {
	raw, err := Call("ui.billboard_add", map[string]any{
		"x": x, "y": y, "z": z,
		"face_x": 0, "face_y": 0, "face_z": 0,
		"right_x": 0, "right_y": 0, "right_z": 0,
		"up_x": 0, "up_y": 0, "up_z": 0,
		"width": width, "height": height,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

func AddBillboardButton(panelID int32, x, y, w, h float32, label string, color Color) (int32, error) {
	raw, err := Call("ui.billboard_button", map[string]any{
		"panel": panelID, "x": x, "y": y, "w": w, "h": h,
		"label": label, "color": color,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

// AddBillboardButtonColored is AddBillboardButton with an explicit label
// colour. Handy for light buttons on a dark panel.
func AddBillboardButtonColored(panelID int32, x, y, w, h float32, label string, background, text Color) (int32, error) {
	return AddBillboardButtonStyled(panelID, x, y, w, h, label, background, text, 0)
}

// AddBillboardButtonStyled also sets the label's screen-space pixel size. A
// size of 0 uses the engine default (15px). Use AddBillboardButtonWorld when
// the label should scale with distance instead.
func AddBillboardButtonStyled(panelID int32, x, y, w, h float32, label string, background, text Color, textSize float32) (int32, error) {
	raw, err := Call("ui.billboard_button", map[string]any{
		"panel": panelID, "x": x, "y": y, "w": w, "h": h,
		"label": label, "color": background, "text_color": text, "size": textSize,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

// AddBillboardButtonWorld is AddBillboardButtonStyled with a world-space label
// height. The text grows and shrinks with the panel as the player moves.
func AddBillboardButtonWorld(panelID int32, x, y, w, h float32, label string, background, text Color, worldHeight float32) (int32, error) {
	raw, err := Call("ui.billboard_button", map[string]any{
		"panel": panelID, "x": x, "y": y, "w": w, "h": h,
		"label": label, "color": background, "text_color": text,
		"size": worldHeight, "world": true,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

func AddBillboardText(panelID int32, x, y float32, label string, size float32, color Color) (int32, error) {
	raw, err := Call("ui.billboard_text", map[string]any{
		"panel": panelID, "x": x, "y": y,
		"label": label, "size": size, "color": color,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

// AddBillboardTextWorld creates text whose size is a world-space height, so it
// scales with the panel as the player moves closer or farther away.
func AddBillboardTextWorld(panelID int32, x, y float32, label string, worldHeight float32, color Color) (int32, error) {
	raw, err := Call("ui.billboard_text", map[string]any{
		"panel": panelID, "x": x, "y": y,
		"label": label, "size": worldHeight, "color": color, "world": true,
	})
	if err != nil {
		return 0, err
	}
	var id int32
	_ = json.Unmarshal(raw, &id)
	return id, nil
}

func RemoveBillboardPanel(panelID int32) error {
	_, err := Call("ui.billboard_remove", map[string]any{"panel": panelID})
	return err
}

// SetBillboardTools restricts which tools may click a panel. Pass nil or an
// empty slice to allow every tool. Tool IDs are 0.. for built-in tools and
// 1000+ for mod-registered tools (see ToolInfo is not needed; use the ID
// returned by RegisterTool plus 1000).
func SetBillboardTools(panelID int32, tools []string) error {
	return callInto("ui.billboard_set_tools", map[string]any{"panel": panelID, "tools": tools}, nil)
}

// MoveBillboardPanel repositions an existing panel in world space. Panels
// always face the camera, so only the centre position is needed.
func MoveBillboardPanel(panelID int32, x, y, z float32) error {
	_, err := Call("ui.billboard_move", map[string]any{
		"panel": panelID, "x": x, "y": y, "z": z,
	})
	return err
}

func ClearBillboardPanels() error {
	_, err := Call("ui.billboard_clear", nil)
	return err
}

// BillboardElementUpdate describes a partial change to a billboard button or
// text. Only the non-nil fields are sent, so a zero value cannot accidentally
// overwrite something.
type BillboardElementUpdate struct {
	X         *float32
	Y         *float32
	W         *float32
	H         *float32
	Label     *string
	Color     *Color
	TextColor *Color
	TextSize  *float32
	// WorldSize switches the label size between screen pixels (false) and
	// world-space height (true).
	WorldSize *bool
}

// UpdateBillboardElement changes an existing button/text in place. Reusing
// elements (instead of rebuilding the panel) keeps their IDs stable, so click
// events continue to point at the right row.
func UpdateBillboardElement(elementID int32, u BillboardElementUpdate) error {
	args := map[string]any{"element": elementID}
	if u.X != nil {
		args["x"] = *u.X
	}
	if u.Y != nil {
		args["y"] = *u.Y
	}
	if u.W != nil {
		args["w"] = *u.W
	}
	if u.H != nil {
		args["h"] = *u.H
	}
	if u.Label != nil {
		args["label"] = *u.Label
	}
	if u.Color != nil {
		args["color"] = *u.Color
	}
	if u.TextColor != nil {
		args["text_color"] = *u.TextColor
	}
	if u.TextSize != nil {
		args["size"] = *u.TextSize
	}
	if u.WorldSize != nil {
		args["world"] = *u.WorldSize
	}
	_, err := Call("ui.billboard_update", args)
	return err
}

// BillboardHover describes which billboard the crosshair is currently over.
// Panel is 0 when no billboard is under the crosshair.
type BillboardHover struct {
	Panel   int32 `json:"panel"`
	Element int32 `json:"element"`
}

// GetBillboardHover returns the billboard panel and button under the crosshair.
// Mods use it to decide whether wheel/key input should affect their panel.
func GetBillboardHover() (BillboardHover, error) {
	raw, err := Call("ui.billboard_hover", nil)
	if err != nil {
		return BillboardHover{}, err
	}
	var h BillboardHover
	if err := json.Unmarshal(raw, &h); err != nil {
		return BillboardHover{}, err
	}
	return h, nil
}

// ---------------------------------------------------------------------------
// Audio
// ---------------------------------------------------------------------------

func LoadSound(name, path string) error {
	_, err := Call("audio.load_sound", map[string]any{"name": name, "path": path})
	return err
}

func PlaySound(name string) error {
	_, err := Call("audio.play_sound", map[string]any{"name": name})
	return err
}

func PlaySound3D(name string, pos Vec3, volume float32, maxDistance float32) error {
	_, err := Call("audio.play_sound_3d", map[string]any{"name": name, "position": pos, "volume": volume, "max_distance": maxDistance})
	return err
}

func StopSound(name string) error {
	_, err := Call("audio.stop_sound", map[string]any{"name": name})
	return err
}

func LoadMusic(name, path string) error {
	_, err := Call("audio.load_music", map[string]any{"name": name, "path": path})
	return err
}

func PlayMusic(name string, volume float32) error {
	_, err := Call("audio.play_music", map[string]any{"name": name, "volume": volume})
	return err
}

func StopMusic(name string) error {
	_, err := Call("audio.stop_music", map[string]any{"name": name})
	return err
}

// UnloadAudio frees a sound or music stream previously loaded under name.
func UnloadAudio(name string) error {
	return callInto("audio.unload", map[string]any{"name": name}, nil)
}

func SetMusicVolume(name string, volume float32) error {
	_, err := Call("audio.set_music_volume", map[string]any{"name": name, "volume": volume})
	return err
}

func AudioReady() (bool, error) {
	raw, err := Call("audio.is_ready", nil)
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

// ---------------------------------------------------------------------------
// Keyboard input
// ---------------------------------------------------------------------------

func InputKeyPressed(key int32) (bool, error) {
	raw, err := Call("input.key_pressed", map[string]any{"key": key})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

func InputKeyDown(key int32) (bool, error) {
	raw, err := Call("input.key_down", map[string]any{"key": key})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

func InputKeyReleased(key int32) (bool, error) {
	raw, err := Call("input.key_released", map[string]any{"key": key})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

// ---------------------------------------------------------------------------
// Mouse input
// ---------------------------------------------------------------------------

func InputMouseButtonPressed(button int32) (bool, error) {
	raw, err := Call("input.mouse_pressed", map[string]any{"button": button})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

func InputMouseButtonDown(button int32) (bool, error) {
	raw, err := Call("input.mouse_down", map[string]any{"button": button})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

func InputMouseButtonReleased(button int32) (bool, error) {
	raw, err := Call("input.mouse_released", map[string]any{"button": button})
	if err != nil {
		return false, err
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b, nil
}

// InputWheel returns the mouse wheel scroll amount.
func InputWheel() (float32, error) {
	raw, err := Call("input.wheel", nil)
	if err != nil {
		return 0, err
	}
	var v float32
	_ = json.Unmarshal(raw, &v)
	return v, nil
}

// ---------------------------------------------------------------------------
// Player info (for mods to detect proximity themselves)
// ---------------------------------------------------------------------------

type AnimationInfo struct {
	Index     int32   `json:"index"`
	Name      string  `json:"name"`
	Keyframes int32   `json:"keyframes"`
	Bones     int32   `json:"bones"`
	Current   bool    `json:"current"`
	Playing   bool    `json:"playing"`
	Frame     float32 `json:"frame"`
	Speed     float32 `json:"speed"`
}

type EntityInfo struct {
	ID           int32   `json:"id"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	Static       bool    `json:"static"`
	Frozen       bool    `json:"frozen"`
	Invulnerable bool    `json:"invulnerable"`
	Transient    bool    `json:"transient"`
	Health       float32 `json:"health"`
	MaxHealth    float32 `json:"max_health"`
	Position     Vec3    `json:"position"`
	Rotation     Quat    `json:"rotation"`
	Velocity     Vec3    `json:"velocity"`
	AngularVel   Vec3    `json:"angular_velocity"`
	Mass         float32 `json:"mass"`
	Color        Color   `json:"color"`
	Half         Vec3    `json:"half"`
	Radius       float32 `json:"radius"`
	Height       float32 `json:"height"`
	Density      float32 `json:"density"`
	Friction     float32 `json:"friction"`
	Restitution  float32 `json:"restitution"`
	Owner        string  `json:"owner"`
	HasModel     bool    `json:"has_model"`
}

type PlayerInfo struct {
	Position   Vec3    `json:"position"`
	Rotation   Quat    `json:"rotation"`
	Velocity   Vec3    `json:"velocity"`
	Yaw        float32 `json:"yaw"`
	Pitch      float32 `json:"pitch"`
	Fly        bool    `json:"fly"`
	Grounded   bool    `json:"grounded"`
	Health     float32 `json:"health"`
	MaxHealth  float32 `json:"max_health"`
	SpeedScale float32 `json:"speed_scale"`
	JumpScale  float32 `json:"jump_scale"`
	Eye        Vec3    `json:"eye"`
	Forward    Vec3    `json:"forward"`
	Color      Color   `json:"color"`
}

// EngineInfo is a small snapshot of the engine state. State is one of
// "menu", "map_menu", "playing", "paused", "help", "settings", "spawn" or
// "tools".
type EngineInfo struct {
	Game        string     `json:"game"`
	Version     string     `json:"version"`
	APIVersion  string     `json:"api_version"`
	State       string     `json:"state"`
	Time        float64    `json:"time"`
	Frame       int64      `json:"frame"`
	FPS         int32      `json:"fps"`
	TimeScale   float64    `json:"time_scale"`
	Paused      bool       `json:"paused"`
	EntityCount int        `json:"entity_count"`
	PropCount   int        `json:"prop_count"`
	Player      PlayerInfo `json:"player"`
}

func GetEngineInfo() (EngineInfo, error) {
	raw, err := Call("engine.info", nil)
	if err != nil {
		return EngineInfo{}, err
	}
	var info EngineInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return EngineInfo{}, err
	}
	return info, nil
}

func GetPlayerInfo() (PlayerInfo, error) {
	raw, err := Call("player.get", nil)
	if err != nil {
		return PlayerInfo{}, err
	}
	var pi PlayerInfo
	if err := json.Unmarshal(raw, &pi); err != nil {
		return PlayerInfo{}, err
	}
	return pi, nil
}

func GetEntity(id int32) (EntityInfo, error) {
	raw, err := Call("entity.get", map[string]any{"id": id})
	if err != nil {
		return EntityInfo{}, err
	}
	var e EntityInfo
	if err := json.Unmarshal(raw, &e); err != nil {
		return EntityInfo{}, err
	}
	return e, nil
}

// GetEntities returns every entity in the world. Filter with the static/props
// booleans to narrow the list.
func GetEntities(static bool, props bool) ([]EntityInfo, error) {
	raw, err := Call("entity.list", map[string]any{"static": static, "props": props, "transient": false})
	if err != nil {
		return nil, err
	}
	var out []EntityInfo
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func GetAnimations(id int32) ([]AnimationInfo, error) {
	raw, err := Call("entity.get_animations", map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	var out []AnimationInfo
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func PlayAnimationByIndex(id, animIndex int32, speed float32) error {
	_, err := Call("entity.play_animation", map[string]any{"id": id, "anim_index": animIndex, "speed": speed})
	return err
}

func StopAnimation(id int32) error {
	_, err := Call("entity.stop_animation", map[string]any{"id": id})
	return err
}

func PauseAnimation(id int32) error {
	_, err := Call("entity.pause_animation", map[string]any{"id": id})
	return err
}

func ResumeAnimation(id int32) error {
	_, err := Call("entity.resume_animation", map[string]any{"id": id})
	return err
}

func SetAnimationSpeed(id int32, speed float32) error {
	_, err := Call("entity.set_animation_speed", map[string]any{"id": id, "speed": speed})
	return err
}

func SetAnimationFrame(id int32, frame float32) error {
	_, err := Call("entity.set_animation_frame", map[string]any{"id": id, "frame": frame})
	return err
}

func SetGravityScale(id int32, scale float32) error {
	_, err := Call("entity.set_gravity_scale", map[string]any{"id": id, "scale": scale})
	return err
}

func SetAngularVelocity(id int32, v Vec3) error {
	_, err := Call("entity.set_angular_velocity", map[string]any{"id": id, "velocity": v})
	return err
}

func SetMotionLocks(id int32, angularX, angularY, angularZ bool) error {
	_, err := Call("entity.set_motion_locks", map[string]any{
		"id":        id,
		"angular_x": angularX,
		"angular_y": angularY,
		"angular_z": angularZ,
	})
	return err
}

type MotionLocks struct {
	LinearX  bool `json:"linear_x"`
	LinearY  bool `json:"linear_y"`
	LinearZ  bool `json:"linear_z"`
	AngularX bool `json:"angular_x"`
	AngularY bool `json:"angular_y"`
	AngularZ bool `json:"angular_z"`
}

func GetMotionLocks(id int32) (MotionLocks, error) {
	raw, err := Call("entity.get_motion_locks", map[string]any{"id": id})
	if err != nil {
		return MotionLocks{}, err
	}
	var locks MotionLocks
	_ = json.Unmarshal(raw, &locks)
	return locks, nil
}

// ---------------------------------------------------------------------------
// Inter-mod communication
// ---------------------------------------------------------------------------

// Broadcast sends data on a named channel to all other mods. Receivers listen
// with On("net.<channel>", handler).
func Broadcast(channel string, data any) error {
	_, err := Call("net.broadcast", map[string]any{"channel": channel, "data": data})
	return err
}

// FireEvent broadcasts an event to all mods. Receivers listen with
// On(event, handler) or registered hooks.
func FireEvent(event string, data any) error {
	_, err := Call("hook.fire", map[string]any{"event": event, "data": data})
	return err
}

// AddHook registers an exported function as a hook for an event. When the
// event fires, the engine calls the named exported function.
func AddHook(event, hook string) (bool, error) {
	raw, err := Call("hook.add", map[string]any{"event": event, "hook": hook})
	if err != nil {
		return false, err
	}
	var ok bool
	_ = json.Unmarshal(raw, &ok)
	return ok, nil
}

// RemoveHook unregisters a previously added hook.
func RemoveHook(event, hook string) (bool, error) {
	raw, err := Call("hook.remove", map[string]any{"event": event, "hook": hook})
	if err != nil {
		return false, err
	}
	var ok bool
	_ = json.Unmarshal(raw, &ok)
	return ok, nil
}

// ListHooks returns the hooks registered by this mod.
func ListHooks() (map[string][]string, error) {
	raw, err := Call("hook.list", nil)
	if err != nil {
		return nil, err
	}
	var out map[string][]string
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

// ---------------------------------------------------------------------------
// Shared value types
// ---------------------------------------------------------------------------

// Quat is a rotation quaternion in (x, y, z, w) order.
type Quat struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	W float32 `json:"w"`
}

// Vec2 is a two-component vector, used for screen-space coordinates.
type Vec2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// Ray is a world-space ray.
type Ray struct {
	Origin    Vec3 `json:"origin"`
	Direction Vec3 `json:"direction"`
}

// callInto runs one command and decodes its result into out. A null or empty
// result leaves out untouched, which lets void commands share one helper.
func callInto(cmd string, args any, out any) error {
	raw, err := Call(cmd, args)
	if err != nil {
		return err
	}
	if out == nil || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// ---------------------------------------------------------------------------
// Engine control
// ---------------------------------------------------------------------------

// GetAPIVersion returns the mod API version implemented by the running game.
func GetAPIVersion() (string, error) {
	var v string
	err := callInto("engine.api_version", nil, &v)
	return v, err
}

// SetTimeScale changes simulation speed. 1.0 is real time, 0 pauses the
// simulation without opening the pause menu. Returns the applied scale.
func SetTimeScale(scale float32) (float64, error) {
	var v float64
	err := callInto("engine.set_time_scale", map[string]any{"scale": scale}, &v)
	return v, err
}

// GetTime returns the number of simulated seconds since the world started.
func GetTime() (float64, error) {
	var v float64
	err := callInto("engine.get_time", nil, &v)
	return v, err
}

// SetPaused opens or closes the pause menu.
func SetPaused(paused bool) error {
	return callInto("engine.set_paused", map[string]any{"paused": paused}, nil)
}

// ResetWorld rebuilds the current map. withStarter controls whether the
// default prop layout is spawned.
func ResetWorld(withStarter bool) error {
	return callInto("engine.reset_world", map[string]any{"with_starter": withStarter}, nil)
}

// SaveGame writes the current world to the save slot.
func SaveGame() error { return callInto("engine.save", nil, nil) }

// LoadGame restores the world from the save slot.
func LoadGame() error { return callInto("engine.load", nil, nil) }

// Quit closes the game after the current frame.
func Quit() error { return callInto("engine.quit", nil, nil) }

// Screenshot captures the framebuffer to path (relative to the game root).
// Returns the path that was written.
func Screenshot(path string) (string, error) {
	var out string
	err := callInto("engine.screenshot", map[string]any{"path": path}, &out)
	return out, err
}

// BindKey fires event whenever key is pressed. Use the Key* constants.
func BindKey(key int32, event string) error {
	return callInto("engine.bind_key", map[string]any{"key": key, "event": event}, nil)
}

// UnbindKey removes a key binding added with BindKey.
func UnbindKey(key int32, event string) error {
	return callInto("engine.unbind_key", map[string]any{"key": key, "event": event}, nil)
}

// BindMouse fires event whenever button is pressed. Use MouseButton* constants.
func BindMouse(button int32, event string) error {
	return callInto("engine.bind_mouse", map[string]any{"button": button, "event": event}, nil)
}

// UnbindMouse removes a mouse binding added with BindMouse.
func UnbindMouse(button int32, event string) error {
	return callInto("engine.unbind_mouse", map[string]any{"button": button, "event": event}, nil)
}

// SetHUDHook registers an exported function called during the HUD pass.
func SetHUDHook(hook string) error {
	return callInto("engine.set_hud_hook", map[string]any{"hook": hook}, nil)
}

// HasPermission reports whether the mod's manifest granted permission.
func HasPermission(permission string) (bool, error) {
	var ok bool
	err := callInto("engine.has_permission", map[string]any{"permission": permission}, &ok)
	return ok, err
}

// ---------------------------------------------------------------------------
// Player control
// ---------------------------------------------------------------------------

func SetPlayerPos(pos Vec3) error {
	return callInto("player.set_pos", map[string]any{"position": pos}, nil)
}

func SetPlayerRot(yaw, pitch float32) error {
	return callInto("player.set_rot", map[string]any{"yaw": yaw, "pitch": pitch}, nil)
}

func SetPlayerVelocity(v Vec3) error {
	return callInto("player.set_velocity", map[string]any{"velocity": v}, nil)
}

func SetPlayerFly(fly bool) error {
	return callInto("player.set_fly", map[string]any{"fly": fly}, nil)
}

func SetPlayerSpeedScale(scale float32) (float32, error) {
	var v float32
	err := callInto("player.set_speed_scale", map[string]any{"scale": scale}, &v)
	return v, err
}

func SetPlayerJumpScale(scale float32) (float32, error) {
	var v float32
	err := callInto("player.set_jump_scale", map[string]any{"scale": scale}, &v)
	return v, err
}

func SetPlayerHealth(health, maxHealth float32) (float32, error) {
	var v float32
	err := callInto("player.set_health", map[string]any{"health": health, "max_health": maxHealth}, &v)
	return v, err
}

func DamagePlayer(amount float32) (float32, error) {
	var v float32
	err := callInto("player.damage", map[string]any{"amount": amount}, &v)
	return v, err
}

func HealPlayer(amount float32) (float32, error) {
	var v float32
	err := callInto("player.heal", map[string]any{"amount": amount}, &v)
	return v, err
}

func RespawnPlayer() error { return callInto("player.respawn", nil, nil) }

func SetPlayerGravityScale(scale float32) error {
	return callInto("player.set_gravity_scale", map[string]any{"scale": scale}, nil)
}

// SetPlayerModel shows a loaded model in third person.
func SetPlayerModel(model int32) error {
	return callInto("player.set_model", map[string]any{"model": model}, nil)
}

func ClearPlayerModel() error { return callInto("player.clear_model", nil, nil) }

// ---------------------------------------------------------------------------
// Camera
// ---------------------------------------------------------------------------

type CameraInfo struct {
	Position Vec3    `json:"position"`
	Target   Vec3    `json:"target"`
	Up       Vec3    `json:"up"`
	FOV      float32 `json:"fov"`
	Mode     int32   `json:"mode"`
	Distance float32 `json:"distance"`
}

func GetCamera() (CameraInfo, error) {
	var c CameraInfo
	err := callInto("camera.get", nil, &c)
	return c, err
}

// SetCameraMode selects first person (0) or third person (1).
func SetCameraMode(mode int32) (int32, error) {
	var v int32
	err := callInto("camera.set_mode", map[string]any{"mode": mode}, &v)
	return v, err
}

func SetCameraDistance(distance float32) (float32, error) {
	var v float32
	err := callInto("camera.set_distance", map[string]any{"distance": distance}, &v)
	return v, err
}

func SetCameraOffset(offset Vec3) (Vec3, error) {
	var v Vec3
	err := callInto("camera.set_offset", map[string]any{"offset": offset}, &v)
	return v, err
}

func ShakeCamera(amount float32) error {
	return callInto("camera.shake", map[string]any{"amount": amount}, nil)
}

// ---------------------------------------------------------------------------
// Physics
// ---------------------------------------------------------------------------

type PhysicsStats struct {
	AwakeBodies int32 `json:"awake_bodies"`
	Bodies      int32 `json:"bodies"`
	Props       int32 `json:"props"`
	BoundsMin   Vec3  `json:"bounds_min"`
	BoundsMax   Vec3  `json:"bounds_max"`
}

// RaycastInfo is the result of a physics ray query.
type RaycastInfo struct {
	Hit      bool        `json:"hit"`
	EntityID int32       `json:"entity_id"`
	Entity   *EntityInfo `json:"entity,omitempty"`
	Point    Vec3        `json:"point"`
	Normal   Vec3        `json:"normal"`
	Fraction float32     `json:"fraction"`
	Distance float32     `json:"distance"`
}

type ConstraintInfo struct {
	ID      int32  `json:"id"`
	Type    string `json:"type"`
	EntityA int32  `json:"entity_a"`
	EntityB int32  `json:"entity_b"`
	Owner   string `json:"owner"`
}

// ConstraintDesc describes a joint between two entities. Type is one of
// "weld", "distance", "revolute", "prismatic", "spherical", "parallel",
// "motor" or "wheel". Zero values use the engine defaults.
type ConstraintDesc struct {
	Type    string `json:"type"`
	A       int32  `json:"a"`
	B       int32  `json:"b"`
	Anchor  *Vec3  `json:"anchor,omitempty"`
	Axis    *Vec3  `json:"axis,omitempty"`
	Collide *bool  `json:"collide,omitempty"`

	Length        float32 `json:"length,omitempty"`
	EnableSpring  bool    `json:"enable_spring,omitempty"`
	Hertz         float32 `json:"hertz,omitempty"`
	DampingRatio  float32 `json:"damping_ratio,omitempty"`
	MinLength     float32 `json:"min_length,omitempty"`
	MaxLength     float32 `json:"max_length,omitempty"`
	EnableLimit   bool    `json:"enable_limit,omitempty"`
	EnableMotor   bool    `json:"enable_motor,omitempty"`
	MaxMotorForce float32 `json:"max_motor_force,omitempty"`
	MotorSpeed    float32 `json:"motor_speed,omitempty"`

	TargetAngle    float32 `json:"target_angle,omitempty"`
	LowerAngle     float32 `json:"lower_angle,omitempty"`
	UpperAngle     float32 `json:"upper_angle,omitempty"`
	MaxMotorTorque float32 `json:"max_motor_torque,omitempty"`

	TargetTranslation float32 `json:"target_translation,omitempty"`
	LowerTranslation  float32 `json:"lower_translation,omitempty"`
	UpperTranslation  float32 `json:"upper_translation,omitempty"`

	ConeAngle       float32 `json:"cone_angle,omitempty"`
	LowerTwistAngle float32 `json:"lower_twist_angle,omitempty"`
	UpperTwistAngle float32 `json:"upper_twist_angle,omitempty"`

	LinearHertz         float32 `json:"linear_hertz,omitempty"`
	AngularHertz        float32 `json:"angular_hertz,omitempty"`
	LinearDampingRatio  float32 `json:"linear_damping_ratio,omitempty"`
	AngularDampingRatio float32 `json:"angular_damping_ratio,omitempty"`
	MaxTorque           float32 `json:"max_torque,omitempty"`

	SuspensionHertz        float32 `json:"suspension_hertz,omitempty"`
	SuspensionDampingRatio float32 `json:"suspension_damping_ratio,omitempty"`
	LowerSuspensionLimit   float32 `json:"lower_suspension_limit,omitempty"`
	UpperSuspensionLimit   float32 `json:"upper_suspension_limit,omitempty"`
	SpinSpeed              float32 `json:"spin_speed,omitempty"`
	MaxSpinTorque          float32 `json:"max_spin_torque,omitempty"`
	SteeringHertz          float32 `json:"steering_hertz,omitempty"`
	SteeringDampingRatio   float32 `json:"steering_damping_ratio,omitempty"`
	TargetSteeringAngle    float32 `json:"target_steering_angle,omitempty"`
	EnableSteering         bool    `json:"enable_steering,omitempty"`
}

func SetWorldGravityScale(scale float32) (Vec3, error) {
	var v Vec3
	err := callInto("world.set_gravity", map[string]any{"scale": scale}, &v)
	return v, err
}

func SetWorldGravityY(y float32) (Vec3, error) {
	var v Vec3
	err := callInto("world.set_gravity", map[string]any{"y": y}, &v)
	return v, err
}

func GetGravity() (Vec3, error) {
	var v Vec3
	err := callInto("physics.get_gravity", nil, &v)
	return v, err
}

func Raycast(origin, direction Vec3, maxDistance float32, mask *uint64) (RaycastInfo, error) {
	args := map[string]any{"origin": origin, "direction": direction, "max_distance": maxDistance}
	if mask != nil {
		args["mask"] = *mask
	}
	var out RaycastInfo
	err := callInto("physics.raycast", args, &out)
	return out, err
}

func RaycastCamera(maxDistance float32, mask *uint64) (RaycastInfo, error) {
	args := map[string]any{"max_distance": maxDistance}
	if mask != nil {
		args["mask"] = *mask
	}
	var out RaycastInfo
	err := callInto("physics.raycast_camera", args, &out)
	return out, err
}

func OverlapSphere(center Vec3, radius float32, mask *uint64) ([]EntityInfo, error) {
	args := map[string]any{"center": center, "radius": radius}
	if mask != nil {
		args["mask"] = *mask
	}
	var out []EntityInfo
	err := callInto("physics.overlap_sphere", args, &out)
	return out, err
}

func Explode(center Vec3, radius, falloff, impulsePerArea float32, mask *uint64) error {
	args := map[string]any{
		"center": center, "radius": radius, "falloff": falloff,
		"impulse_per_area": impulsePerArea,
	}
	if mask != nil {
		args["mask"] = *mask
	}
	return callInto("physics.explode", args, nil)
}

func GetPhysicsStats() (PhysicsStats, error) {
	var out PhysicsStats
	err := callInto("physics.stats", nil, &out)
	return out, err
}

func SetWorkerCount(count int32) error {
	return callInto("physics.set_worker_count", map[string]any{"count": count}, nil)
}

func CreateConstraint(desc ConstraintDesc) (int32, error) {
	var id int32
	err := callInto("physics.create_constraint", desc, &id)
	return id, err
}

func RemoveConstraint(id int32) (bool, error) {
	var ok bool
	err := callInto("physics.remove_constraint", map[string]any{"id": id}, &ok)
	return ok, err
}

func ListConstraints() ([]ConstraintInfo, error) {
	var out []ConstraintInfo
	err := callInto("physics.list_constraints", nil, &out)
	return out, err
}

// ---------------------------------------------------------------------------
// Particles
// ---------------------------------------------------------------------------

func SpawnParticle(pos, velocity Vec3, color Color, size, life, gravity float32) (int32, error) {
	var n int32
	err := callInto("particle.spawn", map[string]any{
		"position": pos, "velocity": velocity, "color": color,
		"size": size, "life": life, "gravity": gravity,
	}, &n)
	return n, err
}

func ParticleBurst(pos Vec3, color Color, count int32, speed, life, size, gravity float32) (int32, error) {
	var n int32
	err := callInto("particle.burst", map[string]any{
		"position": pos, "color": color, "count": count,
		"speed": speed, "life": life, "size": size, "gravity": gravity,
	}, &n)
	return n, err
}

func ClearParticles() error { return callInto("particle.clear", nil, nil) }

func ParticleCount() (int, error) {
	var n int
	err := callInto("particle.count", nil, &n)
	return n, err
}

// ---------------------------------------------------------------------------
// Weapons
// ---------------------------------------------------------------------------

type WeaponInfo struct {
	ID              int32   `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Damage          float32 `json:"damage"`
	Range           float32 `json:"range"`
	Ammo            int32   `json:"ammo"`
	MaxAmmo         int32   `json:"max_ammo"`
	FireRate        float32 `json:"fire_rate"`
	Recoil          float32 `json:"recoil"`
	Spread          float32 `json:"spread"`
	ProjectileSpeed float32 `json:"projectile_speed"`
	ProjectileLife  float32 `json:"projectile_life"`
	ProjectileSize  float32 `json:"projectile_size"`
	Color           Color   `json:"color"`
	Owner           string  `json:"owner"`
}

type WeaponDesc struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Damage          float32 `json:"damage"`
	Range           float32 `json:"range"`
	Ammo            int32   `json:"ammo"`
	MaxAmmo         int32   `json:"max_ammo"`
	FireRate        float32 `json:"fire_rate"`
	Recoil          float32 `json:"recoil"`
	Spread          float32 `json:"spread"`
	ProjectileSpeed float32 `json:"projectile_speed"`
	ProjectileLife  float32 `json:"projectile_life"`
	ProjectileSize  float32 `json:"projectile_size"`
	Color           Color   `json:"color"`
}

type WeaponFireResult struct {
	Fired bool  `json:"fired"`
	Ammo  int32 `json:"ammo"`
}

func RegisterWeapon(desc WeaponDesc) (int32, error) {
	var id int32
	err := callInto("weapon.register", desc, &id)
	return id, err
}

func UnregisterWeapon(id int32) (bool, error) {
	var ok bool
	err := callInto("weapon.unregister", map[string]any{"id": id}, &ok)
	return ok, err
}

func ListWeapons() ([]WeaponInfo, error) {
	var out []WeaponInfo
	err := callInto("weapon.list", nil, &out)
	return out, err
}

func SelectWeapon(id int32) error {
	return callInto("weapon.select", map[string]any{"id": id}, nil)
}

func GetActiveWeapon() (*WeaponInfo, error) {
	raw, err := Call("weapon.get_active", nil)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil, err
	}
	var w WeaponInfo
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// FireWeapon fires id, or the active weapon when id is 0.
func FireWeapon(id int32) (WeaponFireResult, error) {
	var out WeaponFireResult
	err := callInto("weapon.fire", map[string]any{"id": id}, &out)
	return out, err
}

func SetWeaponAmmo(id, ammo int32) (int32, error) {
	var v int32
	err := callInto("weapon.set_ammo", map[string]any{"id": id, "ammo": ammo}, &v)
	return v, err
}

func AddWeaponAmmo(id, amount int32) (int32, error) {
	var v int32
	err := callInto("weapon.add_ammo", map[string]any{"id": id, "amount": amount}, &v)
	return v, err
}

func GetWeaponAmmo(id int32) (int32, error) {
	var v int32
	err := callInto("weapon.get_ammo", map[string]any{"id": id}, &v)
	return v, err
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

type ToolInfo struct {
	ID            int32  `json:"id"`
	Name          string `json:"name"`
	PrimaryDesc   string `json:"primary_desc"`
	SecondaryDesc string `json:"secondary_desc"`
	ReloadDesc    string `json:"reload_desc"`
	Color         Color  `json:"color"`
	PrimaryHook   string `json:"primary_hook"`
	SecondaryHook string `json:"secondary_hook"`
	ReloadHook    string `json:"reload_hook"`
	Owner         string `json:"owner"`
	Billboard     bool   `json:"billboard"`
}

type ToolDesc struct {
	Name          string `json:"name"`
	PrimaryDesc   string `json:"primary_desc"`
	SecondaryDesc string `json:"secondary_desc"`
	ReloadDesc    string `json:"reload_desc"`
	Color         Color  `json:"color"`
	PrimaryHook   string `json:"primary_hook"`
	SecondaryHook string `json:"secondary_hook"`
	ReloadHook    string `json:"reload_hook"`
	Billboard     bool   `json:"billboard"`
}

func RegisterTool(desc ToolDesc) (int32, error) {
	var id int32
	err := callInto("tool.register", desc, &id)
	return id, err
}

func UnregisterTool(id int32) (bool, error) {
	var ok bool
	err := callInto("tool.unregister", map[string]any{"id": id}, &ok)
	return ok, err
}

func ListTools() ([]ToolInfo, error) {
	var out []ToolInfo
	err := callInto("tool.list", nil, &out)
	return out, err
}

func SelectTool(id int32) error {
	return callInto("tool.select", map[string]any{"id": id}, nil)
}

func GetActiveTool() (*ToolInfo, error) {
	raw, err := Call("tool.get_active", nil)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil, err
	}
	var t ToolInfo
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// ToolPrimary runs the primary action of id, or the active tool when id is 0.
func ToolPrimary(id int32) error {
	return callInto("tool.primary", map[string]any{"id": id}, nil)
}

func ToolSecondary(id int32) error {
	return callInto("tool.secondary", map[string]any{"id": id}, nil)
}

func ToolReload(id int32) error {
	return callInto("tool.reload", map[string]any{"id": id}, nil)
}

// SetToolViewModel tints the engine's procedural tool gun.
func SetToolViewModel(color Color) error {
	return callInto("tool.set_viewmodel", map[string]any{"color": color}, nil)
}

// ---------------------------------------------------------------------------
// World
// ---------------------------------------------------------------------------

type SkyInfo struct {
	Top    Color `json:"top"`
	Bottom Color `json:"bottom"`
}

func SetSun(direction Vec3) (Vec3, error) {
	var v Vec3
	err := callInto("world.set_sun", map[string]any{"direction": direction}, &v)
	return v, err
}

func GetSun() (Vec3, error) {
	var v Vec3
	err := callInto("world.get_sun", nil, &v)
	return v, err
}

func SetSky(top, bottom *Color) (SkyInfo, error) {
	args := map[string]any{}
	if top != nil {
		args["top"] = *top
	}
	if bottom != nil {
		args["bottom"] = *bottom
	}
	var out SkyInfo
	err := callInto("world.set_sky", args, &out)
	return out, err
}

func GetWorldTime() (float64, error) {
	var v float64
	err := callInto("world.get_time", nil, &v)
	return v, err
}

func SetWorldTimeScale(scale float32) (float64, error) {
	var v float64
	err := callInto("world.set_time_scale", map[string]any{"scale": scale}, &v)
	return v, err
}

func PauseWorld() error  { return callInto("world.pause", nil, nil) }
func ResumeWorld() error { return callInto("world.resume", nil, nil) }
func SaveWorld() error   { return callInto("world.save", nil, nil) }
func LoadWorld() error   { return callInto("world.load", nil, nil) }

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

type SettingsInfo struct {
	Gravity     float32 `json:"gravity_scale"`
	Sensitivity float32 `json:"mouse_sensitivity"`
	FOV         float32 `json:"fov"`
	UIScale     float32 `json:"ui_scale"`
	InvertY     bool    `json:"invert_y"`
	Shadows     bool    `json:"shadows"`
	DisplayMode int32   `json:"display_mode"`
	RawInput    bool    `json:"raw_input"`
}

// SettingsUpdate only sends the fields that are set.
type SettingsUpdate struct {
	Gravity     *float32
	Sensitivity *float32
	FOV         *float32
	UIScale     *float32
	InvertY     *bool
	Shadows     *bool
	RawInput    *bool
}

func GetSettings() (SettingsInfo, error) {
	var s SettingsInfo
	err := callInto("settings.get", nil, &s)
	return s, err
}

func SetSettings(u SettingsUpdate) (SettingsInfo, error) {
	args := map[string]any{}
	if u.Gravity != nil {
		args["gravity"] = *u.Gravity
	}
	if u.Sensitivity != nil {
		args["sensitivity"] = *u.Sensitivity
	}
	if u.FOV != nil {
		args["fov"] = *u.FOV
	}
	if u.UIScale != nil {
		args["ui_scale"] = *u.UIScale
	}
	if u.InvertY != nil {
		args["invert_y"] = *u.InvertY
	}
	if u.Shadows != nil {
		args["shadows"] = *u.Shadows
	}
	if u.RawInput != nil {
		args["raw_input"] = *u.RawInput
	}
	var s SettingsInfo
	err := callInto("settings.set", args, &s)
	return s, err
}

func SaveSettings() error { return callInto("settings.save", nil, nil) }

// ---------------------------------------------------------------------------
// Render (3D debug drawing)
// ---------------------------------------------------------------------------

func DrawLine3D(from, to Vec3, color Color, thickness float32) error {
	return callInto("render.draw_line_3d", map[string]any{
		"from": from, "to": to, "color": color, "thickness": thickness,
	}, nil)
}

func DrawBox3D(center, size Vec3, color Color) error {
	return callInto("render.draw_box_3d", map[string]any{
		"center": center, "size": size, "color": color,
	}, nil)
}

func DrawSphere3D(center Vec3, radius float32, color Color, wire bool, segments int32) error {
	return callInto("render.draw_sphere_3d", map[string]any{
		"center": center, "radius": radius, "color": color,
		"wire": wire, "segments": segments,
	}, nil)
}

func DrawRay3D(origin, direction Vec3, length float32, color Color) error {
	return callInto("render.draw_ray_3d", map[string]any{
		"origin": origin, "direction": direction, "length": length, "color": color,
	}, nil)
}

func DrawCircle3D(center, normal Vec3, radius float32, color Color, segments int32) error {
	return callInto("render.draw_circle_3d", map[string]any{
		"center": center, "normal": normal, "radius": radius,
		"color": color, "segments": segments,
	}, nil)
}

func ClearDebugDraw() error { return callInto("render.clear_debug", nil, nil) }

func WorldToScreen(pos Vec3) (Vec2, error) {
	var v Vec2
	err := callInto("render.world_to_screen", map[string]any{"position": pos}, &v)
	return v, err
}

func ScreenToWorldRay(x, y float32) (Ray, error) {
	var r Ray
	err := callInto("render.screen_to_world_ray", map[string]any{"x": x, "y": y}, &r)
	return r, err
}

func SetFOV(fov float32) (float32, error) {
	var v float32
	err := callInto("render.set_fov", map[string]any{"fov": fov}, &v)
	return v, err
}

func SetShadows(enabled bool) error {
	return callInto("render.set_shadows", map[string]any{"enabled": enabled}, nil)
}

func GetScreenSize() (int32, int32, error) {
	var s struct {
		Width  int32 `json:"width"`
		Height int32 `json:"height"`
	}
	err := callInto("render.screen_size", nil, &s)
	return s.Width, s.Height, err
}

// ---------------------------------------------------------------------------
// Screen-space UI drawing
// ---------------------------------------------------------------------------

func DrawText(text string, x, y, size, spacing float32, color Color) error {
	return callInto("ui.draw_text", map[string]any{
		"text": text, "x": x, "y": y, "size": size, "spacing": spacing, "color": color,
	}, nil)
}

func DrawRect(x, y, w, h float32, color Color, filled bool, thickness float32) error {
	return callInto("ui.draw_rect", map[string]any{
		"x": x, "y": y, "w": w, "h": h, "color": color,
		"filled": filled, "thickness": thickness,
	}, nil)
}

func DrawLine(x0, y0, x1, y1, thickness float32, color Color) error {
	return callInto("ui.draw_line", map[string]any{
		"x0": x0, "y0": y0, "x1": x1, "y1": y1,
		"thickness": thickness, "color": color,
	}, nil)
}

func DrawCircle(x, y, radius float32, color Color, filled bool, thickness float32, segments int32) error {
	return callInto("ui.draw_circle", map[string]any{
		"x": x, "y": y, "radius": radius, "color": color,
		"filled": filled, "thickness": thickness, "segments": segments,
	}, nil)
}

func DrawTriangle(x0, y0, x1, y1, x2, y2 float32, color Color) error {
	return callInto("ui.draw_triangle", map[string]any{
		"x0": x0, "y0": y0, "x1": x1, "y1": y1, "x2": x2, "y2": y2, "color": color,
	}, nil)
}

func MeasureText(text string, size, spacing float32) (float32, error) {
	var w float32
	err := callInto("ui.measure_text", map[string]any{
		"text": text, "size": size, "spacing": spacing,
	}, &w)
	return w, err
}

func GetUIScreenSize() (float32, float32, error) {
	var s struct {
		Width  float32 `json:"width"`
		Height float32 `json:"height"`
	}
	err := callInto("ui.screen_size", nil, &s)
	return s.Width, s.Height, err
}

func ClearDraw() error { return callInto("ui.clear", nil, nil) }

// AddUIElement registers an interactive element. Kind is "button", "panel",
// "slider" or "text_input".
func AddUIElement(kind string, x, y, w, h float32, label string, color Color, value, min, max float32) (int32, error) {
	var id int32
	err := callInto("ui.add_element", map[string]any{
		"kind": kind, "x": x, "y": y, "w": w, "h": h,
		"label": label, "color": color,
		"value": value, "min": min, "max": max,
	}, &id)
	return id, err
}

func ListUIElements() ([]UIElement, error) {
	var out []UIElement
	err := callInto("ui.list_elements", nil, &out)
	return out, err
}

func ClearUIElements() error { return callInto("ui.clear_elements", nil, nil) }

// ---------------------------------------------------------------------------
// Content presets
// ---------------------------------------------------------------------------

type PresetInfo struct {
	Index       int32   `json:"index"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Kind        string  `json:"kind"`
	Half        Vec3    `json:"half"`
	Radius      float32 `json:"radius"`
	Height      float32 `json:"height"`
	Density     float32 `json:"density"`
	Friction    float32 `json:"friction"`
	Restitution float32 `json:"restitution"`
	Color       Color   `json:"color"`
	HasModel    bool    `json:"has_model"`
}

type PresetDesc struct {
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Kind        string  `json:"kind"`
	Half        *Vec3   `json:"half,omitempty"`
	Radius      float32 `json:"radius,omitempty"`
	Height      float32 `json:"height,omitempty"`
	Density     float32 `json:"density,omitempty"`
	Friction    float32 `json:"friction,omitempty"`
	Restitution float32 `json:"restitution,omitempty"`
	Color       Color   `json:"color"`
}

func ListPresets() ([]PresetInfo, error) {
	var out []PresetInfo
	err := callInto("content.list_presets", nil, &out)
	return out, err
}

func AddPreset(desc PresetDesc) (int32, error) {
	var id int32
	err := callInto("content.add_preset", desc, &id)
	return id, err
}

// RemovePreset removes a preset by name when name is non-empty, otherwise by
// index.
func RemovePreset(index int32, name string) (bool, error) {
	var ok bool
	err := callInto("content.remove_preset", map[string]any{"index": index, "name": name}, &ok)
	return ok, err
}

// ---------------------------------------------------------------------------
// Sandboxed mod resources
// ---------------------------------------------------------------------------

func ReadResource(path string) (string, error) {
	var s string
	err := callInto("resource.read_text", map[string]any{"path": path}, &s)
	return s, err
}

func WriteResource(path, content string) error {
	return callInto("resource.write_text", map[string]any{"path": path, "content": content}, nil)
}

func AppendResource(path, content string) error {
	return callInto("resource.write_text", map[string]any{"path": path, "content": content, "append": true}, nil)
}

func ResourceExists(path string) (bool, error) {
	var ok bool
	err := callInto("resource.exists", map[string]any{"path": path}, &ok)
	return ok, err
}

func DeleteResource(path string) error {
	return callInto("resource.delete", map[string]any{"path": path}, nil)
}

func ListResources(path string) ([]string, error) {
	var out []string
	err := callInto("resource.list", map[string]any{"path": path}, &out)
	return out, err
}

func ResourceDataDir() (string, error) {
	var s string
	err := callInto("resource.data_dir", nil, &s)
	return s, err
}

// ---------------------------------------------------------------------------
// Input extras
// ---------------------------------------------------------------------------

func GetMousePosition() (Vec2, error) {
	var v Vec2
	err := callInto("input.mouse_pos", nil, &v)
	return v, err
}

func GetMouseDelta() (Vec2, error) {
	var v Vec2
	err := callInto("input.mouse_delta", nil, &v)
	return v, err
}

func InputFocused() (bool, error) {
	var ok bool
	err := callInto("input.is_focused", nil, &ok)
	return ok, err
}

func SetCursorLocked(locked bool) error {
	return callInto("input.set_cursor_locked", map[string]any{"locked": locked}, nil)
}

// ---------------------------------------------------------------------------
// Entity setters and physics helpers
// ---------------------------------------------------------------------------

type DamageResult struct {
	Killed bool    `json:"killed"`
	Health float32 `json:"health"`
}

func SetEntityPos(id int32, pos Vec3) error {
	return callInto("entity.set_pos", map[string]any{"id": id, "position": pos}, nil)
}

func SetEntityRotation(id int32, yaw, pitch float32) error {
	return callInto("entity.set_rot", map[string]any{"id": id, "yaw": yaw, "pitch": pitch}, nil)
}

func SetEntityQuat(id int32, q Quat) error {
	return callInto("entity.set_rot", map[string]any{"id": id, "rotation": q}, nil)
}

func SetEntityTransform(id int32, pos Vec3, q Quat) error {
	return callInto("entity.set_transform", map[string]any{
		"id": id, "position": pos, "rotation": q,
	}, nil)
}

func SetEntityName(id int32, name string) error {
	return callInto("entity.set_name", map[string]any{"id": id, "name": name}, nil)
}

func SetEntityFrozen(id int32, frozen bool) error {
	return callInto("entity.set_frozen", map[string]any{"id": id, "frozen": frozen}, nil)
}

func SetEntityInvulnerable(id int32, invulnerable bool) error {
	return callInto("entity.set_invulnerable", map[string]any{"id": id, "invulnerable": invulnerable}, nil)
}

func SetEntityHealth(id int32, health, maxHealth float32) error {
	return callInto("entity.set_health", map[string]any{
		"id": id, "health": health, "max_health": maxHealth,
	}, nil)
}

func SetEntityStatic(id int32, static bool) error {
	return callInto("entity.set_static", map[string]any{"id": id, "static": static}, nil)
}

func SetEntityLinearVelocity(id int32, v Vec3) error {
	return callInto("entity.set_linear_velocity", map[string]any{"id": id, "velocity": v}, nil)
}

func ApplyEntityForce(id int32, force Vec3) error {
	return callInto("entity.apply_force", map[string]any{"id": id, "fx": force.X, "fy": force.Y, "fz": force.Z}, nil)
}

func ApplyEntityImpulse(id int32, impulse Vec3) error {
	return callInto("entity.apply_impulse", map[string]any{"id": id, "ix": impulse.X, "iy": impulse.Y, "iz": impulse.Z}, nil)
}

func ApplyEntityTorque(id int32, torque Vec3) error {
	return callInto("entity.apply_torque", map[string]any{"id": id, "tx": torque.X, "ty": torque.Y, "tz": torque.Z}, nil)
}

func ApplyEntityAngularImpulse(id int32, impulse Vec3) error {
	return callInto("entity.apply_angular_impulse", map[string]any{"id": id, "tx": impulse.X, "ty": impulse.Y, "tz": impulse.Z}, nil)
}

func ApplyEntityForceAtPoint(id int32, force, point Vec3) error {
	return callInto("entity.apply_force_at_point", map[string]any{
		"id": id, "fx": force.X, "fy": force.Y, "fz": force.Z, "point": point,
	}, nil)
}

func ApplyEntityImpulseAtPoint(id int32, impulse, point Vec3) error {
	return callInto("entity.apply_impulse_at_point", map[string]any{
		"id": id, "ix": impulse.X, "iy": impulse.Y, "iz": impulse.Z, "point": point,
	}, nil)
}

func SetEntityMaterial(id int32, density, friction, restitution *float32) error {
	args := map[string]any{"id": id}
	if density != nil {
		args["density"] = *density
	}
	if friction != nil {
		args["friction"] = *friction
	}
	if restitution != nil {
		args["restitution"] = *restitution
	}
	return callInto("entity.set_material", args, nil)
}

func SetEntityBullet(id int32, bullet bool) error {
	return callInto("entity.set_bullet", map[string]any{"id": id, "bullet": bullet}, nil)
}

func SetEntityCollision(id int32, enabled bool) error {
	return callInto("entity.set_collision", map[string]any{"id": id, "enabled": enabled}, nil)
}

func SetEntityAwake(id int32, awake bool) error {
	cmd := "entity.sleep"
	if awake {
		cmd = "entity.wake"
	}
	return callInto(cmd, map[string]any{"id": id}, nil)
}

// SpawnEntityCopy duplicates an entity, optionally at a new position.
func SpawnEntityCopy(id int32, pos *Vec3) (EntityInfo, error) {
	args := map[string]any{"id": id}
	if pos != nil {
		args["position"] = *pos
	}
	var e EntityInfo
	err := callInto("entity.spawn_copy", args, &e)
	return e, err
}

func RemoveEntity(id int32) error {
	return callInto("entity.remove", map[string]any{"id": id}, nil)
}

// ClearEntities removes every prop, including transient ones.
func ClearEntities() (int, error) {
	var n int
	err := callInto("entity.clear", nil, &n)
	return n, err
}

// ClearTransientEntities removes only transient props (projectiles, etc.).
func ClearTransientEntities() (int, error) {
	var n int
	err := callInto("entity.clear", map[string]any{"transient": true}, &n)
	return n, err
}

func DamageEntity(id int32, damage float32, point *Vec3) (DamageResult, error) {
	args := map[string]any{"id": id, "damage": damage}
	if point != nil {
		args["point"] = *point
	}
	var out DamageResult
	err := callInto("entity.damage", args, &out)
	return out, err
}

func HealEntity(id int32, amount float32) (float32, error) {
	var v float32
	err := callInto("entity.heal", map[string]any{"id": id, "amount": amount}, &v)
	return v, err
}

// EntityExists reports whether an entity with the given ID is still in the
// world. It is a thin wrapper over Entity.Get that returns false instead of
// an error for missing entities.
func EntityExists(id int32) (bool, error) {
	raw, err := Call("entity.exists", map[string]any{"id": id})
	if err != nil {
		return false, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return false, nil
	}
	return true, nil
}

// FreezeEntity sets the frozen state of one prop. Passing true freezes it in
// place; false releases it back to physics.
func FreezeEntity(id int32, frozen bool) error {
	return callInto("entity.set_frozen", map[string]any{"id": id, "frozen": frozen}, nil)
}

// ---------------------------------------------------------------------------
// Networking (dedicated server / connected client)
// ---------------------------------------------------------------------------

// NetStatusInfo describes this process's multiplayer role. Mode is "off",
// "server" or "client". On a dedicated server Port, Players and Tick are
// populated; on a client Address, PlayerID and PlayerName are populated.
type NetStatusInfo struct {
	Mode       string `json:"mode"`
	Server     bool   `json:"server"`
	Connected  bool   `json:"connected"`
	Address    string `json:"address,omitempty"`
	Port       int    `json:"port,omitempty"`
	Map        string `json:"map"`
	Players    int    `json:"players"`
	PlayerID   int32  `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Tick       uint64 `json:"tick,omitempty"`
	Protocol   int    `json:"protocol"`
}

// NetPlayerInfo is one connected player. Position is authoritative on the
// server and interpolated on a client. Held is the prop network ID or 0.
type NetPlayerInfo struct {
	ID       int32   `json:"id"`
	Name     string  `json:"name"`
	Position Vec3    `json:"position"`
	Yaw      float32 `json:"yaw"`
	Pitch    float32 `json:"pitch"`
	Fly      bool    `json:"fly"`
	Held     int32   `json:"held"`
	Local    bool    `json:"local"`
}

// NetStatus returns the current multiplayer status. Safe in single-player.
func NetStatus() (NetStatusInfo, error) {
	var info NetStatusInfo
	err := callInto("net.status", nil, &info)
	return info, err
}

// NetPlayers lists every connected player. On a client the local player is
// included first with Local set to true.
func NetPlayers() ([]NetPlayerInfo, error) {
	var out []NetPlayerInfo
	err := callInto("net.players", nil, &out)
	return out, err
}

// BroadcastToast shows a toast on every connected client's HUD. Only valid on
// a dedicated server. Pass nil for the default accent colour.
func BroadcastToast(text string, color *Color) error {
	args := map[string]any{"text": text}
	if color != nil {
		args["color"] = *color
	}
	return callInto("net.broadcast_toast", args, nil)
}

// SendNetEvent delivers a custom event to mods on the other side of the link.
// On a server it broadcasts to all clients; on a client it sends to the server.
func SendNetEvent(event string, data any) error {
	return callInto("net.send_event", map[string]any{"event": event, "data": data}, nil)
}

// KickPlayer disconnects one client. Only valid on a dedicated server. Returns
// false when no client with that ID is connected.
func KickPlayer(id int32, reason string) (bool, error) {
	var ok bool
	err := callInto("net.kick", map[string]any{"id": id, "reason": reason}, &ok)
	return ok, err
}
