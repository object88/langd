package requests

import (
	"context"
	"encoding/json"
	"go/token"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	hoverMethod = "textDocument/hover"
)

// hoverHandler implements the `Hover` request
// https://microsoft.github.io/language-server-protocol/specification#textDocument_hover
type hoverHandler struct {
	requestBase

	p      *token.Position
	result *Hover
}

func createHoverHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &hoverHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *hoverHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got hover\n")

	var typedParams TextDocumentPositionParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return err
	}

	rh.h.log.Verbosef("Got parameters: %#v\n", typedParams)

	path := strings.TrimPrefix(string(typedParams.TextDocument.URI), "file://")

	p := &token.Position{
		Filename: path,
		Line:     typedParams.Position.Line + 1,
		Column:   typedParams.Position.Character,
	}

	rh.p = p
	return nil
}

func (rh *hoverHandler) work() error {
	t, err := rh.h.workspace.Hover(rh.p)
	if err != nil {
		return err
	}

	// TODO: assign Range for interesting highlighting, etc.
	rh.result = &Hover{
		Contents: MarkupContent{
			Kind:  Markdown,
			Value: t,
		},
		Range: nil,
	}
	return nil
}

func (rh *hoverHandler) reply() (interface{}, error) {
	return rh.result, nil
}
