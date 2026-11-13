package game

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type netServerClient struct {
	id   int32
	name string
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder

	spawn    box3d.Pos
	spawnYaw float32

	mu       sync.Mutex
	input    netInput
	hasInput bool
	rotDX    float32
	rotDY    float32

	sendMu sync.Mutex
	closed bool

	player     *Player
	held       *Entity
	heldLocked bool
	holdDist   float32
	tool       ToolID

	needsFull bool
}

func (c *netServerClient) setInput(in *netInput) {
	c.mu.Lock()
	c.input = *in
	c.hasInput = true
	c.rotDX += in.HoldRotX
	c.rotDY += in.HoldRotY
	c.mu.Unlock()
}

func (c *netServerClient) takeInput() (netInput, float32, float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	in := c.input
	dx, dy := c.rotDX, c.rotDY
	c.rotDX, c.rotDY = 0, 0
	return in, dx, dy, c.hasInput
}

// snapshotInput copies the latest input without consuming rotation deltas.
func (c *netServerClient) snapshotInput() netInput {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.input
}

func (c *netServerClient) send(m *netMessage) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closed {
		return net.ErrClosed
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	err := c.enc.Encode(m)
	_ = c.conn.SetWriteDeadline(time.Time{})
	return err
}

// netServer owns the authoritative simulation and all connected clients.
type netServer struct {
	game *Game
	ln   net.Listener
	port int

	cmdCh  chan func(*Game)
	stopCh chan struct{}
	once   sync.Once

	slotMu sync.Mutex
	nextID int32
	slots  int

	clients map[int32]*netServerClient

	tick      uint64
	snapCount uint64
	snapAcc   float64
	sent      map[int32]netPropState
}

func newNetServer(g *Game, port int) *netServer {
	return &netServer{
		game:    g,
		port:    port,
		cmdCh:   make(chan func(*Game), 256),
		stopCh:  make(chan struct{}),
		clients: make(map[int32]*netServerClient),
		sent:    make(map[int32]netPropState),
	}
}

func (s *netServer) start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.ln = ln
	go s.acceptLoop()
	return nil
}

func (s *netServer) stop() {
	s.once.Do(func() {
		close(s.stopCh)
		if s.ln != nil {
			_ = s.ln.Close()
		}
		for _, c := range s.clients {
			c.sendMu.Lock()
			c.closed = true
			_ = c.conn.Close()
			c.sendMu.Unlock()
		}
	})
}

func (s *netServer) run() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	dt := float32(1.0 / netTickRate)
	ticker := time.NewTicker(time.Second / netTickRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-sig:
			return
		case <-ticker.C:
			s.tickOnce(dt)
		}
	}
}

func (s *netServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *netServer) handleConn(conn net.Conn) {
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var hello netMessage
	if err := dec.Decode(&hello); err != nil {
		_ = conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	if hello.Type != msgHello || hello.Protocol != netProtocolVersion {
		_ = enc.Encode(&netMessage{
			Type:   msgBye,
			Reason: fmt.Sprintf("protocol mismatch (server %d)", netProtocolVersion),
		})
		_ = conn.Close()
		return
	}

	name := sanitizeNetName(hello.Name)
	id, spawn, yaw := s.assignSlot()
	c := &netServerClient{
		id:        id,
		name:      name,
		conn:      conn,
		enc:       enc,
		dec:       dec,
		spawn:     spawn,
		spawnYaw:  yaw,
		tool:      toolPhysgun,
		needsFull: true,
	}

	if err := c.send(&netMessage{
		Type:         msgWelcome,
		Protocol:     netProtocolVersion,
		ClientID:     id,
		Map:          s.game.selectedMap,
		TickRate:     netTickRate,
		SnapshotRate: netSnapshotRate,
		Spawn:        netVecFromBp(spawn),
		SpawnYaw:     yaw,
	}); err != nil {
		_ = conn.Close()
		return
	}

	s.cmdCh <- func(g *Game) { s.addClient(c) }

	for {
		var m netMessage
		if err := dec.Decode(&m); err != nil {
			break
		}
		switch m.Type {
		case msgInput:
			if m.Input != nil {
				c.setInput(m.Input)
			}
		case msgAction:
			if m.Action != nil {
				a := *m.Action
				s.cmdCh <- func(g *Game) { s.handleAction(c, &a) }
			}
		case msgEvent:
			if m.Event != "" {
				event, data := m.Event, m.Data
				s.cmdCh <- func(g *Game) {
					if g.addons != nil {
						g.addons.FireEvent(event, data)
					}
				}
			}
		}
	}

	s.cmdCh <- func(g *Game) { s.dropClient(c.id, "left") }
	_ = conn.Close()
}

func (s *netServer) assignSlot() (int32, box3d.Pos, float32) {
	s.slotMu.Lock()
	defer s.slotMu.Unlock()
	s.nextID++
	s.slots++
	id := s.nextID
	// Spread joins around the default drop point so bodies do not spawn inside
	// each other.
	angle := float64(id) * 1.7
	radius := float64(1.4 + float64(id%3)*0.6)
	spawn := box3d.Pos{
		X: float32(math.Cos(angle) * radius),
		Y: defaultSpawn.Y,
		Z: defaultSpawn.Z + float32(math.Sin(angle)*radius),
	}
	return id, spawn, float32(math.Sin(angle) * 0.6)
}

func (s *netServer) processCommands() {
	for {
		select {
		case fn := <-s.cmdCh:
			fn(s.game)
		default:
			return
		}
	}
}

func (s *netServer) addClient(c *netServerClient) {
	g := s.game
	if _, exists := s.clients[c.id]; exists {
		return
	}
	c.player = newPlayer(g.world, c.spawn, c.spawnYaw, 0, false)
	c.held = nil
	s.clients[c.id] = c
	fmt.Printf("[server] %s joined (id %d, %d online)\n", c.name, c.id, len(s.clients))
}

func (s *netServer) dropClient(id int32, reason string) {
	c := s.clients[id]
	if c == nil {
		return
	}
	if c.held != nil {
		s.releaseHeld(c)
	}
	if c.player != nil {
		c.player.body.Destroy()
		c.player = nil
	}
	delete(s.clients, id)
	c.sendMu.Lock()
	c.closed = true
	_ = c.conn.Close()
	c.sendMu.Unlock()
	fmt.Printf("[server] %s disconnected (%s, %d online)\n", c.name, reason, len(s.clients))
}

func (s *netServer) clientSlice() []*netServerClient {
	out := make([]*netServerClient, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, c)
	}
	return out
}

func (s *netServer) tickOnce(dt float32) {
	s.processCommands()
	g := s.game
	g.time += float64(dt)

	clients := s.clientSlice()
	for _, c := range clients {
		if c.player == nil {
			continue
		}
		in, rdx, rdy, ok := c.takeInput()
		state := InputState{Yaw: c.player.yaw, Pitch: c.player.pitch}
		if ok {
			state = InputState{
				Forward: in.Forward,
				Right:   in.Right,
				Jump:    in.Jump,
				Down:    in.Down,
				Sprint:  in.Sprint,
				Yaw:     in.Yaw,
				Pitch:   in.Pitch,
			}
			if c.held != nil && in.HoldDist > 0 {
				c.holdDist = clampf(in.HoldDist, minPlaceDist, maxPlaceDist)
			}
		}
		c.player.tickInput(g, state, dt)
		if c.held != nil {
			s.applyHeldRotation(c, rdx, rdy)
			s.updateHeld(c, dt)
		}
	}

	g.world.Step(dt, 2)

	for _, c := range clients {
		if c.held != nil {
			c.held.Body.SetAngularVelocity(box3d.Vec3Zero)
		}
	}
	g.enforceBounds()
	for _, c := range clients {
		if c.held != nil && g.byID[c.held.ID] != c.held {
			c.held = nil
			c.heldLocked = false
		}
		if c.player == nil {
			continue
		}
		p := c.player.body.GetPosition()
		if p.Y < -6 {
			c.player.respawn(g)
		}
	}

	if g.addons != nil {
		g.addons.Tick()
	}

	s.tick++
	s.snapAcc += float64(dt)
	if s.snapAcc >= 1.0/netSnapshotRate {
		s.snapAcc = 0
		s.broadcastSnapshot()
	}
}

func (s *netServer) playerStates() []netPlayerState {
	out := make([]netPlayerState, 0, len(s.clients))
	for _, c := range s.clients {
		if c.player == nil {
			continue
		}
		p := c.player.body.GetPosition()
		held := int32(0)
		if c.held != nil {
			held = c.held.ID
		}
		out = append(out, netPlayerState{
			ID:    c.id,
			Name:  c.name,
			Pos:   netVecFromBp(p),
			Yaw:   c.player.yaw,
			Pitch: c.player.pitch,
			Fly:   c.player.fly,
			Held:  held,
		})
	}
	return out
}

func (s *netServer) broadcastSnapshot() {
	g := s.game
	full := s.snapCount%netFullSnapshotEvery == 0
	s.snapCount++

	current := make(map[int32]netPropState, len(g.props))
	for _, e := range g.props {
		if e.Static {
			continue
		}
		current[e.ID] = propStateFromEntity(e)
	}
	players := s.playerStates()

	var msg netMessage
	if full {
		s.sent = current
		msg = netMessage{Type: msgSnapshot, Snap: &netSnapshot{
			Tick:    s.tick,
			Time:    g.time,
			Full:    true,
			Players: players,
			Props:   allPropStates(current),
		}}
	} else {
		changed := make([]netPropState, 0, 16)
		for id, ps := range current {
			if old, ok := s.sent[id]; !ok || propStateChanged(old, ps) {
				changed = append(changed, ps)
			}
		}
		removed := make([]int32, 0, 4)
		for id := range s.sent {
			if _, ok := current[id]; !ok {
				removed = append(removed, id)
			}
		}
		s.sent = current
		msg = netMessage{Type: msgSnapshot, Snap: &netSnapshot{
			Tick:    s.tick,
			Time:    g.time,
			Players: players,
			Props:   changed,
			Removed: removed,
		}}
	}

	var toDrop []int32
	for _, c := range s.clients {
		if c.player == nil {
			continue
		}
		if c.needsFull {
			fullMsg := netMessage{Type: msgSnapshot, Snap: &netSnapshot{
				Tick:    s.tick,
				Time:    g.time,
				Full:    true,
				Players: players,
				Props:   allPropStates(current),
			}}
			if err := c.send(&fullMsg); err != nil {
				toDrop = append(toDrop, c.id)
				continue
			}
			c.needsFull = false
			continue
		}
		if err := c.send(&msg); err != nil {
			toDrop = append(toDrop, c.id)
		}
	}
	for _, id := range toDrop {
		s.dropClient(id, "send failed")
	}
}

func allPropStates(m map[int32]netPropState) []netPropState {
	out := make([]netPropState, 0, len(m))
	for _, ps := range m {
		out = append(out, ps)
	}
	return out
}

// broadcastToast sends a HUD toast to every connected client.
func (s *netServer) broadcastToast(text string, color rl.Color) {
	if text == "" {
		return
	}
	msg := netMessage{Type: msgToast, Text: text, Color: packColor(color)}
	s.broadcast(&msg)
}

// broadcastEvent forwards a addon event to every connected client. Client addons
// receive it through their normal event handler.
func (s *netServer) broadcastEvent(event string, data any) {
	if event == "" {
		return
	}
	msg := netMessage{Type: msgEvent, Event: event, Data: data}
	s.broadcast(&msg)
}

// broadcast writes one message to every joined client and drops any client
// whose connection has failed.
func (s *netServer) broadcast(msg *netMessage) {
	var toDrop []int32
	for _, c := range s.clients {
		if c.player == nil {
			continue
		}
		if err := c.send(msg); err != nil {
			toDrop = append(toDrop, c.id)
		}
	}
	for _, id := range toDrop {
		s.dropClient(id, "send failed")
	}
}

// kick removes a client, telling it why first.
func (s *netServer) kick(id int32, reason string) bool {
	c := s.clients[id]
	if c == nil {
		return false
	}
	if reason == "" {
		reason = "kicked by server"
	}
	_ = c.send(&netMessage{Type: msgBye, Reason: reason})
	s.dropClient(id, "kicked: "+reason)
	return true
}

// ---------------------------------------------------------------------------
// Server-side action handling
// ---------------------------------------------------------------------------

func (s *netServer) handleAction(c *netServerClient, a *netAction) {
	if c.player == nil {
		return
	}
	g := s.game
	switch a.Kind {
	case "grab":
		s.grab(c, a.ID)
	case "release":
		s.releaseHeld(c)
	case "throw":
		s.throwHeld(c)
	case "spawn":
		s.spawnPreset(c, a)
	case "delete":
		if e := g.byID[a.ID]; e != nil && !e.Static {
			s.clearHeldByEntity(e)
			g.removeEntity(e)
		}
	case "duplicate":
		s.duplicate(c, a)
	case "freeze":
		if e := g.byID[a.ID]; e != nil && !e.Static {
			g.toggleFrozen(e)
		}
	case "freeze_all":
		g.setAllFrozen(a.Flag)
	case "blast":
		center := a.Pos.RL()
		if center.X == 0 && center.Y == 0 && center.Z == 0 {
			center = s.aimPoint(c)
		}
		s.blastAt(center)
	case "push":
		s.push(c, a.ID)
	case "launch":
		s.launch(c, a.ID)
	case "remove_radius":
		center := a.Pos.RL()
		if center.X == 0 && center.Y == 0 && center.Z == 0 {
			center = s.aimPoint(c)
		}
		g.removeInRadius(center, 3.2)
	case "gravity_cycle":
		g.cycleGravity()
	case "gravity_zero":
		g.toggleZeroGravity()
	case "gravity_nudge":
		g.nudgeGravity(a.Value)
	case "fly":
		c.player.fly = !c.player.fly
		if c.player.fly {
			c.player.body.SetGravityScale(0)
			c.player.body.SetLinearVelocity(box3d.Vec3Zero)
		} else {
			c.player.body.SetGravityScale(1)
		}
		c.player.body.SetAwake(true)
	case "respawn":
		c.player.respawn(g)
	}
}

func (s *netServer) clearHeldByEntity(e *Entity) {
	for _, c := range s.clients {
		if c.held == e {
			c.held = nil
			c.heldLocked = false
		}
	}
}

func (s *netServer) clientEye(c *netServerClient) rl.Vector3 {
	return eyePositionFor(c.player)
}

func (s *netServer) clientForward(c *netServerClient) rl.Vector3 {
	in := c.snapshotInput()
	return cameraForwardYawPitch(in.Yaw, in.Pitch)
}

func (s *netServer) aimPoint(c *netServerClient) rl.Vector3 {
	eye := s.clientEye(c)
	in := c.snapshotInput()
	fwd := cameraForwardYawPitch(in.Yaw, in.Pitch)
	return rlv(eye.X+fwd.X*6, eye.Y+fwd.Y*6, eye.Z+fwd.Z*6)
}

func (s *netServer) grab(c *netServerClient, id int32) {
	g := s.game
	if c.held != nil {
		return
	}
	e := g.byID[id]
	if e == nil || e.Static || e.Frozen {
		return
	}
	eye := s.clientEye(c)
	p := e.position()
	d := vlen(p.X-eye.X, p.Y-eye.Y, p.Z-eye.Z)
	if d > 32 {
		return
	}
	c.held = e
	c.heldLocked = true
	c.holdDist = clampf(d, minPlaceDist, maxPlaceDist)
	e.Body.SetMotionLocks(box3d.MotionLocks{AngularX: true, AngularY: true, AngularZ: true})
	if g.addons != nil {
		g.addons.FireEvent("entity.grab", g.entityInfo(e))
	}
}

func (s *netServer) releaseHeld(c *netServerClient) {
	if c.held != nil && c.heldLocked {
		c.held.Body.SetMotionLocks(box3d.MotionLocks{})
	}
	c.held = nil
	c.heldLocked = false
}

func (s *netServer) throwHeld(c *netServerClient) {
	e := c.held
	if e == nil {
		return
	}
	in := c.snapshotInput()
	dir := cameraForwardYawPitch(in.Yaw, in.Pitch)
	pv := c.player.body.GetLinearVelocity()
	e.Body.SetLinearVelocity(box3d.Vec3{
		X: dir.X*22 + pv.X*0.35,
		Y: dir.Y*22 + pv.Y*0.25 + 1.5,
		Z: dir.Z*22 + pv.Z*0.35,
	})
	e.Body.SetAngularVelocity(bv(
		(s.game.rng.Float32()-0.5)*7,
		(s.game.rng.Float32()-0.5)*7,
		(s.game.rng.Float32()-0.5)*7,
	))
	if c.heldLocked {
		e.Body.SetMotionLocks(box3d.MotionLocks{})
	}
	c.held = nil
	c.heldLocked = false
	s.game.burst(e.position(), e.Color, 12, 4.5, 0.08)
}

func (s *netServer) applyHeldRotation(c *netServerClient, dx, dy float32) {
	if c.held == nil || (dx == 0 && dy == 0) {
		return
	}
	q := c.held.Body.GetRotation()
	yawQ := box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, dx)
	pitchQ := box3d.MakeQuatFromAxisAngle(s.cameraRightAxis(c), dy)
	q = box3d.MulQuat(pitchQ, box3d.MulQuat(yawQ, q))
	c.held.Body.SetTransform(c.held.Body.GetPosition(), q)
	c.held.Body.SetAngularVelocity(box3d.Vec3Zero)
}

func (s *netServer) cameraRightAxis(c *netServerClient) box3d.Vec3 {
	in := c.snapshotInput()
	fwd := cameraForwardYawPitch(in.Yaw, in.Pitch)
	r := rl.Vector3CrossProduct(fwd, rlv(0, 1, 0))
	if lenRl(r) < 1e-4 {
		r = rightFlatYaw(in.Yaw)
	}
	r = normRl(r)
	return bv(r.X, r.Y, r.Z)
}

func (s *netServer) updateHeld(c *netServerClient, dt float32) {
	e := c.held
	if e == nil {
		return
	}
	if s.game.byID[e.ID] != e {
		c.held = nil
		c.heldLocked = false
		return
	}
	in := c.snapshotInput()
	eye := s.clientEye(c)
	dir := cameraForwardYawPitch(in.Yaw, in.Pitch)
	target := rlv(
		eye.X+dir.X*c.holdDist,
		eye.Y+dir.Y*c.holdDist,
		eye.Z+dir.Z*c.holdDist,
	)
	target.Y -= 0.18 + e.radiusXZ()*0.20

	p := e.Body.GetPosition()
	dx := target.X - p.X
	dy := target.Y - p.Y
	dz := target.Z - p.Z
	dist := vlen(dx, dy, dz)
	if dist > 14 {
		s.releaseHeld(c)
		return
	}
	tx, ty, tz := float32(0), float32(0), float32(0)
	if dist > 1e-4 {
		speed := clampf(dist*18, 0, 40)
		tx = dx / dist * speed
		ty = dy / dist * speed
		tz = dz / dist * speed
	}
	v := e.Body.GetLinearVelocity()
	blend := smoothing(30, dt)
	e.Body.SetLinearVelocity(box3d.Vec3{
		X: lerpf(v.X, tx, blend),
		Y: lerpf(v.Y, ty, blend),
		Z: lerpf(v.Z, tz, blend),
	})
	e.Body.SetAngularVelocity(box3d.Vec3Zero)
}

func (s *netServer) spawnPreset(c *netServerClient, a *netAction) {
	g := s.game
	if len(g.props) >= maxProps {
		return
	}
	p := findPreset(a.Preset)
	if p == nil {
		return
	}
	pos := a.Pos.RL()
	support := p.support(bv(0, 1, 0))
	if pos.Y < support+0.06 {
		pos.Y = support + 0.06
	}
	pos.X = clampf(pos.X, -29, 29)
	pos.Z = clampf(pos.Z, -29, 29)
	rot := a.Rot.BQ()
	if rot.V.X == 0 && rot.V.Y == 0 && rot.V.Z == 0 && rot.S == 0 {
		rot = box3d.QuatIdentity
	}
	e := g.spawnPreset(*p, toBp(pos), rot, p.Color)
	if e == nil {
		return
	}
	g.burst(pos, e.Color, 10, 3.5, 0.09)
	g.wakeAround(pos, 1.4)
}

func (s *netServer) duplicate(c *netServerClient, a *netAction) {
	g := s.game
	if len(g.props) >= maxProps {
		return
	}
	e := g.byID[a.ID]
	if e == nil || e.Static {
		return
	}
	pos := e.Body.GetPosition()
	rot := e.Body.GetRotation()
	if a.Flag {
		p := a.Pos.RL()
		off := e.radiusXZ()*2 + 0.08
		pos = bp(p.X+off, p.Y, p.Z)
	} else {
		in := c.snapshotInput()
		f := cameraForwardYawPitch(in.Yaw, in.Pitch)
		off := e.radiusXZ()*2 + 0.12
		pos = bp(pos.X+f.X*off, pos.Y+f.Y*off+0.10, pos.Z+f.Z*off)
	}
	copy := g.spawnCopy(e, pos, rot)
	if copy != nil {
		g.burst(copy.position(), copy.Color, 8, 3.0, 0.08)
	}
}

func (s *netServer) blastAt(center rl.Vector3) {
	g := s.game
	const radius = 7.5
	const strength = 18.0
	for _, e := range g.props {
		if e.Frozen {
			continue
		}
		p := e.position()
		dx := p.X - center.X
		dy := p.Y - center.Y
		dz := p.Z - center.Z
		d := vlen(dx, dy, dz)
		if d < 1e-4 || d > radius {
			continue
		}
		falloff := 1 - d/radius
		nx, ny, nz := dx/d, dy/d, dz/d
		ny += 0.35
		nl := vlen(nx, ny, nz)
		if nl < 1e-4 {
			continue
		}
		speed := strength * falloff
		mass := e.Body.GetMass()
		if mass <= 0 {
			mass = 1
		}
		e.Body.ApplyLinearImpulseToCenter(bv(
			nx/nl*mass*speed,
			ny/nl*mass*speed,
			nz/nl*mass*speed,
		), true)
		e.Body.ApplyAngularImpulse(bv(
			(g.rng.Float32()-0.5)*mass*falloff*2,
			(g.rng.Float32()-0.5)*mass*falloff*2,
			(g.rng.Float32()-0.5)*mass*falloff*2,
		), true)
	}
	g.waves = append(g.waves, &Shockwave{
		Center: center, Radius: 0.25, MaxRadius: radius,
		Life: 0.55, MaxLife: 0.55, Color: colEnergy,
	})
	g.burst(center, colEnergy, 34, 13, 0.13)
	g.shake = 0.50
}

func (s *netServer) push(c *netServerClient, id int32) {
	g := s.game
	e := g.byID[id]
	if e == nil || e.Static || e.Frozen {
		return
	}
	in := c.snapshotInput()
	dir := cameraForwardYawPitch(in.Yaw, in.Pitch)
	mass := e.Body.GetMass()
	if mass <= 0 {
		mass = 1
	}
	e.Body.ApplyLinearImpulseToCenter(bv(
		dir.X*mass*16,
		(dir.Y+0.18)*mass*16,
		dir.Z*mass*16,
	), true)
	e.Body.ApplyAngularImpulse(bv(
		(g.rng.Float32()-0.5)*mass*2.5,
		(g.rng.Float32()-0.5)*mass*2.5,
		(g.rng.Float32()-0.5)*mass*2.5,
	), true)
	g.burst(e.position(), colAccent, 8, 3.5, 0.08)
	g.shake = 0.08
}

func (s *netServer) launch(c *netServerClient, id int32) {
	g := s.game
	e := g.byID[id]
	if e == nil || e.Static || e.Frozen {
		return
	}
	in := c.snapshotInput()
	dir := cameraForwardYawPitch(in.Yaw, in.Pitch)
	mass := e.Body.GetMass()
	if mass <= 0 {
		mass = 1
	}
	e.Body.ApplyLinearImpulseToCenter(bv(
		dir.X*mass*24,
		(dir.Y+0.22)*mass*24,
		dir.Z*mass*24,
	), true)
	e.Body.ApplyAngularImpulse(bv(
		(g.rng.Float32()-0.5)*mass*4,
		(g.rng.Float32()-0.5)*mass*4,
		(g.rng.Float32()-0.5)*mass*4,
	), true)
	g.burst(e.position(), colEnergy, 12, 5, 0.09)
	g.shake = 0.12
}

// runDedicatedServer initialises the hidden-window game world and serves it
// until the process is interrupted.
func runDedicatedServer(mapName string, port int) int {
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowHidden)
	rl.InitWindow(320, 240, "FSBOX DEDICATED SERVER")
	defer rl.CloseWindow()

	g := NewGame()
	defer g.Close()
	if mapName != "" {
		g.selectedMap = mapName
		g.resetWorld(true, defaultSpawn, 0, -0.12, false)
	}
	g.state = statePlaying
	g.netMode = netModeServer

	// Park the local single-player body far away so it cannot interfere with
	// the authoritative per-client players.
	g.player.body.SetTransform(bp(0, -500, 0), box3d.QuatIdentity)
	g.player.body.SetGravityScale(0)
	g.player.body.SetType(box3d.StaticBody)

	srv := newNetServer(g, port)
	g.netSrv = srv
	if err := srv.start(); err != nil {
		fmt.Printf("[server] listen failed: %v\n", err)
		return 1
	}
	fmt.Printf("FSBOX dedicated server\n")
	fmt.Printf("  map:  %s\n", g.selectedMap)
	fmt.Printf("  port: %d\n", port)
	fmt.Printf("  join: fsbox.exe -connect 127.0.0.1:%d -name PLAYER\n", port)
	fmt.Printf("  stop: Ctrl+C\n")
	srv.run()
	fmt.Printf("[server] shutting down\n")
	return 0
}
