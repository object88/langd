package server

import (
	"context"
	"encoding/json"
	"fmt"

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

func (h *Handler) didChangeConfiguration(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	fmt.Printf("Got '%s'\n", didChangeConfigurationMethod)

	var params DidChangeConfigurationParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	fmt.Printf("All changes: %#v\n", params)
}
