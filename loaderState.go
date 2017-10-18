package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"strings"

	"github.com/object88/langd/collections"
)

type loaderState struct {
	org     string
	orgPath string
	fset    *token.FileSet
	info    *types.Info

	// filesm         sync.Mutex
	// loadedPackages map[string]*ast.Package
	// loadedPaths    map[string]bool

	// m sync.Mutex
	// map of file paths to astFile
	// is this necessary?
	files map[string]*ast.File
	// loadedPkgs map[string]bool
	fileQueue *packageQueue
	errs      []error
	packs     *collections.Caravan
}

func newLoaderState(pkgName string) *loaderState {
	org := ""
	orgPath := ""
	s := strings.Split(pkgName, "/")
	if strings.Index(s[0], ".") != -1 {
		org = fmt.Sprintf("%s/%s", s[0], s[1])
		orgPath = fmt.Sprintf("%s/src/%s/%s", build.Default.GOPATH, s[0], s[1])
	}

	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	ls := &loaderState{
		org:     org,
		orgPath: orgPath,
		fset:    token.NewFileSet(),
		info:    info,

		files: map[string]*ast.File{},
		// loadedPackages: map[string]*ast.Package{},
		// loadedPaths:    map[string]bool{},

		// loadedPkgs: map[string]bool{},
		fileQueue: createPackageQueue(),
		errs:      []error{},
		packs:     collections.CreateCaravan(),
	}

	return ls
}
