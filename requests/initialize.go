package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/object88/langd"
)

const (
	initializeMethod = "initialize"
)

func (h *Handler) processInit(p *json.RawMessage) (interface{}, error) {
	fmt.Printf("Got initialize method\n")

	initParams := string(*p)
	fmt.Printf("Raw init params: %s\n", initParams)

	var params InitializeParams
	if err := json.Unmarshal(*p, &params); err != nil {
		return nil, err
	}

	rootURI := string(params.RootURI)
	fmt.Printf("Got parameters: %#v\n", params)

	h.hFunc = h.initedHandler

	// Special case; normally we would want to start "work" in the `work`
	// method, but since the queue processor isn't running yet, we can't
	// queue up work.
	go h.readRoot(rootURI)

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
	return results, nil
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
	fmt.Printf("Have %d imports...\n", len(w.PkgNames))

	h.workspace = w

	// Start a routine to process requests
	h.startProcessingQueue()
}
