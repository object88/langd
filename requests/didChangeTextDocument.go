package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/object88/langd"
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
	uri := rh.uri

	buf, ok := rh.h.openedFiles[uri]
	if !ok {
		return fmt.Errorf("File %s is not opened\n", uri)
	}

	for k, v := range rh.changes {
		fmt.Printf("%d: %s\n", k, v.String())

		if v.Range == nil || v.RangeLength == nil {
			// Replace the entire document
			buf = []byte(v.Text)
		} else {
			// Check to see if the change changes the buffer size or leaves it as-is.
			delta := len(v.Text) - *v.RangeLength

			// Have position (line, character), need to transform into offset into file
			// Then replace starting from there.
			startOffset, err := langd.CalculateOffsetForPosition(buf, v.Range.Start.Line, v.Range.Start.Character)
			if err != nil {
				// Crap crap crap crap.
			}

			if delta == 0 {
				// Can directly replace inside buf; no need to expand or shrink.
				copy(buf[startOffset:startOffset+*v.RangeLength], []byte(v.Text))

			} else if delta < 0 {
				// The text to be inserted is smaller than the original text, ie., the
				// file is shrinking.
				initialLength := len(buf)
				if len(v.Text) == 0 {
					fmt.Printf("Strict removal of %d chars\n", -delta)

					// We are only removing characters; there is nothing inserted as a replacement
					copy(buf[startOffset:], buf[startOffset+-delta:])
					buf = buf[:initialLength+delta]

				} else {

					// // destination is []

					// copy(buf[startOffset+len(v.Text):])
				}

			} else {
				// The text to be inserted is larger than the original text, ie., the
				// file is growing.
				bytes := bytes.NewBuffer(buf)

				// Want to check first to see if the new size is within capacity for
				// the buffer.  If it is, then we want to shift the existing bytes
				// after the edit, and overwrite the new parts.

				// Otherwise, we will want to reallocate the entire buffer and copy
				// the whole part in.

				// Maybe.  The implementation of Buffer.Grow needs to be studied
				// further.

				// Note: this can panic if we run out of memory.
				bytes.Grow(delta)

			}
		}

	}

	rh.h.openedFiles[uri] = buf

	return nil
}
