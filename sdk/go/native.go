//go:build !wasip1

package fsbox

import "encoding/json"

// EngineCallResult is the standard envelope returned by engine.call.
type EngineCallResult struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}
