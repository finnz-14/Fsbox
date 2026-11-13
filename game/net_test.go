package game

import (
	"bytes"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"

	box3d "github.com/gataengine/box3d-go"
)

func TestNetMessageRoundTrip(t *testing.T) {
	in := netInput{
		Forward: 1, Right: -0.5, Jump: true, Sprint: true,
		Yaw: 1.2, Pitch: -0.4, HoldRotX: 0.03, HoldRotY: -0.02, HoldDist: 4.5,
	}
	msg := netMessage{Type: msgInput, Input: &in}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&msg); err != nil {
		t.Fatalf("encode: %v", err)
	}
	var got netMessage
	if err := json.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Type != msgInput || got.Input == nil {
		t.Fatalf("unexpected envelope: %+v", got)
	}
	if got.Input.Forward != 1 || got.Input.Right != -0.5 || !got.Input.Jump ||
		!got.Input.Sprint || got.Input.Yaw != 1.2 || got.Input.Pitch != -0.4 ||
		got.Input.HoldRotX != 0.03 || got.Input.HoldRotY != -0.02 || got.Input.HoldDist != 4.5 {
		t.Fatalf("input round trip mismatch: %+v", *got.Input)
	}
}

func TestNetSnapshotRoundTrip(t *testing.T) {
	snap := netSnapshot{
		Tick: 42,
		Time: 1.5,
		Full: true,
		Players: []netPlayerState{{
			ID: 1, Name: "TESTER", Pos: netVec3{X: 1, Y: 2, Z: 3}, Yaw: 0.5, Pitch: -0.2, Held: 9,
		}},
		Props: []netPropState{{
			ID: 9, Name: "Crate", Kind: uint8(PropBox),
			Half:  netVec3{X: 0.5, Y: 0.5, Z: 0.5},
			Pos:   netVec3{X: 4, Y: 1, Z: 2},
			Rot:   netQuat{W: 1},
			Color: packColor(rgb(10, 20, 30)),
		}},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&netMessage{Type: msgSnapshot, Snap: &snap}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	var got netMessage
	if err := json.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Snap == nil || !got.Snap.Full || got.Snap.Tick != 42 || len(got.Snap.Props) != 1 {
		t.Fatalf("snapshot envelope mismatch: %+v", got.Snap)
	}
	p := got.Snap.Props[0]
	if p.Name != "Crate" || p.Pos.X != 4 || p.Rot.W != 1 {
		t.Fatalf("prop mismatch: %+v", p)
	}
	if c := unpackColor(p.Color); c.R != 10 || c.G != 20 || c.B != 30 || c.A != 255 {
		t.Fatalf("color mismatch: %+v", c)
	}
}

func TestPropStateChanged(t *testing.T) {
	base := netPropState{ID: 1, Pos: netVec3{X: 1, Y: 2, Z: 3}, Rot: netQuat{W: 1}}
	if propStateChanged(base, base) {
		t.Fatal("identical states should not be marked changed")
	}
	moved := base
	moved.Pos.Y += 0.01
	if !propStateChanged(base, moved) {
		t.Fatal("pose change was not detected")
	}
	frozen := base
	frozen.Frozen = true
	if !propStateChanged(base, frozen) {
		t.Fatal("frozen change was not detected")
	}
	anim := base
	anim.Anim = 3
	if !propStateChanged(base, anim) {
		t.Fatal("animation change was not detected")
	}
}

func TestSanitizeNetName(t *testing.T) {
	if got := sanitizeNetName("  "); got != "PLAYER" {
		t.Fatalf("blank name = %q, want PLAYER", got)
	}
	if got := sanitizeNetName("A\x00B\nC"); got != "ABC" {
		t.Fatalf("control characters leaked: %q", got)
	}
	long := sanitizeNetName("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if len([]rune(long)) != netMaxNameLen {
		t.Fatalf("long name length = %d, want %d", len([]rune(long)), netMaxNameLen)
	}
}

func TestNormalizeNetAddr(t *testing.T) {
	cases := map[string]string{
		"":                  "127.0.0.1:" + strconv.Itoa(netDefaultPort),
		"   ":               "127.0.0.1:" + strconv.Itoa(netDefaultPort),
		"localhost":         "localhost:" + strconv.Itoa(netDefaultPort),
		"10.0.0.5:3000":     "10.0.0.5:3000",
		"::1":               "[::1]:" + strconv.Itoa(netDefaultPort),
		"2001:db8::1:27015": "[2001:db8::1:27015]:" + strconv.Itoa(netDefaultPort),
	}
	for in, want := range cases {
		if got := normalizeNetAddr(in); got != want {
			t.Errorf("normalizeNetAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestServerHandshakeAndSnapshot exercises the real TCP handshake and one
// authoritative snapshot. The server tick is driven manually so the test is
// deterministic and never depends on wall-clock timing.
func TestServerHandshakeAndSnapshot(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	g.selectedMap = "default"
	srv := newNetServer(g, 0)
	if err := srv.start(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer srv.stop()

	tcp, ok := srv.ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address %T", srv.ln.Addr())
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(tcp.Port))

	c, err := dialNetClient(addr, "TESTER")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.close()
	if c.id == 0 {
		t.Fatal("client received no id")
	}
	if c.mapName == "" {
		t.Fatal("welcome did not include a map")
	}

	// Drive enough ticks for the join command and a snapshot interval.
	for i := 0; i < netTickRate; i++ {
		srv.tickOnce(1.0 / netTickRate)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				t.Fatal("connection closed before snapshot")
			}
			if m.Type != msgSnapshot || m.Snap == nil {
				continue
			}
			if !m.Snap.Full {
				continue
			}
			if len(m.Snap.Players) != 1 {
				t.Fatalf("snapshot has %d players, want 1", len(m.Snap.Players))
			}
			ps := m.Snap.Players[0]
			if ps.ID != c.id || ps.Name != "TESTER" {
				t.Fatalf("player state mismatch: %+v (client id %d)", ps, c.id)
			}
			if len(m.Snap.Props) == 0 {
				t.Fatal("full snapshot contained no props")
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for a snapshot")
		}
	}
}

// TestApplyNetSnapshotSpawnsAndRemovesProps validates the client-side mirror
// without needing a window or a live connection.
func TestApplyNetSnapshotSpawnsAndRemovesProps(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	g.netProps = make(map[int32]*Entity)
	g.netRemote = make(map[int32]*netRemotePlayer)

	p := findPreset("Crate")
	if p == nil {
		t.Fatal("missing Crate preset")
	}
	snap := &netSnapshot{
		Tick: 1,
		Full: true,
		Props: []netPropState{{
			ID:    77,
			Name:  p.Name,
			Kind:  uint8(p.Kind),
			Half:  netVecFromBv(p.Half),
			Pos:   netVec3{X: 3, Y: 4, Z: 5},
			Rot:   netQuat{W: 1},
			Color: packColor(rgb(1, 2, 3)),
		}},
	}
	g.applyNetSnapshot(snap)

	e := g.netProps[77]
	if e == nil {
		t.Fatal("snapshot prop was not spawned")
	}
	if !e.NetVisual || e.NetID != 77 || !e.Static {
		t.Fatalf("snapshot prop flags wrong: %+v", e)
	}
	if e.NetTargetPos.X != 3 || e.NetTargetPos.Y != 4 || e.NetTargetPos.Z != 5 {
		t.Fatalf("target pose mismatch: %+v", e.NetTargetPos)
	}
	if _, ok := g.byBody[e.Body]; !ok {
		t.Fatal("snapshot prop was not indexed by body")
	}

	// A full snapshot that omits the prop must remove it.
	g.applyNetSnapshot(&netSnapshot{Tick: 2, Full: true})
	if _, ok := g.netProps[77]; ok {
		t.Fatal("prop survived a full snapshot that omitted it")
	}
	if _, ok := g.byBody[e.Body]; ok {
		t.Fatal("removed prop is still indexed by body")
	}
}

// TestNetServerBroadcastAndKick validates the server-side helpers exposed to
// addons: toast, custom event forwarding and a clean kick.
func TestNetServerBroadcastAndKick(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	g.selectedMap = "default"
	srv := newNetServer(g, 0)
	if err := srv.start(); err != nil {
		t.Fatalf("server listen: %v", err)
	}
	defer srv.stop()

	tcp, ok := srv.ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address %T", srv.ln.Addr())
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(tcp.Port))

	c, err := dialNetClient(addr, "LISTENER")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.close()

	// Process the join command so the client is a full session.
	for i := 0; i < 8 && len(srv.clients) == 0; i++ {
		srv.tickOnce(1.0 / netTickRate)
		time.Sleep(10 * time.Millisecond)
	}
	if len(srv.clients) != 1 {
		t.Fatalf("server has %d clients, want 1", len(srv.clients))
	}

	srv.broadcastToast("HELLO CLIENTS", rgb(240, 173, 78))
	srv.broadcastEvent("round.start", map[string]any{"round": 2})

	gotToast, gotEvent := false, false
	deadline := time.After(3 * time.Second)
	for !gotToast || !gotEvent {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				t.Fatal("connection closed before broadcast arrived")
			}
			switch m.Type {
			case msgToast:
				if m.Text != "HELLO CLIENTS" {
					t.Fatalf("toast text = %q", m.Text)
				}
				gotToast = true
			case msgEvent:
				if m.Event != "round.start" {
					t.Fatalf("event name = %q", m.Event)
				}
				data, ok := m.Data.(map[string]any)
				if !ok || data["round"] != float64(2) {
					t.Fatalf("event data = %#v", m.Data)
				}
				gotEvent = true
			}
		case <-deadline:
			t.Fatalf("timed out waiting for broadcast (toast=%v event=%v)", gotToast, gotEvent)
		}
	}

	if !srv.kick(c.id, "test kick") {
		t.Fatal("kick returned false for a connected client")
	}
	if _, exists := srv.clients[c.id]; exists {
		t.Fatal("kicked client is still in the session map")
	}

	deadline = time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				return
			}
			if m.Type == msgBye {
				if m.Reason != "test kick" {
					t.Fatalf("bye reason = %q", m.Reason)
				}
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for kick bye")
		}
	}
}

// TestNetModAPIStatus checks the addon-facing net.* commands in single-player.
func TestNetModAPIStatus(t *testing.T) {
	t.Chdir(t.TempDir())

	g := newHeadlessGame(t)
	raw, err := g.AddonCall("addon.a", "net.status", nil)
	if err != nil {
		t.Fatalf("net.status: %v", err)
	}
	info, ok := raw.(NetStatusInfo)
	if !ok {
		t.Fatalf("net.status returned %T", raw)
	}
	if info.Mode != "off" || info.Server || info.Connected || info.Protocol != netProtocolVersion {
		t.Fatalf("unexpected offline status: %+v", info)
	}

	raw, err = g.AddonCall("addon.a", "net.players", nil)
	if err != nil {
		t.Fatalf("net.players: %v", err)
	}
	if players, ok := raw.([]NetPlayerInfo); !ok || len(players) != 0 {
		t.Fatalf("net.players = %#v", raw)
	}

	if _, err := g.AddonCall("addon.a", "net.broadcast_toast", json.RawMessage(`{"text":"hi"}`)); err == nil {
		t.Fatal("broadcast_toast should fail outside a server")
	}
	if _, err := g.AddonCall("addon.a", "net.kick", json.RawMessage(`{"id":1}`)); err == nil {
		t.Fatal("kick should fail outside a server")
	}
	if _, err := g.AddonCall("addon.a", "net.send_event", json.RawMessage(`{"event":"x"}`)); err == nil {
		t.Fatal("send_event should fail outside a session")
	}
}

func TestNetQuatRoundTrip(t *testing.T) {
	q := box3d.MulQuat(
		box3d.MakeQuatFromAxisAngle(box3d.Vec3{Y: 1}, 0.7),
		box3d.MakeQuatFromAxisAngle(box3d.Vec3{X: 1}, -0.2),
	)
	nq := netQuatFromBq(q)
	back := nq.BQ()
	if absf(back.V.X-q.V.X) > 1e-6 || absf(back.V.Y-q.V.Y) > 1e-6 ||
		absf(back.V.Z-q.V.Z) > 1e-6 || absf(back.S-q.S) > 1e-6 {
		t.Fatalf("quaternion round trip mismatch: %+v -> %+v", q, back)
	}
}
