package requests

import (
	"context"
	"encoding/json"
	"go/token"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	definitionMethod = "textDocument/definition"
)

// definitionHandler implements the `Goto Definition` request
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#goto-definition-request
type definitionHandler struct {
	requestBase

	p      *token.Position
	result *Location
}

func createDefinitionHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &definitionHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *definitionHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got definition method\n")

	// Example:
	// requests.TextDocumentPositionParams {
	//   TextDocument: requests.TextDocumentIdentifier {
	// 		 URI: "file:///Users/bropa18/work/src/github.com/object88/immutable/memory/types.go",
	//   },
	// 	 Position: requests.Position {
	//     Line: 7,
	//     Character: 15,
	//   }
	// }

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

func (rh *definitionHandler) work() error {
	x, err := rh.h.workspace.LocateIdent(rh.p)
	if err != nil {
		return err
	}

	declPosition, err := rh.h.workspace.LocateDeclaration(rh.p)
	if err != nil {
		return err
	}

	if declPosition != nil {
		rh.result = LocationFromPosition(x.Name, declPosition)
	}

	return nil
}

func (rh *definitionHandler) reply() (interface{}, error) {
	return rh.result, nil
}
