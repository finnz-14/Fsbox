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
	Index     int32
	Name      string
	Keyframes int32
	Bones     int32
	Current   bool
	Playing   bool
	Frame     float32
	Speed     float32
}

type EntityInfo struct {
	ID           int32
	Name         string
	Kind         string
	Static       bool
	Frozen       bool
	Invulnerable bool
	Transient    bool
	Health       float32
	MaxHealth    float32
	Position     Vec3
	Color        Color
	Half         Vec3
	Radius       float32
	Height       float32
	Owner        string
}

type PlayerInfo struct {
	Position Vec3
	Forward  Vec3
}

// EngineInfo is a small snapshot of the engine state. State is one of
// "menu", "map_menu", "playing", "paused", "help", "settings", "spawn" or
// "tools".
type EngineInfo struct {
	State     string  `json:"state"`
	Time      float64 `json:"time"`
	Frame     int64   `json:"frame"`
	FPS       int32   `json:"fps"`
	TimeScale float64 `json:"time_scale"`
	Paused    bool    `json:"paused"`
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
	var pi struct {
		Position Vec3 `json:"position"`
		Forward  Vec3 `json:"forward"`
	}
	if err := json.Unmarshal(raw, &pi); err != nil {
		return PlayerInfo{}, err
	}
	return PlayerInfo{Position: pi.Position, Forward: pi.Forward}, nil
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
