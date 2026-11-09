package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type RPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   = make(map[int64]chan *RPCMessage)
	nextID    int64
	incoming  = make(chan RPCMessage, 64)
)

func writeMsg(msg RPCMessage) {
	data, _ := json.Marshal(msg)
	writeMu.Lock()
	defer writeMu.Unlock()
	os.Stdout.Write(append(data, '\n'))
	os.Stdout.Sync()
}

func sendResponse(id *json.RawMessage, result interface{}) {
	resultRaw, _ := json.Marshal(result)
	writeMsg(RPCMessage{JSONRPC: "2.0", ID: id, Result: resultRaw})
}

func sendError(id *json.RawMessage, code int, message string) {
	writeMsg(RPCMessage{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: message}})
}

func engineCall(cmd string, args interface{}) (json.RawMessage, error) {
	pendingMu.Lock()
	nextID++
	id := nextID
	ch := make(chan *RPCMessage, 1)
	pending[id] = ch
	pendingMu.Unlock()

	idRaw := json.RawMessage(fmt.Sprintf("%d", id))
	paramsRaw, _ := json.Marshal(map[string]any{"cmd": cmd, "args": args})
	writeMsg(RPCMessage{JSONRPC: "2.0", ID: &idRaw, Method: "engine.call", Params: paramsRaw})

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		var r struct {
			OK     bool            `json:"ok"`
			Result json.RawMessage `json:"result"`
			Error  string          `json:"error"`
		}
		_ = json.Unmarshal(resp.Result, &r)
		if !r.OK {
			return nil, fmt.Errorf("engine error: %s", r.Error)
		}
		return r.Result, nil
	case <-time.After(10 * time.Second):
		pendingMu.Lock()
		delete(pending, id)
		pendingMu.Unlock()
		return nil, fmt.Errorf("engine call timeout")
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[go mod] panic: %v\n", r)
		}
	}()
	fmt.Fprintln(os.Stderr, "[go mod] worker starting")

	// Reader goroutine.
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			var msg RPCMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}
			if msg.Method != "" {
				incoming <- msg
				continue
			}
			if msg.ID != nil {
				var id int64
				if err := json.Unmarshal(*msg.ID, &id); err == nil {
					pendingMu.Lock()
					if ch, ok := pending[id]; ok {
						delete(pending, id)
						pendingMu.Unlock()
						ch <- &msg
						close(ch)
					} else {
						pendingMu.Unlock()
					}
				}
			}
		}
		fmt.Fprintln(os.Stderr, "[go mod] stdin closed")
		os.Exit(0)
	}()

	for msg := range incoming {
		switch msg.Method {
		case "mod.init":
			fmt.Fprintln(os.Stderr, "[go mod] init")
			// Example: spawn a box via the engine.
			res, err := engineCall("entity.spawn", map[string]any{
				"kind": "box",
				"position": map[string]float32{"x": 0, "y": 3, "z": 0},
				"color":    map[string]any{"r": 100, "g": 200, "b": 100, "a": 255},
			})
			fmt.Fprintf(os.Stderr, "[go mod] spawn result: %s err=%v\n", string(res), err)
			sendResponse(msg.ID, map[string]string{"status": "ok", "mod": "go.example.mod"})

		case "mod.tick":
			// notification; no response
		case "mod.event":
			// notification
		case "mod.hook":
			// notification
		default:
			if msg.ID != nil {
				sendError(msg.ID, -32601, "method not found")
			}
		}
	}
}
