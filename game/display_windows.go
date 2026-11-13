//go:build windows

package game

import (
	"syscall"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procGetWindowRect  = user32.NewProc("GetWindowRect")
	procGetWindowLongW = user32.NewProc("GetWindowLongW")
	procSetWindowLongW = user32.NewProc("SetWindowLongW")
	procSetWindowPos   = user32.NewProc("SetWindowPos")

	savedWindowRect winRect
	hasSavedRect    bool
)

const (
	wsOverlappedWindow = uintptr(0x00CF0000)
	wsPopup            = uintptr(0x80000000)
	gwlStyle           = ^uintptr(15) // -16
	swpFrameChanged    = 0x0020
	swpShowWindow      = 0x0040
)

func gameHWND() uintptr {
	return uintptr(rl.GetWindowHandle())
}

func saveWindowRect() {
	if hasSavedRect {
		return
	}
	var r winRect
	ret, _, _ := procGetWindowRect.Call(gameHWND(), uintptr(unsafe.Pointer(&r)))
	if ret != 0 {
		savedWindowRect = r
		hasSavedRect = true
	}
}

func getWindowStyle() uintptr {
	ret, _, _ := procGetWindowLongW.Call(gameHWND(), gwlStyle)
	return ret
}

func setWindowStyle(style uintptr) {
	procSetWindowLongW.Call(gameHWND(), gwlStyle, style)
}

func (g *Game) applyBorderlessWindowed() {
	saveWindowRect()
	style := getWindowStyle()
	style &^= wsOverlappedWindow
	style |= wsPopup
	setWindowStyle(style)

	mon := rl.GetCurrentMonitor()
	x := int32(rl.GetMonitorPosition(mon).X)
	y := int32(rl.GetMonitorPosition(mon).Y)
	w := int32(rl.GetMonitorWidth(mon))
	hh := int32(rl.GetMonitorHeight(mon))
	procSetWindowPos.Call(gameHWND(), 0xFFFFFFFE, uintptr(x), uintptr(y), uintptr(w), uintptr(hh), swpFrameChanged|swpShowWindow)
	rl.SetWindowPosition(int(x), int(y))
	rl.SetWindowSize(int(w), int(hh))
}

func (g *Game) restoreWindowed() {
	style := getWindowStyle()
	style &^= wsPopup
	style |= wsOverlappedWindow
	setWindowStyle(style)

	if hasSavedRect {
		r := savedWindowRect
		procSetWindowPos.Call(gameHWND(), 0xFFFFFFFE, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpFrameChanged|swpShowWindow)
		rl.SetWindowPosition(int(r.Left), int(r.Top))
		rl.SetWindowSize(int(r.Right-r.Left), int(r.Bottom-r.Top))
	} else {
		procSetWindowPos.Call(gameHWND(), 0xFFFFFFFE, 100, 100, 1280, 720, swpFrameChanged|swpShowWindow)
	}
}
