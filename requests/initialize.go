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
			HoverProvider:                    true,
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
		Type:    InfoMessageType,
		Message: fmt.Sprintf("Loading AST for '%s'", base),
	}

	if err := h.conn.Notify(context.Background(), "window/showMessage", sm); err != nil {
		fmt.Printf("Failed to deliver message to client: %s\n", err.Error())
	}

	done := h.workspace.Loader.Start()

	fmt.Printf("About to load %s\n", base)
	h.workspace.Loader.LoadDirectory(base)

	// NOTE: We are not doing anything with this, so... BLOCKED.
	fmt.Printf("Waiting...\n")
	<-done

	// Start a routine to process requests
	h.startProcessingQueue()

	// Send off some errors.
	h.workspace.Loader.Errors(func(file string, errs []langd.FileError) {
		params := &PublishDiagnosticsParams{
			URI:         DocumentURI("file://" + file),
			Diagnostics: make([]Diagnostic, len(errs)),
		}
		for k, e := range errs {
			s := ErrorDiagnosticSeverity
			if e.Warning {
				s = WarningDiagnosticSeverity
			}
			params.Diagnostics[k] = Diagnostic{
				Range: Range{
					Start: Position{
						Line:      e.Line - 1,
						Character: e.Column,
					},
					End: Position{
						Line:      e.Line - 1,
						Character: e.Column,
					},
				},
				Severity: &s,
				Message:  e.Message,
			}
		}
		publishDiagnostics(context.Background(), h.conn, params)
	})
}
