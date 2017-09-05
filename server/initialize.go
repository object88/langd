package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	initializeMethod = "initialize"
)

func initialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	fmt.Printf("Got initialize method\n")

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	fmt.Printf("Got parameters: %#v\n", params)

	results := &InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncOptions{
				Change: 0,
			},
			HoverProvider:                    false,
			CompletionProvider:               nil,
			SignatureHelpProvider:            nil,
			DefinitionProvider:               false,
			ReferencesProvider:               false,
			DocumentHighlightProvider:        false,
			DocumentSymbolProvider:           false,
			WorkspaceSymbolProvider:          false,
			CodeActionProvider:               false,
			CodeLensProvider:                 nil,
			DocumentFormattingProvider:       false,
			DocumentRangeFormattingProvider:  false,
			DocumentOnTypeFormattingProvider: nil,
			RenameProvider:                   false,
		},
	}

	err := conn.Reply(ctx, req.ID, results)
	if err != nil {
		fmt.Printf("Reply got error: %s\n", err.Error())
	}
	fmt.Printf("Responded to initialization request\n")
}
