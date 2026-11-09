package fsbox

import "encoding/json"

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
	var info struct {
		ID int32 `json:"id"`
	}
	_ = json.Unmarshal(raw, &info)
	return info.ID, nil
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
// Camera
// ---------------------------------------------------------------------------

func SetCameraMode(mode int32) error {
	_, err := Call("camera.set_mode", map[string]any{"mode": mode})
	return err
}

func SetCameraDistance(d float32) error {
	_, err := Call("camera.set_distance", map[string]any{"distance": d})
	return err
}

func SetCameraOffset(o Vec3) error {
	_, err := Call("camera.set_offset", map[string]any{"offset": o})
	return err
}

// WorldToScreen converts a 3D world position to 2D screen coordinates.
func WorldToScreen(pos Vec3) (Vec3, error) {
	raw, err := Call("render.world_to_screen", map[string]any{"position": pos})
	if err != nil {
		return Vec3{}, err
	}
	var v Vec3
	_ = json.Unmarshal(raw, &v)
	return v, nil
}

// ---------------------------------------------------------------------------
// Cursor
// ---------------------------------------------------------------------------

func SetCursorLocked(locked bool) error {
	_, err := Call("input.set_cursor_locked", map[string]any{"locked": locked})
	return err
}

// InputKeyPressed queries whether a key was pressed this frame.
func InputKeyPressed(key int32) (bool, error) {
	raw, err := Call("input.key_pressed", map[string]any{"key": key})
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
// Animations
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

func GetAnimations(id int32) ([]AnimationInfo, error) {
	raw, err := Call("entity.get_animations", map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	var out []AnimationInfo
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func PlayAnimation(id int32, animName string, speed float32) error {
	_, err := Call("entity.play_animation", map[string]any{"id": id, "anim": animName, "speed": speed})
	return err
}

func PlayAnimationByIndex(id, animIndex int32, speed float32) error {
	_, err := Call("entity.play_animation", map[string]any{"id": id, "anim_index": animIndex, "speed": speed})
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

func StopAnimation(id int32) error {
	_, err := Call("entity.stop_animation", map[string]any{"id": id})
	return err
}

func SetAnimationSpeed(id int32, speed float32) error {
	_, err := Call("entity.set_animation_speed", map[string]any{"id": id, "speed": speed})
	return err
}

// ---------------------------------------------------------------------------
// Player model
// ---------------------------------------------------------------------------

func SetPlayerModel(h int32) error {
	_, err := Call("player.set_model", map[string]any{"model": h})
	return err
}

func ClearPlayerModel() error {
	_, err := Call("player.clear_model", nil)
	return err
}
