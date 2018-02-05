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
	text  *string
}

func createDidSaveHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didSaveHandler{
		requestBase: createRequestBase(ctx, h, req, true),
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
	rh.text = typedParams.Text
	return nil
}

func (rh *didSaveHandler) work() error {
	// TODO: is our in-memory document different from the one that
	// the client is saving?  If not, skip the `ReplaceText` call.

	if rh.text != nil {
		// Provided new contents for the file; update.
		rh.h.workspace.ReplaceFile(rh.fpath, *rh.text)
	}

	return nil
}
