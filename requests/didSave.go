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
	text  []byte
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
		// return noopHandleFuncer
	}

	fmt.Printf("Got parameters: %#v\n", typedParams)

	uri := string(typedParams.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	rh.fpath = fpath
	rh.text = []byte(*typedParams.Text)
	return nil
}

func (rh *didSaveHandler) work() error {
	// Sanity check: is our in-memory document different from the one that
	// the client is saving?

	f := rh.h.openedFiles[rh.fpath]
	if len(f) != len(rh.text) {
		// Not the same length, different file.
		return fmt.Errorf("%s: In-memory does not match client version; different length: %d/%d", rh.fpath, len(f), len(rh.text))
	}

	for i := 0; i < len(f); i++ {
		if f[i] != rh.text[i] {
			// Different byte; different file.
			return fmt.Errorf("%s: In-memory does not match client version; starting at %d", rh.fpath, i)
		}
	}

	return nil
}
