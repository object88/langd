package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
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
		requestBase: createRequestBase(ctx, h, req.ID),
	}

	return rh
}

func (rh *definitionHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got definition method\n")

	// Example:
	// server.TextDocumentPositionParams {
	//   TextDocument: server.TextDocumentIdentifier {
	// 		 URI: "file:///Users/bropa18/work/src/github.com/object88/immutable/memory/types.go",
	//   },
	// 	 Position: server.Position {
	//     Line: 7,
	//     Character: 15,
	//   }
	// }

	var typedParams TextDocumentPositionParams
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

func (rh *definitionHandler) work() error {
	f := rh.h.workspace.Files[rh.p.Filename]
	if f == nil {
		// Failure response is failure.
		return fmt.Errorf("File %s isn't in our workspace\n", rh.p.Filename)
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		pStart := rh.h.workspace.Fset.Position(n.Pos())
		pEnd := rh.h.workspace.Fset.Position(n.End())

		if withinPos(rh.p, &pStart, &pEnd) {
			rh.h.log.Verbosef("** FOUND IT **: [%d:%d]-[%d:%d] %#v\n", pStart.Line, pStart.Column, pEnd.Line, pEnd.Column, n)
			switch v := n.(type) {
			case *ast.Ident:
				if v.Obj == nil {
					// We have an identifier, but it has no object reference.
					// This may happen because the user has an incomplete program.
					return false
				}
				rh.h.log.Verbosef("Have identifier: %s, object %#v\n", v.String(), v.Obj.Decl)
				switch v1 := v.Obj.Decl.(type) {
				case *ast.Field:
					declPosition := rh.h.workspace.Fset.Position(v1.Pos())
					rh.h.log.Verbosef("Have field; declaration at %s\n", declPosition.String())

					rh.result = &Location{
						URI: DocumentURI(fmt.Sprintf("file://%s", declPosition.Filename)),
						Range: Range{
							Start: Position{
								Line:      declPosition.Line - 1,
								Character: declPosition.Column - 1,
							},
							End: Position{
								Line:      declPosition.Line - 1,
								Character: declPosition.Column + len(v.Name) - 1,
							},
						},
					}

				case *ast.TypeSpec:
					declPosition := rh.h.workspace.Fset.Position(v1.Pos())
					rh.h.log.Verbosef("Have typespec; declaration at %s\n", declPosition.String())

					rh.result = &Location{
						URI: DocumentURI(fmt.Sprintf("file://%s", declPosition.Filename)),
						Range: Range{
							Start: Position{
								Line:      declPosition.Line - 1,
								Character: declPosition.Column - 1,
							},
							End: Position{
								Line:      declPosition.Line - 1,
								Character: declPosition.Column + len(v.Name) - 1,
							},
						},
					}

				default:
					// No-op
				}
			default:
				// No-op
			}
			return true
		}
		return false
	})

	// // Didn't find what we were looking for.
	// if !found {
	// 	h.Respond(ctx, id, nil)
	// }
	return nil
}

func (rh *definitionHandler) reply() (interface{}, error) {
	return rh.result, nil
}

func withinPos(pTarget, pStart, pEnd *token.Position) bool {
	if pTarget.Line < pStart.Line || pTarget.Line > pEnd.Line {
		return false
	}

	if pTarget.Line == pStart.Line && pTarget.Column < pStart.Column {
		return false
	}

	if pTarget.Line == pEnd.Line && pTarget.Column >= pEnd.Column {
		return false
	}

	return true
}
