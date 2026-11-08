package main

import "unsafe"

//go:wasmimport fsbox log
func hostLog(ptr, size uint32)

//go:wasmimport fsbox toast
func hostToast(ptr, size uint32)

//go:wasmimport fsbox spawn_box
func hostSpawnBox(x, y, z float64)

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
}

func main() {
}