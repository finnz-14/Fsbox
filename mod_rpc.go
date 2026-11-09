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

const RPCProtocolVersion = "1.0.0"

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

const (
	RPCErrParse      = -32700
	RPCErrInvalidReq = -32600
	RPCErrMethod     = -32601
	RPCErrParams     = -32602
	RPCErrInternal   = -32603
)

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
	WriteMu  sync.Mutex

	// OnRequest handles inbound JSON-RPC requests from the worker (e.g.
	// engine.call). It must be safe to call from the read goroutine; heavy work
	// should be queued to the main thread.
	OnRequest func(method string, params json.RawMessage) (any, error)
}

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
	time.Sleep(100 * time.Millisecond)
	return w, nil
}

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
		if msg.Method != "" {
			// Inbound request or notification.
			if msg.ID == nil {
				// notification: ignore
				continue
			}
			go w.handleRequest(msg)
			continue
		}
		if msg.ID != nil {
			var id int64
			if err := json.Unmarshal(*msg.ID, &id); err == nil {
				w.PendingMu.Lock()
				if ch, ok := w.Pending[id]; ok {
					delete(w.Pending, id)
					w.PendingMu.Unlock()
					ch <- &msg
					close(ch)
				} else {
					w.PendingMu.Unlock()
				}
			}
		}
	}
}

func (w *GoModWorker) handleRequest(msg RPCMessage) {
	var result any
	var rpcErr *RPCError
	if w.OnRequest != nil {
		r, err := w.OnRequest(msg.Method, msg.Params)
		if err != nil {
			rpcErr = &RPCError{Code: RPCErrInternal, Message: err.Error()}
		} else {
			result = r
		}
	} else {
		rpcErr = &RPCError{Code: RPCErrMethod, Message: "method not found"}
	}
	resp := RPCMessage{JSONRPC: "2.0", ID: msg.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		raw, _ := json.Marshal(result)
		resp.Result = raw
	}
	data, _ := json.Marshal(resp)
	w.WriteMu.Lock()
	defer w.WriteMu.Unlock()
	w.Stdin.Write(append(data, '\n'))
}

func (w *GoModWorker) monitor() {
	err := w.Cmd.Wait()
	fmt.Printf("[mod] worker %s exited: %v\n", w.ID, err)
	w.ExitedMu.Lock()
	w.Exited = true
	w.ExitedMu.Unlock()
}

func (w *GoModWorker) Call(method string, params interface{}) (*RPCMessage, error) {
	w.SeqMu.Lock()
	w.Seq++
	id := w.Seq
	w.SeqMu.Unlock()

	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	msg := RPCMessage{JSONRPC: "2.0", Method: method, Params: paramsRaw}
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

	w.WriteMu.Lock()
	_, err = fmt.Fprintln(w.Stdin, string(data))
	w.WriteMu.Unlock()
	if err != nil {
		w.PendingMu.Lock()
		delete(w.Pending, id)
		w.PendingMu.Unlock()
		return nil, fmt.Errorf("send request: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-time.After(10 * time.Second):
		w.PendingMu.Lock()
		delete(w.Pending, id)
		w.PendingMu.Unlock()
		w.ExitedMu.Lock()
		exited := w.Exited
		w.ExitedMu.Unlock()
		if exited {
			return nil, fmt.Errorf("worker %s exited", w.ID)
		}
		return nil, fmt.Errorf("worker %s timeout", w.ID)
	}
}

// Notify sends a JSON-RPC notification (no response expected).
func (w *GoModWorker) Notify(method string, params interface{}) {
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return
	}
	msg := RPCMessage{JSONRPC: "2.0", Method: method, Params: paramsRaw}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	w.WriteMu.Lock()
	defer w.WriteMu.Unlock()
	fmt.Fprintln(w.Stdin, string(data))
}

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


