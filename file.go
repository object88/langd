package langd

import (
	"go/ast"
	"go/token"
	"io"

	"github.com/OneOfOne/xxhash"
	"github.com/object88/langd/collections"
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

func calculateHash(r io.Reader) collections.Hash {
	h := xxhash.New64()
	in := make([]byte, 1024)
	for {
		c, err := r.Read(in)
		if err == io.EOF {
			break
		}
		h.Write(in[0:c])
	}
	hash := h.Sum64()

	return collections.Hash(hash)
}

func calculateHashFromString(s string) collections.Hash {
	h := xxhash.New64()
	h.WriteString(s)
	hash := h.Sum64()

	return collections.Hash(hash)
}
