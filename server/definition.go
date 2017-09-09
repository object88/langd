package server

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/object88/langd"
	"github.com/sourcegraph/jsonrpc2"
)

const (
	definitionMethod = "textDocument/definition"
)

// definition implements the `Goto Definition` request
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#goto-definition-request
func (h *Handler) definition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	fmt.Printf("Got definition method\n")

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

	fmt.Printf("Got parameters: %#v\n", params)

	path := strings.TrimPrefix(string(params.TextDocument.URI), "file://")

	p := token.Position{
		Filename: path,
		Line:     params.Position.Line + 1,
		Column:   params.Position.Character,
	}

	fmt.Printf("Searching for token at %#v\n", p)

	pkg, f := h.locatePosition(p)
	if f == nil {
		// Failure response is failure.
		return
	}

	fmt.Printf("Have package %s\n", pkg.Name)

	fmt.Printf("Looking for offset %d\n", p.Offset)

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		pStart := h.workspace.Fset.Position(n.Pos())
		pEnd := h.workspace.Fset.Position(n.End())

		// fmt.Printf("Start: %s; end: %s\n", pStart.String(), pEnd.String())
		if withinPos(p, pStart, pEnd) {
			fmt.Printf("** FOUND IT **: [%d:%d]-[%d:%d] %#v\n", pStart.Line, pStart.Column, pEnd.Line, pEnd.Column, n)
			switch v := n.(type) {
			case *ast.Ident:
				fmt.Printf("Have identifier: %s, object %#v\n", v.String(), v.Obj.Decl)
				switch v1 := v.Obj.Decl.(type) {
				case *ast.TypeSpec:
					declPosition := h.workspace.Fset.Position(v1.Pos())
					fmt.Printf("Have declaration at %s\n", declPosition.String())

					result := &Location{
						URI: DocumentURI(fmt.Sprintf("file://%s", declPosition.Filename)),
						Range: Range{
							Start: Position{
								Line:      declPosition.Line - 1,
								Character: declPosition.Column - 1,
							},
							End: Position{
								Line:      declPosition.Line - 1,
								Character: declPosition.Column + len(v1.Name.Name) - 1,
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

func (h *Handler) locatePosition(p token.Position) (*langd.Package, *ast.File) {
	for _, v := range h.workspace.Pkgs {
		for n, f := range v.AstPkg.Files {
			if n == p.Filename {
				fmt.Printf("Found %s\n", n)
				return v, f
			}
		}
	}
	return nil, nil
}
