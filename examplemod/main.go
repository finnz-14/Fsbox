package main

import "unsafe"

//go:wasmimport fsbox log
func hostLog(ptr, size uint32)

//go:wasmimport fsbox toast
func hostToast(ptr, size uint32)

//go:wasmimport fsbox spawn_entity
func hostSpawnEntity(presetIdx, x, y, z float64)

//go:wasmimport fsbox remove_entity
func hostRemoveEntity(index int32)

//go:wasmimport fsbox set_entity_pos
func hostSetEntityPos(index, x, y, z float64)

//go:wasmimport fsbox add_entity_force
func hostAddEntityForce(index int32, fx, fy, fz float64)

//go:wasmimport fsbox add_entity_impulse
func hostAddEntityImpulse(index int32, ix, iy, iz float64)

//go:wasmimport fsbox add_entity_torque
func hostAddEntityTorque(index int32, tx, ty, tz float64)

//go:wasmimport fsbox set_entity_gravity
func hostSetEntityGravity(index int32, gravityScale float64)

//go:wasmimport fsbox raycast
func hostRaycast(maxDist float64)

//go:wasmimport fsbox set_player_pos
func hostSetPlayerPos(x, y, z float64)

//go:wasmimport fsbox set_player_rot
func hostSetPlayerRot(yaw, pitch float64)

//go:wasmimport fsbox set_fly
func hostSetFly(enabled int32)

//go:wasmimport fsbox get_player_x
func hostGetPlayerX() float64

//go:wasmimport fsbox register_tick_hook
func hostRegisterTickHook(ptr, size uint32)

func logMsg(msg string) {
	if len(msg) == 0 {
		return
	}
	ptr := uintptr(unsafe.Pointer(unsafe.StringData(msg)))
	hostLog(uint32(ptr), uint32(len(msg)))
}

func toastMsg(msg string) {
	if len(msg) == 0 {
		return
	}
	ptr := uintptr(unsafe.Pointer(unsafe.StringData(msg)))
	hostToast(uint32(ptr), uint32(len(msg)))
}

//go:wasmexport mod_init
func modInit() {
	logMsg("example mod initialized")
	toastMsg("Example mod loaded")
}

//go:wasmexport mod_update
func modUpdate() {
	// Called every frame by the engine.
}

func main() {
}