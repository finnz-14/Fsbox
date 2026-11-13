package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// netClient is the transport side of a connection to a dedicated server. All
// Game mutation happens on the main thread; this struct only decodes packets
// into a channel.
type netClient struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder

	id       int32
	name     string
	mapName  string
	spawn    rl.Vector3
	spawnYaw float32

	msgs chan netMessage
	done chan struct{}

	closeOnce sync.Once
	sendMu    sync.Mutex

	sendAcc float64
	rotDX   float32
	rotDY   float32
}

type netConnectResult struct {
	client *netClient
	err    error
}

type netRemotePlayer struct {
	ID        int32
	Name      string
	Pos       rl.Vector3
	Target    rl.Vector3
	Yaw       float32
	Pitch     float32
	Fly       bool
	Held      int32
	HasTarget bool
}

func (c *netClient) write(m *netMessage) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.enc.Encode(m)
}

func (c *netClient) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *netClient) readLoop() {
	defer close(c.msgs)
	for {
		var m netMessage
		if err := c.dec.Decode(&m); err != nil {
			return
		}
		select {
		case c.msgs <- m:
		case <-c.done:
			return
		}
	}
}

// dialNetClient performs the handshake synchronously (so the caller can show a
// useful error) and then starts the decode goroutine. The Game is untouched.
func dialNetClient(addr, name string) (*netClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 8*time.Second)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	if err := enc.Encode(&netMessage{
		Type:     msgHello,
		Protocol: netProtocolVersion,
		Name:     sanitizeNetName(name),
	}); err != nil {
		conn.Close()
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(8 * time.Second))
	var welcome netMessage
	if err := dec.Decode(&welcome); err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetReadDeadline(time.Time{})

	if welcome.Type == msgBye {
		conn.Close()
		if welcome.Reason == "" {
			welcome.Reason = "server refused connection"
		}
		return nil, errors.New(welcome.Reason)
	}
	if welcome.Type != msgWelcome {
		conn.Close()
		return nil, fmt.Errorf("unexpected handshake %q", welcome.Type)
	}
	if welcome.Protocol != netProtocolVersion {
		conn.Close()
		return nil, fmt.Errorf("protocol mismatch (server %d, client %d)", welcome.Protocol, netProtocolVersion)
	}

	c := &netClient{
		conn:     conn,
		enc:      enc,
		dec:      dec,
		id:       welcome.ClientID,
		name:     sanitizeNetName(name),
		mapName:  welcome.Map,
		spawn:    welcome.Spawn.RL(),
		spawnYaw: welcome.SpawnYaw,
		msgs:     make(chan netMessage, 512),
		done:     make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

// ---------------------------------------------------------------------------
// Game integration
// ---------------------------------------------------------------------------

func (g *Game) startNetClient(c *netClient) {
	g.netClient = c
	g.netMode = netModeClient
	g.netConnecting = false
	g.netStatus = "CONNECTED"
	g.netStatusAt = colAccent

	g.selectedMap = c.mapName
	g.resetWorld(false, toBp(c.spawn), c.spawnYaw, 0, false)
	// The client player is a camera anchor driven by snapshots. It must never
	// fall or be pushed around by the local world.
	g.player.body.SetGravityScale(0)
	g.player.body.SetLinearVelocity(box3d.Vec3Zero)
	g.player.body.SetAngularVelocity(box3d.Vec3Zero)

	g.netProps = make(map[int32]*Entity)
	g.netRemote = make(map[int32]*netRemotePlayer)
	g.netPlayerTarget = c.spawn
	g.netHasTarget = true
	g.state = statePlaying
	g.resumePlaying()
	g.toast("JOINED SERVER // "+c.mapName, colAccent)
}

func (g *Game) disconnectNetClient(reason string) {
	if g.netClient != nil {
		g.netClient.close()
		g.netClient = nil
	}
	g.netMode = netModeOff
	g.netProps = nil
	g.netRemote = nil
	g.netHasTarget = false
	g.netConnecting = false
	g.held = nil
	g.resetWorld(true, defaultSpawn, 0, -0.12, false)
	g.state = stateMainMenu
	if reason != "" {
		g.toast(reason, colWarn)
	}
}

func (g *Game) netClientPoll() {
	c := g.netClient
	if c == nil {
		return
	}
	for i := 0; i < 128; i++ {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				g.disconnectNetClient("SERVER CLOSED CONNECTION")
				return
			}
			g.handleNetMessage(c, &m)
		default:
			return
		}
	}
}

func (g *Game) handleNetMessage(c *netClient, m *netMessage) {
	switch m.Type {
	case msgSnapshot:
		if m.Snap != nil {
			g.applyNetSnapshot(m.Snap)
		}
	case msgToast:
		g.toast(m.Text, unpackColor(m.Color))
	case msgEvent:
		if g.addons != nil && m.Event != "" {
			g.addons.FireEvent(m.Event, m.Data)
		}
	case msgBye:
		g.disconnectNetClient("DISCONNECTED: " + m.Reason)
	}
}

// netClientStep replaces the local fixed-step physics loop for connected
// clients. It advances visual time, interpolates snapshot targets and sends
// the current input packet.
func (g *Game) netClientStep(dt float32) {
	g.time += float64(dt)
	g.updateNetVisuals(dt)
	g.smoothLocalPlayer(dt)
	g.updateRemotePlayers(dt)
	g.sendNetInput(dt)
}

func (g *Game) sendNetInput(dt float32) {
	c := g.netClient
	if c == nil {
		return
	}
	c.sendAcc += float64(dt)
	if c.sendAcc < 1.0/netInputRate {
		return
	}
	c.sendAcc = 0

	in := localInput(g.player)
	pkt := netInput{
		Forward:  in.Forward,
		Right:    in.Right,
		Jump:     in.Jump,
		Down:     in.Down,
		Sprint:   in.Sprint,
		Yaw:      in.Yaw,
		Pitch:    in.Pitch,
		HoldRotX: c.rotDX,
		HoldRotY: c.rotDY,
	}
	if g.held != nil {
		pkt.HoldDist = g.holdDist
	}
	c.rotDX, c.rotDY = 0, 0

	if err := c.write(&netMessage{Type: msgInput, Input: &pkt}); err != nil {
		g.disconnectNetClient("CONNECTION LOST")
	}
}

func (g *Game) netClientHeldUpdate(dt float32) {
	if g.held == nil {
		return
	}
	// Push/pull while holding E, mirroring the local physgun control.
	if rl.IsKeyDown(rl.KeyE) {
		if rl.IsKeyDown(rl.KeyW) {
			g.holdDist = clampf(g.holdDist+9*dt, minPlaceDist, maxPlaceDist)
		}
		if rl.IsKeyDown(rl.KeyS) {
			g.holdDist = clampf(g.holdDist-9*dt, minPlaceDist, maxPlaceDist)
		}
	}
	if !rl.IsKeyDown(rl.KeyE) && !rl.IsKeyDown(rl.KeyR) {
		return
	}
	md := rl.GetMouseDelta()
	if md.X == 0 && md.Y == 0 {
		return
	}
	sens := float32(0.003) * g.settings.Sensitivity
	if sens <= 0 {
		sens = 0.003
	}
	c := g.netClient
	if c == nil {
		return
	}
	c.rotDX += md.X * sens
	c.rotDY += md.Y * sens
}

// ---------------------------------------------------------------------------
// Snapshot application
// ---------------------------------------------------------------------------

func (g *Game) applyNetSnapshot(snap *netSnapshot) {
	seen := make(map[int32]bool, len(snap.Props))
	for i := range snap.Props {
		ps := &snap.Props[i]
		seen[ps.ID] = true
		if e := g.netProps[ps.ID]; e != nil {
			e.NetTargetPos = ps.Pos.BP()
			e.NetTargetRot = ps.Rot.BQ()
			e.NetHasTarget = true
			e.Frozen = ps.Frozen
			e.AnimPlaying = ps.AnimPlaying
			e.AnimActive = ps.AnimActive
			e.AnimSpeed = ps.AnimSpeed
			if int32(ps.Anim) != e.AnimCurrent {
				e.AnimCurrent = int32(ps.Anim)
			}
			// Only hard-correct the frame when it drifted; otherwise let the
			// local advanceAnimations keep playback smooth at 60 fps.
			if math.Abs(float64(ps.AnimFrame-e.AnimFrame)) > 3 {
				e.AnimFrame = ps.AnimFrame
			}
			continue
		}
		g.spawnNetProp(ps)
	}
	for _, id := range snap.Removed {
		g.removeNetProp(id)
	}
	if snap.Full {
		for id := range g.netProps {
			if !seen[id] {
				g.removeNetProp(id)
			}
		}
	}

	playerSeen := make(map[int32]bool, len(snap.Players))
	for _, ps := range snap.Players {
		playerSeen[ps.ID] = true
		if g.netClient != nil && ps.ID == g.netClient.id {
			g.netPlayerTarget = ps.Pos.RL()
			g.netHasTarget = true
			// Reconcile the locally predicted grab with the authoritative
			// held id. A short grace window avoids flicker while the server
			// processes the grab action.
			if ps.Held != 0 {
				if e := g.netProps[ps.Held]; e != nil && g.held != e {
					g.held = e
					g.heldLocked = true
				}
			} else if g.held != nil && g.time-g.netGrabAt > 0.5 {
				g.held = nil
				g.heldLocked = false
			}
			continue
		}
		rp := g.netRemote[ps.ID]
		if rp == nil {
			rp = &netRemotePlayer{ID: ps.ID, Pos: ps.Pos.RL()}
			g.netRemote[ps.ID] = rp
		}
		rp.Target = ps.Pos.RL()
		rp.Name = ps.Name
		rp.Yaw = ps.Yaw
		rp.Pitch = ps.Pitch
		rp.Fly = ps.Fly
		rp.Held = ps.Held
		rp.HasTarget = true
	}
	if snap.Full {
		for id := range g.netRemote {
			if !playerSeen[id] {
				delete(g.netRemote, id)
			}
		}
	}
}

func (g *Game) spawnNetProp(ps *netPropState) {
	preset := (*ShapePreset)(nil)
	if ps.Name != "" {
		preset = findPreset(ps.Name)
	}
	pos := ps.Pos.BP()
	rot := ps.Rot.BQ()
	if rot.V.X == 0 && rot.V.Y == 0 && rot.V.Z == 0 && rot.S == 0 {
		rot = box3d.QuatIdentity
	}
	col := unpackColor(ps.Color)
	if col.A == 0 {
		col.A = 255
	}

	var e *Entity
	if preset != nil {
		e = g.spawnPresetMode(*preset, pos, rot, col, true)
	} else {
		p := ShapePreset{
			Name:        ps.Name,
			Kind:        PropKind(ps.Kind),
			Half:        ps.Half.BV(),
			Radius:      ps.Radius,
			Height:      ps.Height,
			Density:     120,
			Friction:    0.6,
			Restitution: 0.1,
			Color:       col,
		}
		if p.Half.X <= 0 || p.Half.Y <= 0 || p.Half.Z <= 0 {
			p.Half = bv(0.5, 0.5, 0.5)
		}
		if p.Radius <= 0 {
			p.Radius = 0.5
		}
		if p.Height <= 0 {
			p.Height = 1
		}
		e = g.spawnPresetMode(p, pos, rot, col, true)
	}
	if e == nil {
		return
	}
	e.NetID = ps.ID
	e.NetVisual = true
	e.NetTargetPos = pos
	e.NetTargetRot = rot
	e.NetHasTarget = true
	e.Frozen = ps.Frozen
	e.AnimCurrent = int32(ps.Anim)
	e.AnimFrame = ps.AnimFrame
	e.AnimPlaying = ps.AnimPlaying
	e.AnimActive = ps.AnimActive
	e.AnimSpeed = ps.AnimSpeed
	g.netProps[ps.ID] = e
}

func (g *Game) removeNetProp(id int32) {
	e := g.netProps[id]
	if e == nil {
		return
	}
	if g.held == e {
		g.held = nil
	}
	delete(g.netProps, id)
	g.removeEntity(e)
}

func (g *Game) updateNetVisuals(dt float32) {
	blend := smoothing(28, dt)
	for _, e := range g.netProps {
		if !e.NetHasTarget {
			continue
		}
		p := e.Body.GetPosition()
		t := e.NetTargetPos
		np := box3d.Pos{
			X: lerpf(p.X, t.X, blend),
			Y: lerpf(p.Y, t.Y, blend),
			Z: lerpf(p.Z, t.Z, blend),
		}
		q := e.Body.GetRotation()
		nq := slerpQuat(q, e.NetTargetRot, blend)
		e.Body.SetTransform(np, nq)
	}
}

func (g *Game) smoothLocalPlayer(dt float32) {
	if !g.netHasTarget || g.player == nil {
		return
	}
	p := g.player.body.GetPosition()
	t := g.netPlayerTarget
	if vlen(p.X-t.X, p.Y-t.Y, p.Z-t.Z) > 8 {
		g.player.body.SetTransform(toBp(t), box3d.QuatIdentity)
		g.player.body.SetLinearVelocity(box3d.Vec3Zero)
		return
	}
	blend := smoothing(16, dt)
	np := box3d.Pos{
		X: lerpf(p.X, t.X, blend),
		Y: lerpf(p.Y, t.Y, blend),
		Z: lerpf(p.Z, t.Z, blend),
	}
	g.player.body.SetTransform(np, box3d.QuatIdentity)
	g.player.body.SetLinearVelocity(box3d.Vec3Zero)
	g.player.body.SetAngularVelocity(box3d.Vec3Zero)
}

func (g *Game) updateRemotePlayers(dt float32) {
	blend := smoothing(14, dt)
	for _, rp := range g.netRemote {
		if !rp.HasTarget {
			continue
		}
		rp.Pos = rlv(
			lerpf(rp.Pos.X, rp.Target.X, blend),
			lerpf(rp.Pos.Y, rp.Target.Y, blend),
			lerpf(rp.Pos.Z, rp.Target.Z, blend),
		)
	}
}

// slerpQuat interpolates two quaternions. The shortest arc is chosen so props
// never spin the long way around between snapshots.
func slerpQuat(a, b box3d.Quat, t float32) box3d.Quat {
	ax, ay, az, aw := a.V.X, a.V.Y, a.V.Z, a.S
	bx, by, bz, bw := b.V.X, b.V.Y, b.V.Z, b.S
	dot := ax*bx + ay*by + az*bz + aw*bw
	if dot < 0 {
		bx, by, bz, bw = -bx, -by, -bz, -bw
		dot = -dot
	}
	if dot > 0.9995 {
		q := box3d.Quat{
			V: box3d.Vec3{
				X: lerpf(ax, bx, t),
				Y: lerpf(ay, by, t),
				Z: lerpf(az, bz, t),
			},
			S: lerpf(aw, bw, t),
		}
		return normalizeQuat(q)
	}
	theta := float32(math.Acos(float64(dot)))
	sinTheta := float32(math.Sin(float64(theta)))
	if sinTheta < 1e-6 {
		return a
	}
	s0 := float32(math.Sin(float64((1-t)*theta))) / sinTheta
	s1 := float32(math.Sin(float64(t*theta))) / sinTheta
	return box3d.Quat{
		V: box3d.Vec3{
			X: ax*s0 + bx*s1,
			Y: ay*s0 + by*s1,
			Z: az*s0 + bz*s1,
		},
		S: aw*s0 + bw*s1,
	}
}

// ---------------------------------------------------------------------------
// Client actions
// ---------------------------------------------------------------------------

func (g *Game) netClientActionInput() {
	if g.justResumed {
		g.justResumed = false
		return
	}
	if rl.IsKeyPressed(rl.KeyV) {
		g.netSendAction(netAction{Kind: "fly"})
	}
	if rl.IsKeyPressed(rl.KeyF) {
		g.netFreezeAimed()
	}
	if rl.IsKeyPressed(rl.KeyG) {
		g.netSendAction(netAction{Kind: "gravity_cycle"})
	}
	if rl.IsKeyPressed(rl.KeyLeftBracket) {
		g.netSendAction(netAction{Kind: "gravity_nudge", Value: -0.1})
	}
	if rl.IsKeyPressed(rl.KeyRightBracket) {
		g.netSendAction(netAction{Kind: "gravity_nudge", Value: 0.1})
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		g.netClientPrimary()
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
		g.netClientSecondary()
	}
}

func (g *Game) netSendAction(a netAction) {
	c := g.netClient
	if c == nil {
		return
	}
	pkt := a
	if err := c.write(&netMessage{Type: msgAction, Action: &pkt}); err != nil {
		g.disconnectNetClient("CONNECTION LOST")
	}
}

func (g *Game) netAimEntity() *Entity {
	if g.held != nil {
		return g.held
	}
	e, _ := g.castPropRay(28)
	if e == nil || e.NetID == 0 {
		return nil
	}
	return e
}

func (g *Game) netAimCenter() netVec3 {
	if g.aim != nil {
		return netVecFromRl(g.aimPoint)
	}
	eye := g.eyePosition()
	f := g.cameraForward()
	return netVecFromRl(rlv(eye.X+f.X*6, eye.Y+f.Y*6, eye.Z+f.Z*6))
}

func (g *Game) netFreezeAimed() {
	e := g.netAimEntity()
	if e == nil {
		g.toast("No target to freeze", colMuted)
		return
	}
	g.netSendAction(netAction{Kind: "freeze", ID: e.NetID})
}

func (g *Game) netGrabAimed() {
	e := g.netAimEntity()
	if e == nil {
		g.toast("No target in range", colMuted)
		return
	}
	if e.Frozen {
		g.toast("Target is frozen", colFreeze)
		return
	}
	g.netSendAction(netAction{Kind: "grab", ID: e.NetID})
	g.held = e
	g.netGrabAt = g.time
	eye := g.eyePosition()
	p := e.position()
	g.holdDist = clampf(vlen(p.X-eye.X, p.Y-eye.Y, p.Z-eye.Z), minPlaceDist, maxPlaceDist)
	g.spawnDist = g.holdDist
	g.heldLocked = true
}

func (g *Game) netReleaseHeld() {
	if g.held == nil {
		return
	}
	g.netSendAction(netAction{Kind: "release"})
	g.held = nil
	g.heldLocked = false
}

func (g *Game) netThrowHeld() {
	if g.held == nil {
		return
	}
	g.netSendAction(netAction{Kind: "throw"})
	g.held = nil
	g.heldLocked = false
}

func (g *Game) netClientPrimary() {
	g.toolKick = 1
	switch g.tool {
	case toolHand, toolPhysgun:
		if g.held != nil {
			g.netReleaseHeld()
		} else {
			g.netGrabAimed()
		}
	case toolSpawner:
		g.spawnSelectedAt(g.spawnPoint())
	case toolRemover:
		if e := g.netAimEntity(); e != nil {
			g.netSendAction(netAction{Kind: "delete", ID: e.NetID})
		} else {
			g.toast("Aim at a prop to delete it", colMuted)
		}
	case toolImpact:
		g.netSendAction(netAction{Kind: "blast", Pos: g.netAimCenter()})
	case toolFreeze:
		g.netFreezeAimed()
	case toolDuplicator:
		if e := g.netAimEntity(); e != nil {
			g.netSendAction(netAction{Kind: "duplicate", ID: e.NetID})
		} else {
			g.toast("No target to duplicate", colMuted)
		}
	case toolGravity:
		g.netSendAction(netAction{Kind: "gravity_cycle"})
	}
}

func (g *Game) netClientSecondary() {
	g.toolKick = 1
	switch g.tool {
	case toolHand, toolPhysgun:
		if g.held != nil {
			g.netThrowHeld()
		} else if e := g.netAimEntity(); e != nil {
			g.netSendAction(netAction{Kind: "push", ID: e.NetID})
		}
	case toolSpawner:
		g.openSpawnMenu()
	case toolRemover:
		g.netSendAction(netAction{Kind: "remove_radius", Pos: g.netAimCenter()})
	case toolImpact:
		if e := g.netAimEntity(); e != nil {
			g.netSendAction(netAction{Kind: "launch", ID: e.NetID})
		}
	case toolFreeze:
		g.netSendAction(netAction{Kind: "freeze_all", Flag: !g.netAllFrozen()})
	case toolDuplicator:
		if e := g.netAimEntity(); e != nil {
			g.netSendAction(netAction{Kind: "duplicate", ID: e.NetID, Flag: true, Pos: g.netAimCenter()})
		}
	case toolGravity:
		g.netSendAction(netAction{Kind: "gravity_zero"})
	}
}

func (g *Game) netAllFrozen() bool {
	if len(g.props) == 0 {
		return false
	}
	for _, e := range g.props {
		if !e.Frozen {
			return false
		}
	}
	return true
}

// netClientSpawn asks the server to spawn a preset. The client never creates
// authoritative props locally; it waits for the next snapshot.
func (g *Game) netClientSpawn(index int, pos rl.Vector3) {
	if index < 0 || index >= len(presets) {
		return
	}
	p := presets[index]
	support := p.support(bv(0, 1, 0))
	if pos.Y < support+0.06 {
		pos.Y = support + 0.06
	}
	pos.X = clampf(pos.X, -29, 29)
	pos.Z = clampf(pos.Z, -29, 29)
	g.netSendAction(netAction{
		Kind:   "spawn",
		Preset: p.Name,
		Pos:    netVecFromRl(pos),
		Rot:    netQuatFromBq(box3d.QuatIdentity),
	})
}
