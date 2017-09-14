package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didChangeConfigurationMethod = "workspace/didChangeConfiguration"
)

// DidChangeConfigurationParams contains all settings that have changed
type DidChangeConfigurationParams struct {
	// Settings are the changed settings
	Settings map[string]interface{} `json:"settings"`
}

func (h *Handler) didChangeConfiguration(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
	h.log.Verbosef("Got '%s'\n", didChangeConfigurationMethod)

	var params DidChangeConfigurationParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return noopHandleFuncer
	}

	h.log.Verbosef("All changes: %#v\n", params)

	return noopHandleFuncer
}
