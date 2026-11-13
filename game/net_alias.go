package game

// Aliases into the netproto package so the rest of the code can keep using the
// unqualified wire-protocol identifiers.

import (
	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"

	"sandbox3d/network"
)

type (
	netVec3        = network.Vec3
	netQuat        = network.Quat
	netInput       = network.Input
	netAction      = network.Action
	netPropState   = network.PropState
	netPlayerState = network.PlayerState
	netSnapshot    = network.Snapshot
	netMessage     = network.Message
)

const (
	netProtocolVersion = network.ProtocolVersion
	netDefaultPort     = network.DefaultPort

	netTickRate          = network.TickRate
	netSnapshotRate      = network.SnapshotRate
	netInputRate         = network.InputRate
	netFullSnapshotEvery = network.FullSnapshotEvery

	netMaxNameLen = network.MaxNameLen
)

const (
	msgHello    = network.MsgHello
	msgWelcome  = network.MsgWelcome
	msgInput    = network.MsgInput
	msgAction   = network.MsgAction
	msgSnapshot = network.MsgSnapshot
	msgToast    = network.MsgToast
	msgEvent    = network.MsgEvent
	msgBye      = network.MsgBye
)

func netVecFromRl(v rl.Vector3) netVec3  { return network.VecFromRl(v) }
func netVecFromBp(p box3d.Pos) netVec3   { return network.VecFromBp(p) }
func netVecFromBv(v box3d.Vec3) netVec3  { return network.VecFromBv(v) }
func netQuatFromBq(q box3d.Quat) netQuat { return network.QuatFromBq(q) }
func packColor(c rl.Color) uint32        { return network.PackColor(c) }
func unpackColor(v uint32) rl.Color      { return network.UnpackColor(v) }
func propStateChanged(a, b netPropState) bool {
	return network.PropStateChanged(a, b)
}
func sanitizeNetName(name string) string { return network.SanitizeName(name) }

// propStateFromEntity snapshots the authoritative pose and animation state of
// one dynamic prop.
func propStateFromEntity(e *Entity) netPropState {
	p := e.Body.GetPosition()
	q := e.Body.GetRotation()
	anim := int32(0)
	if e.AnimCurrent > 0 {
		anim = e.AnimCurrent
	}
	return netPropState{
		ID:          e.ID,
		Name:        e.Name,
		Kind:        uint8(e.Kind),
		Half:        netVecFromBv(e.Half),
		Radius:      e.Radius,
		Height:      e.Height,
		Pos:         netVecFromBp(p),
		Rot:         netQuatFromBq(q),
		Color:       packColor(e.Color),
		Frozen:      e.Frozen,
		Anim:        uint8(anim),
		AnimFrame:   e.AnimFrame,
		AnimPlaying: e.AnimPlaying,
		AnimActive:  e.AnimActive,
		AnimSpeed:   e.AnimSpeed,
	}
}
