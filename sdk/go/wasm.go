//go:build wasip1

package fsbox

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

//go:wasmimport fsbox fsbox_call
func hostCall(reqPtr, reqLen, respPtr, respCap uint32) int32

var reqBuf [1 << 20]byte
var respBuf [1 << 20]byte

func reqPtr() uint32 { return uint32(uintptr(unsafe.Pointer(&reqBuf[0]))) }
func respPtr() uint32 { return uint32(uintptr(unsafe.Pointer(&respBuf[0]))) }

// Call invokes one engine command and returns its JSON result.
func Call(cmd string, args any) (json.RawMessage, error) {
	req := map[string]any{"cmd": cmd, "args": args}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if len(reqBytes) > len(reqBuf) {
		return nil, fmt.Errorf("request too large")
	}
	copy(reqBuf[:], reqBytes)
	written := hostCall(reqPtr(), uint32(len(reqBytes)), respPtr(), uint32(len(respBuf)))
	if written < 0 {
		return nil, fmt.Errorf("host call failed")
	}
	if int(written) > len(respBuf) {
		return nil, fmt.Errorf("response too large: %d", written)
	}
	var resp struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(respBuf[:written], &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Result, nil
}

// Exported helpers so the engine can push events and allocate memory.
//
//go:export mod_alloc
func ModAlloc(size uint32) uint32 { return respPtr() }

//go:export mod_scratch
func ModScratch(size uint32) uint32 { return respPtr() }

//go:export mod_event
func ModEvent(ptr, size uint32) {
	// Default no-op; mods can override by not using the SDK event dispatcher.
}
