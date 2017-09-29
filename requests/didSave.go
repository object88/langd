package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didSaveNotification = "textDocument/didSave"
)

type didSaveHandler struct {
	requestBase

	fpath string
	text  string
}

func createDidSaveHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didSaveHandler{
		requestBase: createRequestBase(ctx, h, req),
	}

	return rh
}

func (rh *didSaveHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Debugf("Received exit request\n")

	var typedParams DidSaveTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return fmt.Errorf("Failed to unmarshal params")
	}

	fmt.Printf("Got parameters: %#v\n", typedParams)

	uri := string(typedParams.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	rh.fpath = fpath
	rh.text = *typedParams.Text
	return nil
}

func (rh *didSaveHandler) work() error {
	// Sanity check: is our in-memory document different from the one that
	// the client is saving?

	r := rh.h.openedFiles[rh.fpath]
	if r.ByteLength() != len(rh.text) {
		// Not the same length, different file.
		fmt.Printf("in-memory:\n%s\nprovided:\n%s\n", r.String(), rh.text)
		return fmt.Errorf("%s: In-memory does not match client version; different length: %d/%d", rh.fpath, r.ByteLength(), len(rh.text))
	}

	rtext := r.String()
	for i := 0; i < len(rtext); i++ {
		if rtext[i] != rh.text[i] {
			// Different byte; different file.
			fmt.Printf("in-memory:\n%s\nprovided:\n%s\n", rtext, rh.text)
			return fmt.Errorf("%s: In-memory does not match client version; starting at %d", rh.fpath, i)
		}
	}

	return nil
}
