package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	Code    int         `json:"code"`
	Message string      `json:"message"`
}

func sendResponse(id *json.RawMessage, result interface{}) {
	resultRaw, _ := json.Marshal(result)
	msg := RPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultRaw,
	}
	data, _ := json.Marshal(msg)
	os.Stdout.Write(append(data, '\n'))
	os.Stdout.Sync()
}

func sendError(id *json.RawMessage, code int, message string) {
	msg := RPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(msg)
	os.Stdout.Write(append(data, '\n'))
	os.Stdout.Sync()
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.Stderr.WriteString(fmt.Sprintf("[go mod] panic: %v\n", r))
			os.Stderr.Sync()
		}
	}()
	os.Stderr.WriteString("[go mod] worker starting\n")
	os.Stderr.Sync()

	reader := bufio.NewReaderSize(os.Stdin, 4096)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "[go mod] stdin read error: %v\n", err)
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg RPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			sendError(nil, -32700, "parse error")
			continue
		}

		switch msg.Method {
		case "mod.hello":
			var params struct {
				Game    string `json:"game"`
				Version string `json:"version"`
			}
			_ = json.Unmarshal(msg.Params, &params)
			os.Stderr.WriteString(fmt.Sprintf("[go mod] hello from engine: %s v%s\n", params.Game, params.Version))
			os.Stderr.Sync()
			sendResponse(msg.ID, map[string]string{"status": "ok", "mod": "example"})

		case "mod.tick":
			var params struct {
				Dt float64 `json:"dt"`
			}
			_ = json.Unmarshal(msg.Params, &params)
			_ = params.Dt
			sendResponse(msg.ID, map[string]string{"status": "tick"})

		default:
			sendError(msg.ID, -32601, "method not found")
		}
	}
	os.Stderr.WriteString("[go mod] worker stdin closed, exiting\n")
	os.Stderr.Sync()
}