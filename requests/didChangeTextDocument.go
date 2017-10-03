package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/object88/langd"
	"github.com/object88/rope"
	"github.com/sourcegraph/jsonrpc2"
)

const (
	didChangeTextDocumentNotification = "textDocument/didChange"
)

type didChangeTextDocumentHandler struct {
	requestBase

	uri     string
	changes []TextDocumentContentChangeEvent
}

func createDidChangeTextDocumentHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didChangeTextDocumentHandler{
		requestBase: createRequestBase(ctx, h, req, true),
	}

	return rh
}

func (rh *didChangeTextDocumentHandler) preprocess(params *json.RawMessage) error {
	var typedParams DidChangeTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return nil
	}

	uri := string(typedParams.TextDocument.URI)
	uri = strings.TrimPrefix(uri, "file://")

	rh.uri = uri
	rh.changes = typedParams.ContentChanges

	return nil
}

func (rh *didChangeTextDocumentHandler) work() error {
	// Wait in this until we have a way to validate the edits.
	uri := rh.uri

	buf, ok := rh.h.workspace.OpenedFiles[uri]
	if !ok {
		return fmt.Errorf("File %s is not opened\n", uri)
	}

	for k, v := range rh.changes {
		if v.Range == nil || v.RangeLength == nil {
			// Replace the entire document
			buf = rope.CreateRope(v.Text)
		} else {
			// Have position (line, character), need to transform into offset into file
			// Then replace starting from there.
			r1 := buf.NewReader()
			startOffset, err := langd.CalculateOffsetForPosition(r1, v.Range.Start.Line, v.Range.Start.Character)
			if err != nil {
				// Crap crap crap crap.
				fmt.Printf("Error from start: %s", err.Error())
			}

			r2 := buf.NewReader()
			endOffset, err := langd.CalculateOffsetForPosition(r2, v.Range.End.Line, v.Range.End.Character)
			if err != nil {
				// Crap crap crap crap.
				fmt.Printf("Error from end: %s", err.Error())
			}

			fmt.Printf("%d: offsets: [%d:%d]; %s\n", k, startOffset, endOffset, v.String())

			buf.Alter(startOffset, endOffset, v.Text)
		}
	}

	rh.h.workspace.OpenedFiles[uri] = buf

	return nil
}
