package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	initializeMethod = "initialize"
)

func (h *Handler) processInit(p *json.RawMessage) (interface{}, error) {
	fmt.Printf("Got initialize method\n")

	// fmt.Printf("Raw init params: %s\n", string(*p))

	var params InitializeParams
	if err := json.Unmarshal(*p, &params); err != nil {
		return nil, err
	}

	rootURI := string(params.RootURI)
	fmt.Printf("Got parameters: %#v\n", params)

	h.hFunc = h.initedHandler

	go h.readRoot(rootURI)

	results := &InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncOptions{
				Change:    Incremental,
				OpenClose: true,
				Save: &SaveOptions{
					IncludeText: true,
				},
				WillSave: true,
			},
			HoverProvider:                    false,
			CompletionProvider:               nil,
			SignatureHelpProvider:            nil,
			DefinitionProvider:               true,
			ReferencesProvider:               true,
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

	// l := langd.NewLoader()
	h.workspace.Loader.Start(base)
	done := h.workspace.Loader.LoadDirectory(base, true)
	// if loadErr != nil {
	// 	fmt.Printf("OHSHANP: %s\n", loadErr.Error())
	// }

	// NOTE: We are not doing anything with this, so... BLOCKED.
	<-done

	fmt.Printf("Have %d imports...\n", len(h.workspace.PkgNames))

	// Start a routine to process requests
	h.startProcessingQueue()
}
