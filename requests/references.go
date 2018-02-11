package requests

import (
	"context"
	"encoding/json"
	"fmt"
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

	p       *token.Position
	options *ReferenceContext
	result  *[]Location
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
	// 		Position: requests.Position {
	// 			Line: 11,
	// 			Character: 14
	// 		}
	// 	},
	// 	Context: requests.ReferenceContext {
	//		IncludeDeclaration: true
	// 	}
	// }
	var typedParams ReferenceParams
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

	rh.options = typedParams.Context
	rh.p = p
	return nil
}

func (rh *referencesHandler) work() error {
	x, err := rh.h.workspace.LocateIdent(rh.p)
	if err != nil {
		return err
	}

	if x == nil {
		fmt.Printf("No identifier located; nothing we can do.\n	")
		return nil
	}

	declPosition, err := rh.h.workspace.LocateDeclaration(rh.p)
	if err != nil {
		fmt.Printf("Failed to find declaration position: %s\n", err.Error())
	}
	// declIdent, err := rh.h.workspace.LocateIdent(declPosition)
	// if err != nil {
	// 	fmt.Printf("Failed to find ident: %s\n", err.Error())
	// }

	usePositions := rh.h.workspace.LocateReferences(rh.p)

	size := 0
	if usePositions != nil && len(usePositions) > 0 {
		size = len(usePositions)
	}

	if rh.options.IncludeDeclaration {
		if declPosition != nil {
			size++
		}
	} else {
		fmt.Printf("No include declaration\n")
	}

	if size == 0 {
		fmt.Printf("No locations found\n")
		return nil
	}

	locs := make([]Location, size)
	offset := 0
	if declPosition != nil {
		locs[0] = *LocationFromPosition(x.Name, declPosition)
		offset = 1
	}
	if usePositions != nil {
		for k, v := range usePositions {
			fmt.Printf("\t%d: %v\n", k, v)
			locs[k+offset] = *LocationFromPosition(x.Name, &v)
		}
	}

	// fmt.Printf("X:\n%#v\n", x.Obj)
	// foo := rh.h.workspace.Info.Uses[x]
	// fmt.Printf("Uses:\n%#v\n", foo)

	rh.result = &locs

	return nil
}

func (rh *referencesHandler) reply() (interface{}, error) {
	return rh.result, nil
}
