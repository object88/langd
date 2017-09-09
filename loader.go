package langd

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/object88/langd/log"
)

// Loader will load code into an AST
type Loader struct {
	config  *types.Config
	srcDirs []string
	stderr  *log.Log
}

// NewLoader constructs a new Loader struct
func NewLoader() *Loader {
	l := log.Stderr()
	config := &types.Config{
		Error: func(e error) {
			l.Warnf("%s\n", e.Error())
		},
		Importer: importer.Default(),
	}
	srcDirs := build.Default.SrcDirs()
	return &Loader{config, srcDirs, l}
}

// Load reads in the AST
func (l *Loader) Load(ctx context.Context, base string) (*Workspace, error) {
	if l == nil {
		return nil, errors.New("No pointer receiver")
	}

	abs, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("Provided path '%s' must be a directory", base)
	}

	pkgName := ""
	for _, v := range l.srcDirs {
		if strings.HasPrefix(abs, v) {
			pkgName = abs[len(v)+1:]
		}
	}
	if pkgName == "" {
		return nil, fmt.Errorf("Failed to find '%s'", base)
	}

	l.stderr.Verbosef("pkgName: '%s'\n", pkgName)

	// [GOPATH]/src/github.com/object88/pkgA0/pkgA1/pkgA2
	l.stderr.Debugf("Source dirs:\n")
	for _, v := range l.srcDirs {
		l.stderr.Debugf("%s\n", v)
	}

	l.stderr.Debugf("Pkgname: %s\n", pkgName)

	ls := newLoaderState(pkgName)

	err = l.load(ctx, ls, abs, pkgName, 0)
	if err != nil {
		return nil, err
	}

	workspace := newWorkspace(ls.fset, ls.pkgs)

	return workspace, nil
}

func (l *Loader) load(ctx context.Context, ls *loaderState, fpath, base string, depth int) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	pkgs, err := l.buildPackages(ls, fpath, base, depth)
	if err != nil {
		return err
	}

	for _, v := range *pkgs {
		l.stderr.Verbosef("%sProcessing '%s' imports...\n", ls.getSpacer(depth), v.Pkg.Path())

		err = l.visitImports(ctx, v.Pkg, ls, depth)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) buildAstPackages(buildP *build.Package, ls *loaderState) map[string]*ast.Package {
	astPkgs := map[string]*ast.Package{}

	for _, v := range buildP.GoFiles {
		fpath := path.Join(buildP.Dir, v)

		astf, err := parser.ParseFile(ls.fset, fpath, nil, 0)
		if err != nil {
			l.stderr.Verbosef("Got error while parsing file '%s':\n%s\n", fpath, err.Error())
		}

		name := astf.Name.Name
		astPkg, found := astPkgs[name]
		if !found {
			astPkg = &ast.Package{
				Name:  name,
				Files: make(map[string]*ast.File),
			}
			astPkgs[name] = astPkg
		}
		astPkg.Files[fpath] = astf
	}

	return astPkgs
}

func (l *Loader) buildPackages(ls *loaderState, fpath, base string, depth int) (*[]*Package, error) {
	buildP, err := build.ImportDir(fpath, 0)
	if err != nil {
		return nil, err
	}

	pkgs := []*Package{}

	astPackages := l.buildAstPackages(buildP, ls)

	for k, v := range astPackages {
		pkg := newPkg(cleanPath(base))
		pkg.AstPkg = v

		files := getFileFlatlist(v)

		p, err := l.config.Check(k, ls.fset, *files, pkg.Info)
		if err != nil {
			l.stderr.Verbosef("Got error checking package '%s':\n%s\n", k, err.Error())
		}

		path, err := l.findSourcePath(base)
		if err != nil {
			return nil, err
		}
		l.stderr.Verbosef("%s-- Adding key '%s' / '%s'\n", ls.getSpacer(depth), base, path)

		pkg.Pkg = p
		ls.pkgs[path] = pkg
	}

	// pkg, ok := ls.pkgs[fpath]
	// if !ok {
	// 	return nil, errors.New("Failed to find package with directory-eponymous name")
	// }

	return &pkgs, nil
}

func (l *Loader) visitImports(ctx context.Context, p *types.Package, ls *loaderState, depth int) error {
	imports := p.Imports()
	for _, v0 := range imports {
		id := v0.Path()
		path, err := l.findSourcePath(id)
		if err != nil {
			return err
		}

		if _, ok := ls.pkgs[path]; ok {
			l.stderr.Verbosef("%s** Checking for '%s' / '%s' already parsed; skipping...\n", ls.getSpacer(depth), id, path)
			continue
		}

		l.stderr.Verbosef("%s** Checking for '%s' / '%s' processing...\n", ls.getSpacer(depth), id, path)
		err = l.load(ctx, ls, path, id, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) findSourcePath(pkgName string) (string, error) {
	if pkgName == "." {
		p, err := os.Getwd()
		if err != nil {
			return "", err
		}
		l.stderr.Verbosef("Got '.'; using '%s'\n", p)

		return p, nil
	}

	for _, v := range l.srcDirs {
		fpath := path.Join(v, pkgName)
		isDir := false
		if build.Default.IsDir != nil {
			isDir = build.Default.IsDir(fpath)
		} else {
			s, err := os.Stat(fpath)
			if err != nil {
				continue
			}
			isDir = s.IsDir()
		}
		if isDir {
			return fpath, nil
		}
	}

	return "", fmt.Errorf("Failed to locate package '%s'", pkgName)
}

func cleanPath(path string) string {
	idx := strings.LastIndex(path, "vendor")
	if idx != -1 {
		path = path[idx+len("vendor")+1:]
	}
	return path
}

func getFileFlatlist(pkg *ast.Package) *[]*ast.File {
	asts := make([]*ast.File, len(pkg.Files))
	i := 0
	for _, f := range pkg.Files {
		asts[i] = f
		i++
	}

	return &asts
}
