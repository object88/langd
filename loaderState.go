package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"strings"
)

type loaderState struct {
	org      string
	orgPath  string
	fset     *token.FileSet
	files    map[string]*ast.File
	pkgNames map[string]bool
	info     *types.Info
	spacers  *[]string
}

func newLoaderState(pkgName string) *loaderState {
	spacers := make([]string, 8)
	for i := 0; i < 8; i++ {
		spacers[i] = strings.Repeat("  ", i)
	}

	org := ""
	orgPath := ""
	s := strings.Split(pkgName, "/")
	if strings.Index(s[0], ".") != -1 {
		org = fmt.Sprintf("%s/%s", s[0], s[1])
		orgPath = fmt.Sprintf("%s/src/%s/%s", build.Default.GOPATH, s[0], s[1])
	}

	ls := &loaderState{
		org,
		orgPath,
		token.NewFileSet(),
		map[string]*ast.File{},
		map[string]bool{},
		&types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		},
		&spacers,
	}

	return ls
}

func (ls *loaderState) getSpacer(depth int) string {
	initialSize := len(*ls.spacers)
	if depth >= initialSize {
		// Expand the contents
		targetSize := nextPowerOfTwo(depth + 1)
		target := make([]string, targetSize)
		for i := 0; i < initialSize; i++ {
			target[i] = (*ls.spacers)[i]
		}
		for i := initialSize; i < targetSize; i++ {
			target[i] = strings.Repeat("  ", i)
		}

		ls.spacers = &target
	}

	return (*ls.spacers)[depth]
}
