package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// RPCProtocolVersion is the version of the mod engine host ABI.
const RPCProtocolVersion = "1.0.0"

// RPCMessage is a JSON-RPC 2.0 message between the engine and a Go mod worker.
type RPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
}

const (
	RPCErrParse      = -32700
	RPCErrInvalidReq = -32600
	RPCErrMethod     = -32601
	RPCErrParams     = -32602
	RPCErrInternal   = -32603
)

// GoModWorker manages an isolated Go mod worker process communicating via stdio JSON-RPC.
type GoModWorker struct {
	ID       string
	Cmd      *exec.Cmd
	Stdin    io.WriteCloser
	Stdout   io.ReadCloser
	Seq      int64
	SeqMu    sync.Mutex
	Pending  map[int64]chan *RPCMessage
	PendingMu sync.Mutex
	Exited   bool
	ExitedMu sync.Mutex
}

// NewGoModWorker launches a Go mod worker process.
// workDir is the module directory containing go.mod; entryPath is relative to it.
// The worker binary must be pre-built.
func NewGoModWorker(id, workDir, entryPath string, args []string) (*GoModWorker, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("abs workdir: %w", err)
	}
	binPath := filepath.Join(absWorkDir, "bin", "worker.exe")
	if _, err := os.Stat(binPath); err != nil {
		return nil, fmt.Errorf("worker binary not found at %s: %w", binPath, err)
	}
	cmd := exec.Command(binPath, args...)
	cmd.Dir = absWorkDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start worker: %w", err)
	}

	w := &GoModWorker{
		ID:       id,
		Cmd:      cmd,
		Stdin:    stdin,
		Stdout:   stdout,
		Pending:  make(map[int64]chan *RPCMessage),
		Exited:   false,
	}

	go w.readLoop()
	go w.monitor()

	// Give the worker a moment to initialize.
	time.Sleep(100 * time.Millisecond)

	return w, nil
}

// readLoop reads JSON-RPC messages from the worker's stdout.
func (w *GoModWorker) readLoop() {
	defer func() {
		w.ExitedMu.Lock()
		w.Exited = true
		w.ExitedMu.Unlock()
	}()
	dec := json.NewDecoder(w.Stdout)
	for {
		var msg RPCMessage
		if err := dec.Decode(&msg); err != nil {
			return
		}
		w.PendingMu.Lock()
		if msg.ID != nil {
			var id int64
			if err := json.Unmarshal(*msg.ID, &id); err == nil {
				if ch, ok := w.Pending[id]; ok {
					delete(w.Pending, id)
					fmt.Printf("[mod] rpc dispatching id %d to channel\n", id)
					ch <- &msg
					close(ch)
				}
			}
		}
		w.PendingMu.Unlock()
	}
}

// monitor waits for the worker process to exit.
func (w *GoModWorker) monitor() {
	err := w.Cmd.Wait()
	fmt.Printf("[mod] worker %s exited: %v\n", w.ID, err)
	w.ExitedMu.Lock()
	w.Exited = true
	w.ExitedMu.Unlock()
}

// Call sends a JSON-RPC request and waits for the response.
func (w *GoModWorker) Call(method string, params interface{}) (*RPCMessage, error) {
	w.SeqMu.Lock()
	w.Seq++
	id := w.Seq
	w.SeqMu.Unlock()

	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	msg := RPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsRaw,
	}
	idRaw := json.RawMessage(fmt.Sprintf("%d", id))
	msg.ID = &idRaw

	ch := make(chan *RPCMessage, 1)
	w.PendingMu.Lock()
	w.Pending[id] = ch
	w.PendingMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		w.PendingMu.Lock()
		delete(w.Pending, id)
		w.PendingMu.Unlock()
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if _, err := fmt.Fprintln(w.Stdin, string(data)); err != nil {
		w.PendingMu.Lock()
		delete(w.Pending, id)
		w.PendingMu.Unlock()
		return nil, fmt.Errorf("send request: %w", err)
	}
	fmt.Printf("[mod] rpc sent %s to %s\n", method, w.ID)

	select {
	case resp := <-ch:
		fmt.Printf("[mod] rpc response from %s\n", w.ID)
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-time.After(10 * time.Second):
		w.ExitedMu.Lock()
		exited := w.Exited
		w.ExitedMu.Unlock()
		if exited {
			return nil, fmt.Errorf("worker %s exited", w.ID)
		}
		return nil, fmt.Errorf("worker %s timeout", w.ID)
	}
}

// Close terminates the worker process.
func (w *GoModWorker) Close() error {
	if w.Stdin != nil {
		_ = w.Stdin.Close()
	}
	if w.Cmd != nil && w.Cmd.Process != nil {
		_ = w.Cmd.Process.Kill()
	}
	w.ExitedMu.Lock()
	exited := w.Exited
	w.ExitedMu.Unlock()
	if !exited {
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
