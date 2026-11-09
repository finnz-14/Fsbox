package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const ModAPIVersion = "2.0.0"




// ---------------------------------------------------------------------------
// Public mod API data types
//
// Everything a mod can see or send is JSON encoded. The engine keeps these
// structs small, stable and explicit so the WASM ABI and the Go worker RPC can
// share exactly the same command surface.
// ---------------------------------------------------------------------------

type Vec3Info struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type QuatInfo struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	W float32 `json:"w"`
}

type ColorInfo struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

type EntityInfo struct {
	ID           int32     `json:"id"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	Static       bool      `json:"static"`
	Frozen       bool      `json:"frozen"`
	Invulnerable bool      `json:"invulnerable"`
	Transient    bool      `json:"transient"`
	Health       float32   `json:"health"`
	MaxHealth    float32   `json:"max_health"`
	Position     Vec3Info  `json:"position"`
	Rotation     QuatInfo  `json:"rotation"`
	Velocity     Vec3Info  `json:"velocity"`
	AngularVel   Vec3Info  `json:"angular_velocity"`
	Mass         float32   `json:"mass"`
	Color        ColorInfo `json:"color"`
	Half         Vec3Info  `json:"half"`
	Radius       float32   `json:"radius"`
	Height       float32   `json:"height"`
	Density      float32   `json:"density"`
	Friction     float32   `json:"friction"`
	Restitution  float32   `json:"restitution"`
	Owner        string    `json:"owner"`
	HasModel     bool      `json:"has_model"`
}

type PlayerInfo struct {
	Position   Vec3Info  `json:"position"`
	Rotation   QuatInfo  `json:"rotation"`
	Velocity   Vec3Info  `json:"velocity"`
	Yaw        float32   `json:"yaw"`
	Pitch      float32   `json:"pitch"`
	Fly        bool      `json:"fly"`
	Grounded   bool      `json:"grounded"`
	Health     float32   `json:"health"`
	MaxHealth  float32   `json:"max_health"`
	SpeedScale float32   `json:"speed_scale"`
	JumpScale  float32   `json:"jump_scale"`
	Eye        Vec3Info  `json:"eye"`
	Forward    Vec3Info  `json:"forward"`
	Color      ColorInfo `json:"color"`
}

type EngineInfo struct {
	Game        string  `json:"game"`
	Version     string  `json:"version"`
	APIVersion  string  `json:"api_version"`
	Time        float64 `json:"time"`
	Frame       int64   `json:"frame"`
	FPS         int32   `json:"fps"`
	State       string  `json:"state"`
	Paused      bool    `json:"paused"`
	TimeScale   float64 `json:"time_scale"`
	EntityCount int     `json:"entity_count"`
	PropCount   int     `json:"prop_count"`
	Player      PlayerInfo `json:"player"`
}

type PresetInfo struct {
	Index       int32     `json:"index"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Kind        string    `json:"kind"`
	Half        Vec3Info  `json:"half"`
	Radius      float32   `json:"radius"`
	Height      float32   `json:"height"`
	Density     float32   `json:"density"`
	Friction    float32   `json:"friction"`
	Restitution float32   `json:"restitution"`
	Color       ColorInfo `json:"color"`
	HasModel    bool      `json:"has_model"`
}

type RaycastInfo struct {
	Hit      bool     `json:"hit"`
	EntityID int32    `json:"entity_id"`
	Entity   *EntityInfo `json:"entity,omitempty"`
	Point    Vec3Info `json:"point"`
	Normal   Vec3Info `json:"normal"`
	Fraction float32  `json:"fraction"`
	Distance float32  `json:"distance"`
}

type ConstraintInfo struct {
	ID      int32  `json:"id"`
	Type    string `json:"type"`
	EntityA int32  `json:"entity_a"`
	EntityB int32  `json:"entity_b"`
	Owner   string `json:"owner"`
}

type WeaponInfo struct {
	ID              int32     `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Damage          float32   `json:"damage"`
	Range           float32   `json:"range"`
	Ammo            int32     `json:"ammo"`
	MaxAmmo         int32     `json:"max_ammo"`
	FireRate        float32   `json:"fire_rate"`
	Recoil          float32   `json:"recoil"`
	Spread          float32   `json:"spread"`
	ProjectileSpeed float32   `json:"projectile_speed"`
	ProjectileLife  float32   `json:"projectile_life"`
	ProjectileSize  float32   `json:"projectile_size"`
	Color           ColorInfo `json:"color"`
	Owner           string    `json:"owner"`
}

type ToolInfo struct {
	ID            int32     `json:"id"`
	Name          string    `json:"name"`
	PrimaryDesc   string    `json:"primary_desc"`
	SecondaryDesc string    `json:"secondary_desc"`
	ReloadDesc    string    `json:"reload_desc"`
	Color         ColorInfo `json:"color"`
	PrimaryHook   string    `json:"primary_hook"`
	SecondaryHook string    `json:"secondary_hook"`
	ReloadHook    string    `json:"reload_hook"`
	Owner         string    `json:"owner"`
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func vecInfo(v rl.Vector3) Vec3Info {
	return Vec3Info{X: v.X, Y: v.Y, Z: v.Z}
}

func (v Vec3Info) rl() rl.Vector3 {
	return rlv(v.X, v.Y, v.Z)
}

func (v Vec3Info) box() box3d.Vec3 {
	return bv(v.X, v.Y, v.Z)
}

func (v Vec3Info) pos() box3d.Pos {
	return bp(v.X, v.Y, v.Z)
}

func quatInfo(q box3d.Quat) QuatInfo {
	return QuatInfo{X: q.V.X, Y: q.V.Y, Z: q.V.Z, W: q.S}
}

func (q QuatInfo) box() box3d.Quat {
	out := box3d.Quat{V: box3d.Vec3{X: q.X, Y: q.Y, Z: q.Z}, S: q.W}
	return normalizeQuat(out)
}

func colorInfo(c rl.Color) ColorInfo {
	return ColorInfo{R: c.R, G: c.G, B: c.B, A: c.A}
}

func (c ColorInfo) rl() rl.Color {
	a := c.A
	if a == 0 {
		a = 255
	}
	return rl.Color{R: c.R, G: c.G, B: c.B, A: a}
}

func (c ColorInfo) provided() bool {
	return c.R != 0 || c.G != 0 || c.B != 0 || c.A != 0
}

func rotFromArgs(q *QuatInfo, yaw, pitch, roll float32) box3d.Quat {
	if q != nil {
		return q.box()
	}
	if yaw == 0 && pitch == 0 && roll == 0 {
		return box3d.QuatIdentity
	}
	out := box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, yaw)
	out = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, pitch), out)
	out = box3d.MulQuat(box3d.MakeQuatFromAxisAngle(box3d.Vec3{Z: 1}, roll), out)
	return normalizeQuat(out)
}

func decodeArgs(args json.RawMessage, dst any) error {
	if len(args) == 0 || string(args) == "null" {
		args = json.RawMessage("{}")
	}
	if err := json.Unmarshal(args, dst); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Command router
// ---------------------------------------------------------------------------

// ModCall executes one mod command. owner is the manifest ID of the calling
// mod (used for ownership, hooks and resource isolation).
func (g *Game) ModCall(owner, cmd string, args json.RawMessage) (any, error) {
	if g == nil {
		return nil, fmt.Errorf("game is not running")
	}
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	// ----- engine ---------------------------------------------------------
	case "engine.info":
		return g.engineInfo(), nil
	case "engine.log":
		var a struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		fmt.Printf("[mod:%s] %s\n", owner, a.Message)
		return true, nil
	case "engine.toast":
		var a struct {
			Message string    `json:"message"`
			Color   ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		c := colAccent
		if a.Color.provided() {
			c = a.Color.rl()
		}
		g.toast(a.Message, c)
		return true, nil
	case "engine.set_time_scale":
		var a struct {
			Scale float32 `json:"scale"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.timeScale = float64(clampf(a.Scale, 0, 10))
		return g.timeScale, nil
	case "engine.get_time":
		return g.time, nil
	case "engine.set_paused":
		var a struct {
			Paused bool `json:"paused"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Paused {
			g.state = statePaused
		} else if g.state == statePaused {
			g.resumePlaying()
		}
		return true, nil
	case "engine.reset_world":
		var a struct {
			WithStarter *bool `json:"with_starter"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		starter := true
		if a.WithStarter != nil {
			starter = *a.WithStarter
		}
		g.resetWorld(starter, defaultSpawn, 0, -0.12, false)
		return true, nil
	case "engine.save":
		g.saveGame()
		return true, nil
	case "engine.load":
		g.loadGame()
		return true, nil
	case "engine.quit":
		g.quitRequested = true
		return true, nil
	case "engine.screenshot":
		var a struct {
			Path string `json:"path"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Path == "" {
			a.Path = "screenshot.png"
		}
		rl.TakeScreenshot(a.Path)
		return a.Path, nil
	case "engine.bind_key":
		var a struct {
			Key   int32  `json:"key"`
			Event string `json:"event"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.modInputHooks == nil {
			g.modInputHooks = make(map[int32][]string)
		}
		g.modInputHooks[a.Key] = appendUnique(g.modInputHooks[a.Key], a.Event)
		return true, nil
	case "engine.unbind_key":
		var a struct {
			Key   int32  `json:"key"`
			Event string `json:"event"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.modInputHooks[a.Key] = removeString(g.modInputHooks[a.Key], a.Event)
		return true, nil
	case "engine.bind_mouse":
		var a struct {
			Button int32  `json:"button"`
			Event  string `json:"event"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.modMouseHooks == nil {
			g.modMouseHooks = make(map[int32][]string)
		}
		g.modMouseHooks[a.Button] = appendUnique(g.modMouseHooks[a.Button], a.Event)
		return true, nil
	case "engine.unbind_mouse":
		var a struct {
			Button int32  `json:"button"`
			Event  string `json:"event"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.modMouseHooks[a.Button] = removeString(g.modMouseHooks[a.Button], a.Event)
		return true, nil
	case "engine.set_hud_hook":
		var a struct {
			Hook string `json:"hook"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.modHUDHooks = appendUnique(g.modHUDHooks, a.Hook)
		return true, nil
	case "engine.api_version":
		return ModAPIVersion, nil
	case "engine.has_permission":
		var a struct {
			Permission string `json:"permission"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.mods != nil {
			return g.mods.OwnerHasPermission(owner, a.Permission), nil
		}
		return true, nil

	// ----- entity ---------------------------------------------------------
	case "entity.spawn", "entity.spawn_preset", "world.spawn_preset":
		return g.apiSpawnEntity(owner, cmd, args)
	case "entity.remove":
		return g.apiRemoveEntity(args)
	case "entity.clear":
		var a struct {
			Transient *bool `json:"transient"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		n := 0
		for _, e := range append([]*Entity(nil), g.props...) {
			if a.Transient != nil && e.Transient != *a.Transient {
				continue
			}
			g.removeEntity(e)
			n++
		}
		return n, nil
	case "entity.list":
		var a struct {
			Static  bool `json:"static"`
			Props   bool `json:"props"`
			Transient bool `json:"transient"`
		}
		_ = decodeArgs(args, &a)
		out := make([]EntityInfo, 0, len(g.entities))
		for _, e := range g.entities {
			if a.Props && e.Static {
				continue
			}
			if a.Static && !e.Static {
				continue
			}
			if !a.Transient && e.Transient {
				continue
			}
			out = append(out, g.entityInfo(e))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
		return out, nil
	case "entity.get", "entity.exists":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return nil, nil
		}
		return g.entityInfo(e), nil
	case "entity.set_pos":
		return g.apiEntitySetPos(args)
	case "entity.set_rot":
		return g.apiEntitySetRot(args)
	case "entity.set_transform":
		return g.apiEntitySetTransform(args)
	case "entity.set_color":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Color ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Color = a.Color.rl()
		return true, nil
	case "entity.set_name":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Name string `json:"name"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Name = a.Name
		e.Body.SetName(a.Name)
		return true, nil
	case "entity.set_frozen", "entity.freeze":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Frozen bool `json:"frozen"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		g.setFrozen(e, a.Frozen)
		return true, nil
	case "entity.set_invulnerable":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Invulnerable bool `json:"invulnerable"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Invulnerable = a.Invulnerable
		return true, nil
	case "entity.set_health":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Health    float32 `json:"health"`
			MaxHealth float32 `json:"max_health"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		return g.setEntityHealth(e, a.Health, a.MaxHealth), nil
	case "entity.damage":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Damage float32  `json:"damage"`
			Point  *Vec3Info `json:"point"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		point := e.position()
		if a.Point != nil {
			point = a.Point.rl()
		}
		dead := g.damageEntity(e, a.Damage, point)
		return map[string]any{"killed": dead, "health": e.Health}, nil
	case "entity.heal":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Amount float32 `json:"amount"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		g.healEntity(e, a.Amount)
		return e.Health, nil
	case "entity.set_static":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Static bool `json:"static"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		if a.Static {
			e.Body.SetType(box3d.StaticBody)
			e.Static = true
		} else {
			e.Body.SetType(box3d.DynamicBody)
			e.Static = false
		}
		return true, nil
	case "entity.set_linear_velocity":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Velocity Vec3Info `json:"velocity"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Body.SetLinearVelocity(a.Velocity.box())
		e.Body.SetAwake(true)
		return true, nil
	case "entity.set_angular_velocity":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Velocity Vec3Info `json:"velocity"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Body.SetAngularVelocity(a.Velocity.box())
		e.Body.SetAwake(true)
		return true, nil
	case "entity.apply_force", "entity.apply_impulse", "entity.apply_torque",
		"entity.apply_force_at_point", "entity.apply_impulse_at_point",
		"entity.apply_angular_impulse":
		return g.apiEntityPhysics(cmd, args)
	case "entity.set_gravity_scale":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Scale float32 `json:"scale"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Body.SetGravityScale(a.Scale)
		e.Body.SetAwake(true)
		return true, nil
	case "entity.set_material":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Density     *float32 `json:"density"`
			Friction    *float32 `json:"friction"`
			Restitution *float32 `json:"restitution"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		applyMaterial(e, a.Density, a.Friction, a.Restitution)
		return true, nil
	case "entity.set_bullet":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Bullet bool `json:"bullet"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Body.SetBullet(a.Bullet)
		return true, nil
	case "entity.wake", "entity.sleep":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		e.Body.SetAwake(cmd == "entity.wake")
		return true, nil
	case "entity.set_collision":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Enabled bool `json:"enabled"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		shapes := make([]box3d.ShapeID, e.Body.GetShapeCount())
		e.Body.GetShapes(shapes)
		for _, s := range shapes {
			if a.Enabled {
				s.SetFilter(propFilter(), false)
			} else {
				s.SetFilter(box3d.Filter{CategoryBits: 0, MaskBits: 0}, false)
			}
		}
		e.Body.SetAwake(true)
		return true, nil
	case "entity.spawn_copy":
		e, err := g.entityFromArgs(args)
		if err != nil {
			return nil, err
		}
		var a struct {
			Position *Vec3Info `json:"position"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if e == nil {
			return false, nil
		}
		pos := e.Body.GetPosition()
		if a.Position != nil {
			pos = a.Position.pos()
		}
		copy := g.spawnCopy(e, pos, e.Body.GetRotation())
		if copy == nil {
			return nil, nil
		}
		return g.entityInfo(copy), nil

	// ----- physics --------------------------------------------------------
	case "physics.set_gravity":
		var a struct {
			Scale *float32 `json:"scale"`
			Y     *float32 `json:"y"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Y != nil {
			g.world.SetGravity(bv(0, *a.Y, 0))
			g.settings.Gravity = clampf(*a.Y/baseGravity, 0, 2.5)
		} else if a.Scale != nil {
			g.setGravity(*a.Scale)
		}
		g.settingsDirty = true
		return g.world.GetGravity(), nil
	case "physics.get_gravity":
		return g.world.GetGravity(), nil
	case "physics.raycast":
		var a struct {
			Origin  Vec3Info `json:"origin"`
			Dir     Vec3Info `json:"direction"`
			MaxDist float32  `json:"max_distance"`
			Mask    *uint64  `json:"mask"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		dir := a.Dir.rl()
		l := lenRl(dir)
		if l < 1e-6 {
			return RaycastInfo{}, nil
		}
		dir = rlv(dir.X/l, dir.Y/l, dir.Z/l)
		maxDist := a.MaxDist
		if maxDist <= 0 {
			maxDist = 100
		}
		mask := catStatic | catProp
		if a.Mask != nil {
			mask = *a.Mask
		}
		res := g.world.CastRayClosest(a.Origin.pos(), toBv(rlv(dir.X*maxDist, dir.Y*maxDist, dir.Z*maxDist)), queryFilter(mask))
		return g.raycastInfo(res, maxDist), nil
	case "physics.raycast_camera":
		var a struct {
			MaxDist float32 `json:"max_distance"`
			Mask    *uint64 `json:"mask"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		maxDist := a.MaxDist
		if maxDist <= 0 {
			maxDist = 100
		}
		mask := catStatic | catProp
		if a.Mask != nil {
			mask = *a.Mask
		}
		res := g.castRay(maxDist, mask)
		return g.raycastInfo(res, maxDist), nil
	case "physics.overlap_sphere":
		var a struct {
			Center Vec3Info `json:"center"`
			Radius float32  `json:"radius"`
			Mask   *uint64  `json:"mask"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		mask := catStatic | catProp
		if a.Mask != nil {
			mask = *a.Mask
		}
		seen := make(map[int32]bool)
		out := make([]EntityInfo, 0)
		proxy := box3d.ShapeProxy{Points: []box3d.Vec3{{X: a.Center.X, Y: a.Center.Y, Z: a.Center.Z}}, Count: 1, Radius: a.Radius}
		g.world.OverlapShape(bp(a.Center.X, a.Center.Y, a.Center.Z), &proxy, queryFilter(mask), func(shape box3d.ShapeID, context any) bool {
			body := shape.GetBody()
			if e := g.byBody[body]; e != nil && !seen[e.ID] {
				seen[e.ID] = true
				out = append(out, g.entityInfo(e))
			}
			return true
		}, nil)
		return out, nil
	case "physics.explode":
		var a struct {
			Center         Vec3Info `json:"center"`
			Radius         float32  `json:"radius"`
			Falloff        float32  `json:"falloff"`
			ImpulsePerArea float32  `json:"impulse_per_area"`
			Mask           *uint64  `json:"mask"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Radius <= 0 {
			a.Radius = 5
		}
		if a.Falloff <= 0 {
			a.Falloff = a.Radius * 0.5
		}
		if a.ImpulsePerArea == 0 {
			a.ImpulsePerArea = 20
		}
		mask := uint64(catProp | catPlayer)
		if a.Mask != nil {
			mask = *a.Mask
		}
		def := box3d.DefaultExplosionDef()
		def.Position = a.Center.pos()
		def.Radius = a.Radius
		def.Falloff = a.Falloff
		def.ImpulsePerArea = a.ImpulsePerArea
		def.MaskBits = mask
		g.world.Explode(&def)
		g.burst(a.Center.rl(), colWarn, 24, 7, 0.12)
		g.shake = 0.35
		return true, nil
	case "physics.stats":
		bounds := g.world.GetBounds()
		return map[string]any{
			"awake_bodies": g.world.GetAwakeBodyCount(),
			"bodies":       len(g.entities),
			"props":        len(g.props),
			"bounds_min":   vecInfo(toRlV(bounds.LowerBound)),
			"bounds_max":   vecInfo(toRlV(bounds.UpperBound)),
		}, nil
	case "physics.set_worker_count":
		var a struct {
			Count int32 `json:"count"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.world.SetWorkerCount(int(a.Count))
		return true, nil
	case "physics.set_max_linear_speed":
		return nil, fmt.Errorf("physics.set_max_linear_speed is not supported by this box3d version")
	case "physics.create_constraint":
		return g.apiCreateConstraint(owner, args)
	case "physics.remove_constraint":
		var a struct {
			ID int32 `json:"id"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		for _, c := range g.modConstraints {
			if c.ID == a.ID {
				return g.removeConstraint(c), nil
			}
		}
		return false, nil
	case "physics.list_constraints":
		out := make([]ConstraintInfo, 0, len(g.modConstraints))
		for _, c := range g.modConstraints {
			out = append(out, ConstraintInfo{ID: c.ID, Type: c.Type, EntityA: c.EntityA, EntityB: c.EntityB, Owner: c.Owner})
		}
		return out, nil

	// ----- player ---------------------------------------------------------
	case "player.get":
		return g.playerInfo(), nil
	case "player.set_pos":
		var a struct {
			Position Vec3Info `json:"position"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.body.SetTransform(a.Position.pos(), g.player.body.GetRotation())
		g.player.body.SetAwake(true)
		return true, nil
	case "player.set_rot":
		var a struct {
			Yaw   float32    `json:"yaw"`
			Pitch float32    `json:"pitch"`
			Quat  *QuatInfo  `json:"rotation"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.yaw = a.Yaw
		g.player.pitch = a.Pitch
		if a.Quat != nil {
			g.player.body.SetTransform(g.player.body.GetPosition(), a.Quat.box())
		}
		return true, nil
	case "player.set_velocity":
		var a struct {
			Velocity Vec3Info `json:"velocity"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.body.SetLinearVelocity(a.Velocity.box())
		return true, nil
	case "player.set_fly":
		var a struct {
			Fly bool `json:"fly"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.fly = a.Fly
		if a.Fly {
			g.player.body.SetGravityScale(0)
			g.player.body.SetLinearVelocity(box3d.Vec3Zero)
		} else {
			g.player.body.SetGravityScale(1)
		}
		g.player.body.SetAwake(true)
		return true, nil
	case "player.set_speed_scale":
		var a struct {
			Scale float32 `json:"scale"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.speedScale = clampf(a.Scale, 0, 20)
		return g.player.speedScale, nil
	case "player.set_jump_scale":
		var a struct {
			Scale float32 `json:"scale"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.jumpScale = clampf(a.Scale, 0, 20)
		return g.player.jumpScale, nil
	case "player.set_health":
		var a struct {
			Health    float32 `json:"health"`
			MaxHealth float32 `json:"max_health"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		if a.MaxHealth > 0 {
			g.player.maxHealth = a.MaxHealth
		}
		if g.player.maxHealth <= 0 {
			g.player.maxHealth = 100
		}
		g.player.health = clampf(a.Health, 0, g.player.maxHealth)
		if g.player.health <= 0 {
			g.respawnPlayer()
		}
		return g.player.health, nil
	case "player.damage":
		var a struct {
			Amount float32 `json:"amount"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.health -= a.Amount
		if g.mods != nil {
			g.mods.FireEvent("player.damage", map[string]any{"damage": a.Amount, "health": g.player.health})
		}
		if g.player.health <= 0 {
			if g.mods != nil {
				g.mods.FireEvent("player.death", map[string]any{"health": g.player.health})
			}
			g.respawnPlayer()
		}
		return g.player.health, nil
	case "player.heal":
		var a struct {
			Amount float32 `json:"amount"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.health = clampf(g.player.health+a.Amount, 0, g.player.maxHealth)
		return g.player.health, nil
	case "player.respawn":
		g.respawnPlayer()
		return true, nil
	case "player.set_gravity_scale":
		var a struct {
			Scale float32 `json:"scale"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.player == nil {
			return false, nil
		}
		g.player.body.SetGravityScale(a.Scale)
		return true, nil

	// ----- world ----------------------------------------------------------
	case "world.set_sun":
		var a struct {
			Direction Vec3Info  `json:"direction"`
			Color     *ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		dir := a.Direction.rl()
		if lenRl(dir) > 1e-6 {
			g.sunDir = normRl(dir)
		}
		return vecInfo(g.sunDir), nil
	case "world.get_sun":
		return vecInfo(g.sunDir), nil
	case "world.set_sky":
		var a struct {
			Top    *ColorInfo `json:"top"`
			Bottom *ColorInfo `json:"bottom"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Top != nil {
			g.skyTop = a.Top.rl()
		}
		if a.Bottom != nil {
			g.skyBottom = a.Bottom.rl()
		}
		return map[string]any{"top": colorInfo(g.skyTop), "bottom": colorInfo(g.skyBottom)}, nil
	case "world.get_time":
		return g.time, nil
	case "world.set_time_scale":
		var a struct {
			Scale float32 `json:"scale"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.timeScale = float64(clampf(a.Scale, 0, 10))
		return g.timeScale, nil
	case "world.pause":
		g.state = statePaused
		return true, nil
	case "world.resume":
		if g.state == statePaused {
			g.resumePlaying()
		}
		return true, nil
	case "world.reset":
		var a struct {
			WithStarter *bool `json:"with_starter"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		starter := true
		if a.WithStarter != nil {
			starter = *a.WithStarter
		}
		g.resetWorld(starter, defaultSpawn, 0, -0.12, false)
		return true, nil
	case "world.set_gravity":
		var a struct {
			Scale *float32 `json:"scale"`
			Y     *float32 `json:"y"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Y != nil {
			g.world.SetGravity(bv(0, *a.Y, 0))
			g.settings.Gravity = clampf(*a.Y/baseGravity, 0, 2.5)
		} else if a.Scale != nil {
			g.setGravity(*a.Scale)
		}
		return g.world.GetGravity(), nil
	case "world.save":
		g.saveGame()
		return true, nil
	case "world.load":
		g.loadGame()
		return true, nil

	// ----- input ----------------------------------------------------------
	case "input.key_down":
		var a struct{ Key int32 `json:"key"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return rl.IsKeyDown(a.Key), nil
	case "input.key_pressed":
		var a struct{ Key int32 `json:"key"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return rl.IsKeyPressed(a.Key), nil
	case "input.key_released":
		var a struct{ Key int32 `json:"key"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return rl.IsKeyReleased(a.Key), nil
	case "input.mouse_down":
		var a struct{ Button int32 `json:"button"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return rl.IsMouseButtonDown(rl.MouseButton(a.Button)), nil
	case "input.mouse_pressed":
		var a struct{ Button int32 `json:"button"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return rl.IsMouseButtonPressed(rl.MouseButton(a.Button)), nil
	case "input.mouse_released":
		var a struct{ Button int32 `json:"button"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return rl.IsMouseButtonReleased(rl.MouseButton(a.Button)), nil
	case "input.mouse_pos":
		p := rl.GetMousePosition()
		return Vec3Info{X: p.X, Y: p.Y}, nil
	case "input.mouse_delta":
		d := rl.GetMouseDelta()
		return Vec3Info{X: d.X, Y: d.Y}, nil
	case "input.wheel":
		return rl.GetMouseWheelMove(), nil
	case "input.is_focused":
		return rl.IsWindowFocused(), nil
	case "input.set_cursor_locked":
		var a struct{ Locked bool `json:"locked"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Locked {
			rl.DisableCursor()
		} else {
			rl.EnableCursor()
		}
		return true, nil

	// ----- ui -------------------------------------------------------------
	case "ui.toast":
		var a struct {
			Message string    `json:"message"`
			Color   ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		c := colAccent
		if a.Color.provided() {
			c = a.Color.rl()
		}
		g.toast(a.Message, c)
		return true, nil
	case "ui.draw_text":
		var a struct {
			Text    string    `json:"text"`
			X       float32   `json:"x"`
			Y       float32   `json:"y"`
			Size    float32   `json:"size"`
			Spacing float32   `json:"spacing"`
			Color   ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Size <= 0 {
			a.Size = 16
		}
		if a.Spacing <= 0 {
			a.Spacing = 1
		}
		g.modDrawQueue = append(g.modDrawQueue, ModDrawCmd{Kind: "text", Text: a.Text, X: a.X, Y: a.Y, Size: a.Size, Thickness: a.Spacing, Color: a.Color.rl()})
		return true, nil
	case "ui.draw_rect":
		var a struct {
			X       float32   `json:"x"`
			Y       float32   `json:"y"`
			W       float32   `json:"w"`
			H       float32   `json:"h"`
			Color   ColorInfo `json:"color"`
			Filled  *bool     `json:"filled"`
			Thickness float32 `json:"thickness"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		filled := true
		if a.Filled != nil {
			filled = *a.Filled
		}
		g.modDrawQueue = append(g.modDrawQueue, ModDrawCmd{Kind: "rect", X: a.X, Y: a.Y, W: a.W, H: a.H, Color: a.Color.rl(), Filled: filled, Thickness: a.Thickness})
		return true, nil
	case "ui.draw_line":
		var a struct {
			X0, Y0, X1, Y1 float32
			Thickness      float32
			Color          ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Thickness <= 0 {
			a.Thickness = 1
		}
		g.modDrawQueue = append(g.modDrawQueue, ModDrawCmd{Kind: "line", X: a.X0, Y: a.Y0, X2: a.X1, Y2: a.Y1, Color: a.Color.rl(), Thickness: a.Thickness})
		return true, nil
	case "ui.draw_circle":
		var a struct {
			X, Y      float32
			Radius    float32
			Color     ColorInfo `json:"color"`
			Filled    *bool     `json:"filled"`
			Thickness float32   `json:"thickness"`
			Segments  int32     `json:"segments"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		filled := true
		if a.Filled != nil {
			filled = *a.Filled
		}
		g.modDrawQueue = append(g.modDrawQueue, ModDrawCmd{Kind: "circle", X: a.X, Y: a.Y, Radius: a.Radius, Color: a.Color.rl(), Filled: filled, Thickness: a.Thickness, Segments: a.Segments})
		return true, nil
	case "ui.draw_triangle":
		var a struct {
			X0, Y0, X1, Y1, X2, Y2 float32
			Color                  ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.modDrawQueue = append(g.modDrawQueue, ModDrawCmd{Kind: "triangle", X: a.X0, Y: a.Y0, X2: a.X1, Y2: a.Y1, X3: a.X2, Y3: a.Y2, Color: a.Color.rl()})
		return true, nil
	case "ui.measure_text":
		var a struct {
			Text    string  `json:"text"`
			Size    float32 `json:"size"`
			Spacing float32 `json:"spacing"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Size <= 0 {
			a.Size = 16
		}
		if a.Spacing <= 0 {
			a.Spacing = 1
		}
		return g.textWidthSpaced(a.Text, a.Size, a.Spacing), nil
	case "ui.screen_size":
		return map[string]float32{"width": g.uiWidth(), "height": g.uiHeight()}, nil
	case "ui.clear":
		g.modDrawQueue = nil
		return true, nil

	// ----- render ---------------------------------------------------------
	case "render.draw_line_3d":
		var a struct {
			From, To  Vec3Info
			Color     ColorInfo `json:"color"`
			Thickness float32   `json:"thickness"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Thickness <= 0 {
			a.Thickness = 1
		}
		g.modDebugQueue = append(g.modDebugQueue, ModDebugCmd{Kind: "line", From: a.From.rl(), To: a.To.rl(), Color: a.Color.rl(), Thickness: a.Thickness})
		return true, nil
	case "render.draw_box_3d":
		var a struct {
			Center Vec3Info  `json:"center"`
			Size   Vec3Info  `json:"size"`
			Color  ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.modDebugQueue = append(g.modDebugQueue, ModDebugCmd{Kind: "box", Center: a.Center.rl(), Size: a.Size.rl(), Color: a.Color.rl()})
		return true, nil
	case "render.draw_sphere_3d":
		var a struct {
			Center   Vec3Info  `json:"center"`
			Radius   float32   `json:"radius"`
			Color    ColorInfo `json:"color"`
			Wire     bool      `json:"wire"`
			Segments int32     `json:"segments"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		kind := "sphere"
		if a.Wire {
			kind = "sphere_wire"
		}
		g.modDebugQueue = append(g.modDebugQueue, ModDebugCmd{Kind: kind, Center: a.Center.rl(), Radius: a.Radius, Color: a.Color.rl(), Segments: a.Segments})
		return true, nil
	case "render.draw_ray_3d":
		var a struct {
			Origin    Vec3Info  `json:"origin"`
			Direction Vec3Info  `json:"direction"`
			Length    float32   `json:"length"`
			Color     ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		d := normRl(a.Direction.rl())
		to := rlv(a.Origin.X+d.X*a.Length, a.Origin.Y+d.Y*a.Length, a.Origin.Z+d.Z*a.Length)
		g.modDebugQueue = append(g.modDebugQueue, ModDebugCmd{Kind: "line", From: a.Origin.rl(), To: to, Color: a.Color.rl(), Thickness: 1.5})
		return true, nil
	case "render.draw_circle_3d":
		var a struct {
			Center   Vec3Info  `json:"center"`
			Normal   Vec3Info  `json:"normal"`
			Radius   float32   `json:"radius"`
			Color    ColorInfo `json:"color"`
			Segments int32     `json:"segments"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.modDebugQueue = append(g.modDebugQueue, ModDebugCmd{Kind: "circle", Center: a.Center.rl(), Normal: a.Normal.rl(), Radius: a.Radius, Color: a.Color.rl(), Segments: a.Segments})
		return true, nil
	case "render.clear_debug":
		g.modDebugQueue = nil
		return true, nil
	case "render.world_to_screen":
		var a struct {
			Position Vec3Info `json:"position"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		p := rl.GetWorldToScreen(a.Position.rl(), g.camera)
		return Vec3Info{X: p.X, Y: p.Y}, nil
	case "render.screen_to_world_ray":
		var a struct {
			X, Y float32
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		ray := rl.GetScreenToWorldRay(rl.Vector2{X: a.X, Y: a.Y}, g.camera)
		return map[string]any{"origin": vecInfo(ray.Position), "direction": vecInfo(ray.Direction)}, nil
	case "render.set_fov":
		var a struct {
			FOV float32 `json:"fov"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.settings.FOV = clampf(a.FOV, 30, 120)
		g.settingsDirty = true
		return g.settings.FOV, nil
	case "render.set_shadows":
		var a struct {
			Enabled bool `json:"enabled"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.settings.Shadows = a.Enabled
		g.settingsDirty = true
		return true, nil
	case "render.screen_size":
		return map[string]int32{"width": int32(rl.GetScreenWidth()), "height": int32(rl.GetScreenHeight())}, nil

	// ----- particle -------------------------------------------------------
	case "particle.spawn":
		return g.apiSpawnParticle(owner, args)
	case "particle.burst":
		var a struct {
			Position Vec3Info  `json:"position"`
			Color    ColorInfo `json:"color"`
			Count    int32     `json:"count"`
			Speed    float32   `json:"speed"`
			Life     float32   `json:"life"`
			Size     float32   `json:"size"`
			Gravity  *float32  `json:"gravity"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		n := a.Count
		if n <= 0 {
			n = 12
		}
		speed := a.Speed
		if speed <= 0 {
			speed = 4
		}
		life := a.Life
		if life <= 0 {
			life = 0.4
		}
		size := a.Size
		if size <= 0 {
			size = 0.08
		}
		gravity := float32(9)
		if a.Gravity != nil {
			gravity = *a.Gravity
		}
		for i := int32(0); i < n; i++ {
			dir := rlv(g.rng.Float32()*2-1, g.rng.Float32()*2-1, g.rng.Float32()*2-1)
			dir = normRl(dir)
			g.spawnModParticle(a.Position.rl(), rlv(dir.X*speed, dir.Y*speed, dir.Z*speed), a.Color.rl(), size, life, gravity, owner)
		}
		return n, nil
	case "particle.clear":
		g.modParticles = nil
		return true, nil
	case "particle.count":
		return len(g.modParticles), nil

	// ----- audio ----------------------------------------------------------
	case "audio.load_sound":
		return g.apiLoadSound(args)
	case "audio.play_sound":
		var a struct {
			Name   string  `json:"name"`
			Volume *float32 `json:"volume"`
			Pitch  *float32 `json:"pitch"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		s, ok := g.modSounds[a.Name]
		if !ok {
			return false, fmt.Errorf("sound %q is not loaded", a.Name)
		}
		if a.Volume != nil {
			rl.SetSoundVolume(s, *a.Volume)
		}
		if a.Pitch != nil {
			rl.SetSoundPitch(s, *a.Pitch)
		}
		rl.PlaySound(s)
		return true, nil
	case "audio.stop_sound":
		var a struct{ Name string `json:"name"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if s, ok := g.modSounds[a.Name]; ok {
			rl.StopSound(s)
		}
		return true, nil
	case "audio.load_music":
		return g.apiLoadMusic(args)
	case "audio.play_music":
		var a struct {
			Name   string   `json:"name"`
			Volume *float32 `json:"volume"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		m, ok := g.modMusic[a.Name]
		if !ok {
			return false, fmt.Errorf("music %q is not loaded", a.Name)
		}
		vol := float32(1)
		if a.Volume != nil {
			vol = *a.Volume
		}
		g.modMusicVol[a.Name] = vol
		rl.SetMusicVolume(m, vol)
		rl.PlayMusicStream(m)
		return true, nil
	case "audio.stop_music":
		var a struct{ Name string `json:"name"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if m, ok := g.modMusic[a.Name]; ok {
			rl.StopMusicStream(m)
		}
		return true, nil
	case "audio.set_music_volume":
		var a struct {
			Name   string  `json:"name"`
			Volume float32 `json:"volume"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if m, ok := g.modMusic[a.Name]; ok {
			g.modMusicVol[a.Name] = a.Volume
			rl.SetMusicVolume(m, a.Volume)
		}
		return true, nil
	case "audio.is_ready":
		return rl.IsAudioDeviceReady(), nil
	case "audio.unload":
		var a struct{ Name string `json:"name"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if s, ok := g.modSounds[a.Name]; ok {
			rl.UnloadSound(s)
			delete(g.modSounds, a.Name)
		}
		if m, ok := g.modMusic[a.Name]; ok {
			rl.UnloadMusicStream(m)
			delete(g.modMusic, a.Name)
			delete(g.modMusicVol, a.Name)
		}
		return true, nil

	// ----- weapon ---------------------------------------------------------
	case "weapon.register":
		return g.apiRegisterWeapon(owner, args)
	case "weapon.unregister":
		var a struct{ ID int32 `json:"id"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return g.unregisterWeapon(a.ID, owner), nil
	case "weapon.list":
		out := make([]WeaponInfo, 0, len(g.modWeapons))
		for _, w := range g.modWeapons {
			out = append(out, weaponInfo(w))
		}
		return out, nil
	case "weapon.select":
		var a struct{ ID int32 `json:"id"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.findWeapon(a.ID) == nil {
			return false, fmt.Errorf("weapon %d does not exist", a.ID)
		}
		g.activeModWeapon = a.ID
		return true, nil
	case "weapon.get_active":
		if w := g.findWeapon(g.activeModWeapon); w != nil {
			return weaponInfo(w), nil
		}
		return nil, nil
	case "weapon.fire":
		var a struct {
			ID *int32 `json:"id"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		id := g.activeModWeapon
		if a.ID != nil {
			id = *a.ID
		}
		return g.fireWeapon(id, owner)
	case "weapon.set_ammo":
		var a struct {
			ID   int32 `json:"id"`
			Ammo int32 `json:"ammo"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		w := g.findWeapon(a.ID)
		if w == nil {
			return false, nil
		}
		w.Ammo = int32(clampInt(int(a.Ammo), 0, int(w.MaxAmmo)))
		return w.Ammo, nil
	case "weapon.add_ammo":
		var a struct {
			ID     int32 `json:"id"`
			Amount int32 `json:"amount"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		w := g.findWeapon(a.ID)
		if w == nil {
			return false, nil
		}
		w.Ammo = int32(clampInt(int(w.Ammo+a.Amount), 0, int(w.MaxAmmo)))
		return w.Ammo, nil
	case "weapon.get_ammo":
		var a struct{ ID int32 `json:"id"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if w := g.findWeapon(a.ID); w != nil {
			return w.Ammo, nil
		}
		return 0, nil

	// ----- tool -----------------------------------------------------------
	case "tool.register":
		return g.apiRegisterTool(owner, args)
	case "tool.unregister":
		var a struct{ ID int32 `json:"id"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return g.unregisterTool(a.ID, owner), nil
	case "tool.list":
		out := make([]ToolInfo, 0, len(g.modTools))
		for _, t := range g.modTools {
			out = append(out, toolInfo(t))
		}
		return out, nil
	case "tool.select":
		var a struct{ ID int32 `json:"id"` }
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.findTool(a.ID) == nil {
			return false, fmt.Errorf("tool %d does not exist", a.ID)
		}
		g.activeModTool = a.ID
		g.activeModWeapon = -1
		return true, nil
	case "tool.get_active":
		if t := g.findTool(g.activeModTool); t != nil {
			return toolInfo(t), nil
		}
		return nil, nil
	case "tool.primary", "tool.secondary", "tool.reload":
		return g.fireToolAction(cmd, args, owner)
	case "tool.set_viewmodel":
		// The engine renders a procedural tool gun; mods can tint it.
		var a struct {
			Color ColorInfo `json:"color"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.toolModColor = a.Color.rl()
		return true, nil

	// ----- hook -----------------------------------------------------------
	case "hook.add":
		var a struct {
			Event string `json:"event"`
			Hook  string `json:"hook"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.mods != nil {
			return g.mods.AddHook(owner, a.Event, a.Hook), nil
		}
		return false, nil
	case "hook.remove":
		var a struct {
			Event string `json:"event"`
			Hook  string `json:"hook"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.mods != nil {
			return g.mods.RemoveHook(owner, a.Event, a.Hook), nil
		}
		return false, nil
	case "hook.list":
		if g.mods != nil {
			return g.mods.ListHooks(owner), nil
		}
		return map[string][]string{}, nil
	case "hook.fire":
		var a struct {
			Event string          `json:"event"`
			Data  json.RawMessage `json:"data"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.mods != nil {
			g.mods.FireEvent(a.Event, a.Data)
		}
		return true, nil

	// ----- resource -------------------------------------------------------
	case "resource.read_text":
		return g.apiResourceRead(owner, args)
	case "resource.write_text":
		return g.apiResourceWrite(owner, args)
	case "resource.exists":
		return g.apiResourceExists(owner, args)
	case "resource.delete":
		return g.apiResourceDelete(owner, args)
	case "resource.list":
		return g.apiResourceList(owner, args)
	case "resource.data_dir":
		return g.modDataDir(owner), nil

	// ----- content --------------------------------------------------------
	case "content.list_presets":
		out := make([]PresetInfo, 0, len(presets))
		for i, p := range presets {
			out = append(out, presetInfo(int32(i), p))
		}
		return out, nil
	case "content.add_preset":
		return g.apiAddPreset(args)
	case "content.remove_preset":
		var a struct {
			Index int32  `json:"index"`
			Name  string `json:"name"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return g.removePreset(a.Index, a.Name), nil

	// ----- settings -------------------------------------------------------
	case "settings.get":
		return g.settings, nil
	case "settings.set":
		var a struct {
			Gravity     *float32 `json:"gravity"`
			Sensitivity *float32 `json:"sensitivity"`
			FOV         *float32 `json:"fov"`
			UIScale     *float32 `json:"ui_scale"`
			InvertY     *bool    `json:"invert_y"`
			Shadows     *bool    `json:"shadows"`
			RawInput    *bool    `json:"raw_input"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if a.Gravity != nil {
			g.setGravity(*a.Gravity)
		}
		if a.Sensitivity != nil {
			g.settings.Sensitivity = clampf(*a.Sensitivity, 0, 1)
		}
		if a.FOV != nil {
			g.settings.FOV = clampf(*a.FOV, 60, 105)
		}
		if a.UIScale != nil {
			g.settings.UIScale = clampf(*a.UIScale, 0.75, 1.5)
		}
		if a.InvertY != nil {
			g.settings.InvertY = *a.InvertY
		}
		if a.Shadows != nil {
			g.settings.Shadows = *a.Shadows
		}
		if a.RawInput != nil {
			g.settings.RawInput = *a.RawInput
		}
		g.settingsDirty = true
		return g.settings, nil
	case "settings.save":
		if err := g.saveSettings(); err != nil {
			return nil, err
		}
		return true, nil

	// ----- net (local mod transport) -------------------------------------
	case "net.broadcast":
		var a struct {
			Channel string          `json:"channel"`
			Data    json.RawMessage `json:"data"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		if g.mods != nil {
			g.mods.FireEvent("net."+a.Channel, map[string]any{"from": owner, "data": a.Data})
		}
		return true, nil

	// ----- camera ---------------------------------------------------------
	case "camera.shake":
		var a struct {
			Amount float32 `json:"amount"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		g.shake = float64(clampf(a.Amount, 0, 5))
		return true, nil
	case "camera.get":
		return map[string]any{
			"position": vecInfo(g.camera.Position),
			"target":   vecInfo(g.camera.Target),
			"up":       vecInfo(g.camera.Up),
			"fov":      g.camera.Fovy,
		}, nil
	}
	return nil, fmt.Errorf("unknown command %q", cmd)
}

// ---------------------------------------------------------------------------
// Engine state helpers
// ---------------------------------------------------------------------------

func (g *Game) engineInfo() EngineInfo {
	state := "playing"
	switch g.state {
	case statePaused:
		state = "paused"
	case stateHelp:
		state = "help"
	case stateSettings:
		state = "settings"
	case stateSpawn:
		state = "spawn"
	case stateToolMenu:
		state = "tools"
	}
	return EngineInfo{
		Game:        "fsbox",
		Version:     "0.3.0",
		APIVersion:  ModAPIVersion,
		Time:        g.time,
		Frame:       g.frame,
		FPS:         int32(rl.GetFPS()),
		State:       state,
		Paused:      g.state == statePaused,
		TimeScale:   g.timeScale,
		EntityCount: len(g.entities),
		PropCount:   len(g.props),
		Player:      g.playerInfo(),
	}
}

func (g *Game) playerInfo() PlayerInfo {
	if g.player == nil {
		return PlayerInfo{}
	}
	p := g.player.body.GetPosition()
	q := g.player.body.GetRotation()
	v := g.player.body.GetLinearVelocity()
	eye := g.eyePosition()
	f := g.cameraForward()
	return PlayerInfo{
		Position:   Vec3Info{X: p.X, Y: p.Y, Z: p.Z},
		Rotation:   quatInfo(q),
		Velocity:   Vec3Info{X: v.X, Y: v.Y, Z: v.Z},
		Yaw:        g.player.yaw,
		Pitch:      g.player.pitch,
		Fly:        g.player.fly,
		Grounded:   g.player.grounded,
		Health:     g.player.health,
		MaxHealth:  g.player.maxHealth,
		SpeedScale: g.player.speedScale,
		JumpScale:  g.player.jumpScale,
		Eye:        vecInfo(eye),
		Forward:    vecInfo(f),
		Color:      colorInfo(colAccent),
	}
}

func presetInfo(i int32, p ShapePreset) PresetInfo {
	return PresetInfo{
		Index:       i,
		Name:        p.Name,
		Category:    p.Category,
		Kind:        p.Kind.String(),
		Half:        Vec3Info{X: p.Half.X, Y: p.Half.Y, Z: p.Half.Z},
		Radius:      p.Radius,
		Height:      p.Height,
		Density:     p.Density,
		Friction:    p.Friction,
		Restitution: p.Restitution,
		Color:       colorInfo(p.Color),
		HasModel:    p.HasModel,
	}
}

func weaponInfo(w *ModWeapon) WeaponInfo {
	if w == nil {
		return WeaponInfo{}
	}
	return WeaponInfo{
		ID: w.ID, Name: w.Name, Type: w.Type.String(), Damage: w.Damage, Range: w.Range,
		Ammo: w.Ammo, MaxAmmo: w.MaxAmmo, FireRate: w.FireRate, Recoil: w.Recoil,
		Spread: w.Spread, ProjectileSpeed: w.ProjectileSpeed, ProjectileLife: w.ProjectileLife,
		ProjectileSize: w.ProjectileSize, Color: colorInfo(w.Color), Owner: w.Owner,
	}
}

func toolInfo(t *ModTool) ToolInfo {
	if t == nil {
		return ToolInfo{}
	}
	return ToolInfo{
		ID: t.ID, Name: t.Name, PrimaryDesc: t.PrimaryDesc, SecondaryDesc: t.SecondaryDesc,
		ReloadDesc: t.ReloadDesc, Color: colorInfo(t.Color), PrimaryHook: t.PrimaryHook,
		SecondaryHook: t.SecondaryHook, ReloadHook: t.ReloadHook, Owner: t.Owner,
	}
}

func (g *Game) raycastInfo(res box3d.RayResult, maxDist float32) RaycastInfo {
	if !res.Hit {
		return RaycastInfo{}
	}
	info := RaycastInfo{
		Hit:      true,
		Point:    vecInfo(toRlP(res.Point)),
		Normal:   vecInfo(toRlV(res.Normal)),
		Fraction: res.Fraction,
		Distance: res.Fraction * maxDist,
	}
	if res.ShapeId.GetBody().IsValid() {
		if e := g.byBody[res.ShapeId.GetBody()]; e != nil {
			info.EntityID = e.ID
			ei := g.entityInfo(e)
			info.Entity = &ei
		}
	}
	return info
}

// ---------------------------------------------------------------------------
// Small slice helpers
// ---------------------------------------------------------------------------

func appendUnique(list []string, value string) []string {
	if value == "" {
		return list
	}
	for _, v := range list {
		if v == value {
			return list
		}
	}
	return append(list, value)
}

func removeString(list []string, value string) []string {
	out := list[:0]
	for _, v := range list {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Per-frame mod systems
// ---------------------------------------------------------------------------

func (g *Game) fireInputEvents() {
	if g.mods == nil {
		return
	}
	for key, events := range g.modInputHooks {
		pressed := rl.IsKeyPressed(key)
		released := rl.IsKeyReleased(key)
		if !pressed && !released {
			continue
		}
		data := map[string]any{"key": key, "pressed": pressed, "released": released, "down": rl.IsKeyDown(key)}
		for _, name := range events {
			g.mods.FireEvent(name, data)
		}
	}
	for button, events := range g.modMouseHooks {
		pressed := rl.IsMouseButtonPressed(rl.MouseButton(button))
		released := rl.IsMouseButtonReleased(rl.MouseButton(button))
		if !pressed && !released {
			continue
		}
		data := map[string]any{"button": button, "pressed": pressed, "released": released, "down": rl.IsMouseButtonDown(rl.MouseButton(button))}
		for _, name := range events {
			g.mods.FireEvent(name, data)
		}
	}
}

func (g *Game) spawnModParticle(pos, vel rl.Vector3, c rl.Color, size, life, gravity float32, owner string) *ModParticle {
	if size <= 0 {
		size = 0.08
	}
	if life <= 0 {
		life = 0.5
	}
	g.modParticleLast++
	p := &ModParticle{
		ID: g.modParticleLast, Pos: pos, Vel: vel, Color: c,
		Size: size, Life: life, MaxLife: life, Gravity: gravity,
	}
	g.modParticles = append(g.modParticles, p)
	return p
}

func (g *Game) updateModParticles(dt float32) {
	if len(g.modParticles) == 0 {
		return
	}
	dst := g.modParticles[:0]
	for _, p := range g.modParticles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.Vel.Y -= p.Gravity * dt
		p.Pos.X += p.Vel.X * dt
		p.Pos.Y += p.Vel.Y * dt
		p.Pos.Z += p.Vel.Z * dt
		dst = append(dst, p)
	}
	g.modParticles = dst
}

func (g *Game) updateMusic() {
	for _, m := range g.modMusic {
		rl.UpdateMusicStream(m)
	}
}

func (g *Game) drawModUI() {
	if g.mods != nil && len(g.modHUDHooks) > 0 {
		g.mods.FireEvent("ui.draw", g.engineInfo())
	}
	for _, cmd := range g.modDrawQueue {
		switch cmd.Kind {
		case "text":
			g.drawTextSpaced(cmd.Text, cmd.X, cmd.Y, cmd.Size, cmd.Thickness, cmd.Color)
		case "rect":
			if cmd.Filled {
				rl.DrawRectangleRec(rl.Rectangle{X: cmd.X, Y: cmd.Y, Width: cmd.W, Height: cmd.H}, cmd.Color)
			} else {
				t := cmd.Thickness
				if t <= 0 {
					t = 1
				}
				rl.DrawRectangleLinesEx(rl.Rectangle{X: cmd.X, Y: cmd.Y, Width: cmd.W, Height: cmd.H}, t, cmd.Color)
			}
		case "line":
			t := cmd.Thickness
			if t <= 0 {
				t = 1
			}
			rl.DrawLineEx(rl2(cmd.X, cmd.Y), rl2(cmd.X2, cmd.Y2), t, cmd.Color)
		case "circle":
			if cmd.Filled {
				rl.DrawCircleV(rl2(cmd.X, cmd.Y), cmd.Radius, cmd.Color)
			} else {
				rl.DrawCircleLines(int32(cmd.X), int32(cmd.Y), cmd.Radius, cmd.Color)
			}
		case "triangle":
			rl.DrawTriangle(rl2(cmd.X, cmd.Y), rl2(cmd.X2, cmd.Y2), rl2(cmd.X3, cmd.Y3), cmd.Color)
		}
	}
	g.modDrawQueue = g.modDrawQueue[:0]
}

func (g *Game) drawModDebug() {
	for _, cmd := range g.modDebugQueue {
		switch cmd.Kind {
		case "line":
			rl.DrawLine3D(cmd.From, cmd.To, cmd.Color)
		case "box":
			half := rlv(cmd.Size.X*0.5, cmd.Size.Y*0.5, cmd.Size.Z*0.5)
			drawBoxEdges(cmd.Center, box3d.QuatIdentity, toBv(half), cmd.Color)
		case "sphere":
			rl.DrawSphereEx(cmd.Center, cmd.Radius, 8, 12, cmd.Color)
		case "sphere_wire":
			rl.DrawSphereWires(cmd.Center, cmd.Radius, 8, 12, cmd.Color)
		case "circle":
			drawGreatCircle(cmd.Center, normRl(cmd.Normal), cmd.Radius, cmd.Color)
		}
	}
	g.modDebugQueue = g.modDebugQueue[:0]
}

func (g *Game) drawModParticles() {
	for _, p := range g.modParticles {
		a := uint8(255 * clampf(p.Life/p.MaxLife, 0, 1))
		rl.DrawSphereEx(p.Pos, p.Size, 4, 6, withAlpha(p.Color, a))
	}
}

// ---------------------------------------------------------------------------
// Mod data directory
// ---------------------------------------------------------------------------

func sanitizeModID(id string) string {
	if id == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || out == "." || out == ".." {
		return "unknown"
	}
	return out
}

func (g *Game) modDataDir(owner string) string {
	return filepath.Join("ModData", sanitizeModID(owner))
}

func safeModPath(owner, rel string) (string, error) {
	root := filepath.Clean(filepath.Join("ModData", sanitizeModID(owner)))
	full := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes mod data directory")
	}
	return full, nil
}

// ---------------------------------------------------------------------------
// Misc math used by API implementations
// ---------------------------------------------------------------------------

func degToRad(d float32) float32 {
	return d * float32(math.Pi) / 180
}
