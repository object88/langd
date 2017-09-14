package requests

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

type initializeHandler struct {
	requestBase

	rootURI string
}

func createInitializeHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	ih := &initializeHandler{
		requestBase: createRequestBase(ctx, h, req.ID),
	}

	return ih
}

func (ih *initializeHandler) preprocess(p *json.RawMessage) {
	fmt.Printf("Got initialize method\n")

	initParams := string(*p)
	fmt.Printf("Raw init params: %s\n", initParams)

	var params InitializeParams
	if err := json.Unmarshal(*p, &params); err != nil {
		return
	}

	ih.rootURI = string(params.RootURI)
	fmt.Printf("Got parameters: %#v\n", params)
}

func (ih *initializeHandler) work() {
	go ih.readRoot(ih.rootURI)
}

func (ih *initializeHandler) reply() interface{} {
	results := &InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncOptions{
				Change:    Incremental,
				OpenClose: true,
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
	return results
}

func (ih *initializeHandler) readRoot(root string) {
	base := strings.TrimPrefix(root, "file://")

	sm := &ShowMessageParams{
		Type:    Info,
		Message: fmt.Sprintf("Loading AST for '%s'", base),
	}

	if err := ih.h.conn.Notify(context.Background(), "window/showMessage", sm); err != nil {
		fmt.Printf("Failed to deliver message to client: %s\n", err.Error())
	}

	l := langd.NewLoader()
	w, loadErr := l.Load(context.Background(), base)
	if loadErr != nil {
		fmt.Printf("OHSHANP: %s\n", loadErr.Error())
	}
	fmt.Printf("Have %d imports...\n", len(w.PkgNames))

	ih.h.workspace = w
}
