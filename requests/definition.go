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

// definition implements the `Goto Definition` request
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#goto-definition-request
func (h *Handler) definition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.log.Verbosef("Got definition method\n")

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

	var params TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	h.log.Verbosef("Got parameters: %#v\n", params)

	path := strings.TrimPrefix(string(params.TextDocument.URI), "file://")

	p := token.Position{
		Filename: path,
		Line:     params.Position.Line + 1,
		Column:   params.Position.Character,
	}

	h.log.Verbosef("Searching for token at %#v\n", p)

	f := h.workspace.Files[p.Filename]
	if f == nil {
		// Failure response is failure.
		h.log.Errorf("File %s isn't in our workspace\n", p.Filename)
		return
	}

	found := false

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		pStart := h.workspace.Fset.Position(n.Pos())
		pEnd := h.workspace.Fset.Position(n.End())

		if withinPos(p, pStart, pEnd) {
			h.log.Verbosef("** FOUND IT **: [%d:%d]-[%d:%d] %#v\n", pStart.Line, pStart.Column, pEnd.Line, pEnd.Column, n)
			switch v := n.(type) {
			case *ast.Ident:
				if v.Obj == nil {
					// We have an identifier, but it has no object reference.
					// This may happen because the user has an incomplete program.
					return false
				}
				h.log.Verbosef("Have identifier: %s, object %#v\n", v.String(), v.Obj.Decl)
				switch v1 := v.Obj.Decl.(type) {
				case *ast.Field:
					declPosition := h.workspace.Fset.Position(v1.Pos())
					h.log.Verbosef("Have field; declaration at %s\n", declPosition.String())

					found = true
					result := &Location{
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

					conn.Reply(ctx, req.ID, result)

				case *ast.TypeSpec:
					declPosition := h.workspace.Fset.Position(v1.Pos())
					h.log.Verbosef("Have typespec; declaration at %s\n", declPosition.String())

					found = true
					result := &Location{
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

					conn.Reply(ctx, req.ID, result)

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

	// Didn't find what we were looking for.
	if !found {
		conn.Reply(ctx, req.ID, nil)
	}
}

func withinPos(pTarget, pStart, pEnd token.Position) bool {
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
