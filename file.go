package langd

import (
	"go/ast"
	"go/token"
)

// File is an AST file and any errors that types.Config.Check discovers
type File struct {
	file *ast.File
	errs []FileError
}

// FileError is a translation of the types.Error struct
type FileError struct {
	token.Position
	Message string
	Warning bool
}
