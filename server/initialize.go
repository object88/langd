package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/object88/langd"
	"github.com/sourcegraph/jsonrpc2"
)

const (
	initializeMethod = "initialize"
)

func (h *Handler) initialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	fmt.Printf("Got initialize method\n")

	initParams := string(*req.Params)
	fmt.Printf("Raw init params: %s\n", initParams)

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	fmt.Printf("Got parameters: %#v\n", params)

	// Example:
	// server.InitializeParams {
	//   ProcessID: 62151,
	//   RootURI: "file:///Users/bropa18/work/src/github.com/object88/immutable",
	//   InitializationOptions: interface {}(nil),
	//   Capabilities: server.ClientCapabilities{},
	// }

	results := &InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncOptions{
				Change: 0,
			},
			HoverProvider:                    false,
			CompletionProvider:               nil,
			SignatureHelpProvider:            nil,
			DefinitionProvider:               true,
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

	go h.readRoot(string(params.RootURI))
}

func (h *Handler) readRoot(root string) {
	base := strings.TrimPrefix(root, "file://")

	sm := &ShowMessageParams{
		Type:    Info,
		Message: fmt.Sprintf("Loading AST for '%s'", base),
	}

	if err := h.conn.Notify(context.Background(), "window/showMessage", sm); err != nil {
		fmt.Printf("Failed to deliver message to client: %s\n", err.Error())
	}

	l := langd.NewLoader()
	w, loadErr := l.Load(context.Background(), base)
	if loadErr != nil {
		fmt.Printf("OHSHANP: %s\n", loadErr.Error())
	}
	fmt.Printf("Have %d imports...\n", len(w.Pkgs))

	h.workspace = w
}
