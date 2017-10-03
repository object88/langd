package requests

import (
	"context"
	"encoding/json"
	"go/token"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	referencesMethod = "textDocument/references"
)

// referenceHandler implements the `Find References` request
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#textDocument_references
type referencesHandler struct {
	requestBase

	p      *token.Position
	result *[]Location
}

func createReferencesHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &referencesHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *referencesHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got references method\n")

	// Example:
	// requests.ReferenceParams {
	// 	TextDocumentPositionParams: requests.TextDocumentPositionParams {
	// 		TextDocument:requests.TextDocumentIdentifier {
	// 			URI:"file:///Users/bropa18/work/src/github.com/object88/immutable/intToStringHashmap.go"
	// 		},
	// 		Position: requests.Position{
	// 			Line: 11,
	// 			Character: 14
	// 		}
	// 	},
	// 	Context: (*requests.ReferenceContext)
	// }
	var typedParams ReferenceParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return err
		// return noopHandleFuncer
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

func (rh *referencesHandler) work() error {
	return nil
}

func (rh *referencesHandler) reply() (interface{}, error) {
	return rh.result, nil
}
