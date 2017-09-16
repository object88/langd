package requests

import (
	"context"
	"encoding/json"
	"strings"

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
		requestBase: createRequestBase(ctx, h, req),
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
	// uri := rh.uri

	// buf, ok := rh.h.openedFiles[uri]
	// if !ok {
	// 	return fmt.Errorf("File %s is not opened\n", uri)
	// }

	// for k, v := range rh.changes {
	// 	fmt.Printf("%d: %s\n", k, v.String())

	// 	if v.Range == nil || v.RangeLength == nil {
	// 		// Replace the entire document
	// 		buf = []byte(v.Text)
	// 	} else {
	// 		// Check to see if the change changes the buffer size or leaves it as-is.
	// 		delta := len(v.Text) - *v.RangeLength

	// 		if delta == 0 {
	// 			// Can directly replace inside buf; no need to expand or shrink.
	// 			// Have position (line, character), need to transform into offset into file
	// 			// Then replace starting from there.

	// 			insertOffset, err := langd.CalculateOffsetForPosition(buf, v.Range.Start.Line, v.Range.Start.Character)
	// 			if err != nil {
	// 				// Crap crap crap crap.
	// 			}
	// 			copy(buf[insertOffset:insertOffset+*v.RangeLength], []byte(v.Text))
	// 		}
	// 	}

	// }

	return nil
}
