package game

import (
	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	walkSpeed    = 5.6
	sprintSpeed  = 9.3
	flySpeed     = 12.0
	flySprint    = 19.0
	flyVertical  = 9.0
	jumpVelocity = 8.6
	eyeHeight    = 0.55
)

// InputState is one frame (or one network packet) of player intent. Local play
// builds it from the keyboard and mouse; the dedicated server builds it from
// the latest client input message. Keeping movement in one place means local
// and networked players use exactly the same physics tuning.
type InputState struct {
	Forward float32
	Right   float32
	Jump    bool
	Down    bool
	Sprint  bool
	Yaw     float32
	Pitch   float32
}

// localInput samples the keyboard into an InputState. Look angles are copied
// from the player so the same movement code works for local and server players.
func localInput(p *Player) InputState {
	in := InputState{Yaw: p.yaw, Pitch: p.pitch}
	if rl.IsKeyDown(rl.KeyW) {
		in.Forward += 1
	}
	if rl.IsKeyDown(rl.KeyS) {
		in.Forward -= 1
	}
	if rl.IsKeyDown(rl.KeyD) {
		in.Right += 1
	}
	if rl.IsKeyDown(rl.KeyA) {
		in.Right -= 1
	}
	in.Jump = rl.IsKeyDown(rl.KeySpace)
	in.Down = rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	in.Sprint = rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	return in
}

// Player is a dynamic capsule with locked rotation. Box3D resolves all of its
// collisions; we only steer velocity, which keeps the character physics-driven
// instead of using a separate AABB mover.
type Player struct {
	body box3d.BodyID

	yaw   float32
	pitch float32

	grounded   bool
	coyote     int
	jumpBuffer int
	jumpHeld   bool
	fly        bool

	// Addon-controllable gameplay modifiers and health. The defaults match the
	// original movement tuning exactly (1.0 multipliers).
	speedScale float32
	jumpScale  float32
	health     float32
	maxHealth  float32

	spawn box3d.Pos
}

func newPlayer(w box3d.WorldID, spawn box3d.Pos, yaw, pitch float32, fly bool) *Player {
	bd := box3d.DefaultBodyDef()
	bd.Type = box3d.DynamicBody
	bd.Position = spawn
	bd.Rotation = box3d.QuatIdentity
	bd.Name = "Player"
	bd.EnableSleep = false
	bd.EnableContactRecycling = false
	bd.MotionLocks = box3d.MotionLocks{
		AngularX: true,
		AngularY: true,
		AngularZ: true,
	}
	if fly {
		bd.GravityScale = 0
	}

	body := w.CreateBody(&bd)

	sd := box3d.DefaultShapeDef()
	sd.Density = 120
	sd.BaseMaterial = box3d.DefaultSurfaceMaterial()
	sd.BaseMaterial.Friction = 0.42
	sd.BaseMaterial.Restitution = 0.0
	sd.Filter = playerFilter()

	capsule := box3d.Capsule{
		Center1: box3d.Vec3{X: 0, Y: -0.5, Z: 0},
		Center2: box3d.Vec3{X: 0, Y: 0.5, Z: 0},
		Radius:  0.35,
	}
	body.CreateCapsuleShape(&sd, &capsule)

	return &Player{
		body:       body,
		yaw:        yaw,
		pitch:      pitch,
		fly:        fly,
		speedScale: 1,
		jumpScale:  1,
		health:     100,
		maxHealth:  100,
		spawn:      spawn,
	}
}

func (p *Player) tick(g *Game, dt float32) {
	p.tickInput(g, localInput(p), dt)
}

// tickInput applies one step of movement from an InputState. The dedicated
// server calls this for every connected client using the latest packet; local
// play calls it through tick() with keyboard state.
func (p *Player) tickInput(g *Game, in InputState, dt float32) {
	p.yaw = in.Yaw
	p.pitch = clampf(in.Pitch, -1.52, 1.52)
	p.updateGrounded(g)

	// A fresh jump press (edge) fills the buffer. Holding space does not
	// auto-bounce; the player has to release and press again.
	if in.Jump && !p.jumpHeld {
		p.jumpBuffer = 8
	}
	p.jumpHeld = in.Jump
	if p.jumpBuffer > 0 {
		p.jumpBuffer--
	}

	move := rlv(0, 0, 0)
	fwd := forwardFlatYaw(p.yaw)
	right := rightFlatYaw(p.yaw)
	move.X = fwd.X*in.Forward + right.X*in.Right
	move.Z = fwd.Z*in.Forward + right.Z*in.Right
	if l := lenRl(move); l > 1 {
		move = rlv(move.X/l, 0, move.Z/l)
	}

	v := p.body.GetLinearVelocity()

	if p.fly {
		speed := float32(flySpeed)
		if in.Sprint {
			speed = flySprint
		}
		speed *= p.speedScale
		desired := rlv(move.X*speed, 0, move.Z*speed)
		if in.Jump {
			desired.Y += flyVertical
		}
		if in.Down {
			desired.Y -= flyVertical
		}
		blend := smoothing(9, dt)
		v.X = lerpf(v.X, desired.X, blend)
		v.Y = lerpf(v.Y, desired.Y, blend)
		v.Z = lerpf(v.Z, desired.Z, blend)
		p.body.SetLinearVelocity(v)
		return
	}

	if p.jumpBuffer > 0 && p.coyote > 0 {
		v.Y = jumpVelocity * p.jumpScale
		p.coyote = 0
		p.jumpBuffer = 0
		p.grounded = false
	}

	speed := float32(walkSpeed)
	if in.Sprint {
		speed = sprintSpeed
	}
	speed *= p.speedScale
	// Preserve horizontal momentum for bunny hopping: never slow the player
	// down below their current speed while they are moving and jumping.
	horizSpeed := vlen(v.X, 0, v.Z)
	if horizSpeed > speed {
		speed = horizSpeed
	}
	target := rlv(move.X*speed, 0, move.Z*speed)

	rate := float32(4.0)
	if p.grounded {
		rate = 18.0
	}
	blend := smoothing(rate, dt)
	v.X = lerpf(v.X, target.X, blend)
	v.Z = lerpf(v.Z, target.Z, blend)
	p.body.SetLinearVelocity(v)
}

func (p *Player) updateGrounded(g *Game) {
	pos := p.body.GetPosition()
	origin := box3d.Pos{X: pos.X, Y: pos.Y + 0.2, Z: pos.Z}
	res := g.world.CastRayClosest(origin, box3d.Vec3{Y: -1.35}, queryFilter(catStatic|catProp))
	wasGrounded := p.grounded
	p.grounded = res.Hit && res.Normal.Y > 0.45
	if p.grounded {
		p.coyote = 6
	} else if p.coyote > 0 {
		p.coyote--
	}
	if p.grounded && !wasGrounded && p.fly {
		// Landing in fly mode is fine; no special handling.
		_ = wasGrounded
	}
}

func (p *Player) respawn(g *Game) {
	p.body.SetTransform(p.spawn, box3d.QuatIdentity)
	p.body.SetLinearVelocity(box3d.Vec3Zero)
	p.body.SetAngularVelocity(box3d.Vec3Zero)
	p.body.SetAwake(true)
	p.grounded = false
	p.coyote = 0
	p.jumpBuffer = 0
	p.jumpHeld = false
}
