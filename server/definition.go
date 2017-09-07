package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	definitionMethod = "textDocument/definition"
)

// definition implements the `Goto Definition` request
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#goto-definition-request
func (h *Handler) definition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	fmt.Printf("Got definition method\n")

	var params TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	fmt.Printf("Got parameters: %#v\n", params)

	// Example:
	// server.TextDocumentPositionParams {
	//   TextDocument: server.TextDocumentIdentifier {
	// 		 URI: "file:///Users/bropa18/work/src/github.com/object88/immutable/memory/types.go",
	//   },
	// 	 Position: server.Position {
	//     Line: 7,
	//     Character: 15,
	//   }
	// }
}
