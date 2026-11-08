package main

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
	eyeHeight    = 0.68
)

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
	fly        bool

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
		body:  body,
		yaw:   yaw,
		pitch: pitch,
		fly:   fly,
		spawn: spawn,
	}
}

func (p *Player) tick(g *Game, dt float32) {
	p.updateGrounded(g)

	if p.jumpBuffer > 0 {
		p.jumpBuffer--
	}

	var f, r float32
	if rl.IsKeyDown(rl.KeyW) {
		f += 1
	}
	if rl.IsKeyDown(rl.KeyS) {
		f -= 1
	}
	if rl.IsKeyDown(rl.KeyD) {
		r += 1
	}
	if rl.IsKeyDown(rl.KeyA) {
		r -= 1
	}

	fwd := g.forwardFlat()
	right := g.rightFlat()
	move := rlv(fwd.X*f+right.X*r, 0, fwd.Z*f+right.Z*r)
	if l := lenRl(move); l > 1 {
		move = rlv(move.X/l, 0, move.Z/l)
	}

	sprint := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	v := p.body.GetLinearVelocity()

	if p.fly {
		speed := float32(flySpeed)
		if sprint {
			speed = flySprint
		}
		desired := rlv(move.X*speed, 0, move.Z*speed)
		if rl.IsKeyDown(rl.KeySpace) {
			desired.Y += flyVertical
		}
		if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
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
		v.Y = jumpVelocity
		p.coyote = 0
		p.jumpBuffer = 0
		p.grounded = false
	}

	speed := float32(walkSpeed)
	if sprint {
		speed = sprintSpeed
	}
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
}
