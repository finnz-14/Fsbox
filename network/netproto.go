// Package netproto defines the wire protocol for dedicated-server
// multiplayer.
//
// Transport is TCP with newline-free JSON frames produced by json.Encoder /
// json.Decoder, so every message is length-delimited by the decoder itself.
// The server is authoritative: clients send input and intents, the server
// simulates physics and broadcasts snapshots.
package network

import (
	"math"
	"strings"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ProtocolVersion = 1
	DefaultPort     = 27015

	TickRate     = 60
	SnapshotRate = 20
	InputRate    = 30
	// A full snapshot (every prop) is sent this often so a client that missed
	// a delta for any reason always converges.
	FullSnapshotEvery = SnapshotRate * 5

	MaxNameLen = 20
)

type Vec3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type Quat struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	W float32 `json:"w"`
}

func VecFromRl(v rl.Vector3) Vec3 { return Vec3{X: v.X, Y: v.Y, Z: v.Z} }
func VecFromBp(p box3d.Pos) Vec3  { return Vec3{X: p.X, Y: p.Y, Z: p.Z} }
func VecFromBv(v box3d.Vec3) Vec3 { return Vec3{X: v.X, Y: v.Y, Z: v.Z} }
func (v Vec3) RL() rl.Vector3     { return rl.Vector3{X: v.X, Y: v.Y, Z: v.Z} }
func (v Vec3) BP() box3d.Pos      { return box3d.Pos{X: v.X, Y: v.Y, Z: v.Z} }
func (v Vec3) BV() box3d.Vec3     { return box3d.Vec3{X: v.X, Y: v.Y, Z: v.Z} }

func QuatFromBq(q box3d.Quat) Quat {
	return Quat{X: q.V.X, Y: q.V.Y, Z: q.V.Z, W: q.S}
}

func (q Quat) BQ() box3d.Quat {
	return box3d.Quat{V: box3d.Vec3{X: q.X, Y: q.Y, Z: q.Z}, S: q.W}
}

func (q Quat) RL() rl.Quaternion {
	return rl.Quaternion{X: q.X, Y: q.Y, Z: q.Z, W: q.W}
}

func PackColor(c rl.Color) uint32 {
	return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16 | uint32(c.A)<<24
}

func UnpackColor(v uint32) rl.Color {
	return rl.Color{R: uint8(v), G: uint8(v >> 8), B: uint8(v >> 16), A: uint8(v >> 24)}
}

// Input is one client's movement intent plus held-prop controls.
type Input struct {
	Forward  float32 `json:"f"`
	Right    float32 `json:"r"`
	Jump     bool    `json:"j,omitempty"`
	Down     bool    `json:"d,omitempty"`
	Sprint   bool    `json:"s,omitempty"`
	Yaw      float32 `json:"y"`
	Pitch    float32 `json:"p"`
	HoldRotX float32 `json:"rx,omitempty"`
	HoldRotY float32 `json:"ry,omitempty"`
	HoldDist float32 `json:"hd,omitempty"`
}

// Action is a one-shot world intent (spawn, grab, delete, tool use, ...).
type Action struct {
	Kind   string  `json:"k"`
	ID     int32   `json:"id,omitempty"`
	Preset string  `json:"preset,omitempty"`
	Pos    Vec3    `json:"pos,omitempty"`
	Rot    Quat    `json:"rot,omitempty"`
	Color  uint32  `json:"c,omitempty"`
	Value  float32 `json:"v,omitempty"`
	Flag   bool    `json:"b,omitempty"`
	Tool   int     `json:"tool,omitempty"`
}

type PropState struct {
	ID     int32   `json:"id"`
	Name   string  `json:"n,omitempty"`
	Kind   uint8   `json:"k,omitempty"`
	Half   Vec3    `json:"h,omitempty"`
	Radius float32 `json:"r,omitempty"`
	Height float32 `json:"ht,omitempty"`
	Pos    Vec3    `json:"p"`
	Rot    Quat    `json:"q"`
	Color  uint32  `json:"c,omitempty"`
	Frozen bool    `json:"fz,omitempty"`

	Anim        uint8   `json:"a,omitempty"`
	AnimFrame   float32 `json:"af,omitempty"`
	AnimPlaying bool    `json:"ap,omitempty"`
	AnimActive  bool    `json:"aa,omitempty"`
	AnimSpeed   float32 `json:"as,omitempty"`
}

type PlayerState struct {
	ID    int32   `json:"id"`
	Name  string  `json:"n,omitempty"`
	Pos   Vec3    `json:"p"`
	Yaw   float32 `json:"y"`
	Pitch float32 `json:"pi"`
	Fly   bool    `json:"fl,omitempty"`
	Held  int32   `json:"h,omitempty"`
}

type Snapshot struct {
	Tick    uint64        `json:"tick"`
	Time    float64       `json:"time"`
	Full    bool          `json:"full,omitempty"`
	Players []PlayerState `json:"players,omitempty"`
	Props   []PropState   `json:"props,omitempty"`
	Removed []int32       `json:"removed,omitempty"`
}

// Message is the single envelope for every packet in both directions.
type Message struct {
	Type string `json:"t"`

	Protocol     int     `json:"proto,omitempty"`
	Name         string  `json:"name,omitempty"`
	Map          string  `json:"map,omitempty"`
	ClientID     int32   `json:"cid,omitempty"`
	TickRate     int     `json:"tr,omitempty"`
	SnapshotRate int     `json:"sr,omitempty"`
	Spawn        Vec3    `json:"spawn,omitempty"`
	SpawnYaw     float32 `json:"sy,omitempty"`

	Input  *Input    `json:"in,omitempty"`
	Action *Action   `json:"a,omitempty"`
	Snap   *Snapshot `json:"s,omitempty"`

	Text   string `json:"text,omitempty"`
	Color  uint32 `json:"c,omitempty"`
	Reason string `json:"reason,omitempty"`

	// Event carries a server-authored addon event to connected clients. Data is
	// arbitrary JSON produced by the sending addon.
	Event string `json:"ev,omitempty"`
	Data  any    `json:"data,omitempty"`
}

const (
	MsgHello    = "hello"
	MsgWelcome  = "welcome"
	MsgInput    = "input"
	MsgAction   = "action"
	MsgSnapshot = "snapshot"
	MsgToast    = "toast"
	MsgEvent    = "event"
	MsgBye      = "bye"
)

// PropStateChanged reports whether two snapshots differ enough to be worth
// sending. Pose is the hot path; animation flags are compared cheaply.
func PropStateChanged(a, b PropState) bool {
	const posEps = 0.0015
	const rotEps = 0.00002
	if a.Frozen != b.Frozen || a.Anim != b.Anim || a.AnimPlaying != b.AnimPlaying ||
		a.AnimActive != b.AnimActive || a.AnimSpeed != b.AnimSpeed {
		return true
	}
	if math.Abs(float64(a.AnimFrame)-float64(b.AnimFrame)) > 0.01 {
		return true
	}
	if math.Abs(float64(a.Pos.X-b.Pos.X)) > posEps ||
		math.Abs(float64(a.Pos.Y-b.Pos.Y)) > posEps ||
		math.Abs(float64(a.Pos.Z-b.Pos.Z)) > posEps {
		return true
	}
	d := math.Abs(float64(a.Rot.X-b.Rot.X)) + math.Abs(float64(a.Rot.Y-b.Rot.Y)) +
		math.Abs(float64(a.Rot.Z-b.Rot.Z)) + math.Abs(float64(a.Rot.W-b.Rot.W))
	return d > rotEps
}

// SanitizeName trims a display name to a safe, printable length.
func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	out := make([]rune, 0, MaxNameLen)
	for _, r := range name {
		if r < 32 || r == 127 {
			continue
		}
		out = append(out, r)
		if len(out) >= MaxNameLen {
			break
		}
	}
	if len(out) == 0 {
		return "PLAYER"
	}
	return string(out)
}
