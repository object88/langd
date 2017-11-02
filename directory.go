package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"sync"
)

// Directory represents the Go code in a directory
type Directory struct {
	path     string
	buildPkg *build.Package
	pkgs     []*Package
	files    []string
	m        sync.Mutex
}

// CreateDirectory creates a new Directory struct
// func CreateDirectory(root, path string) *Directory {
func CreateDirectory(path string) *Directory {
	return &Directory{
		path: path,
		pkgs: []*Package{},
	}
}

// Scan imports the files in a directory and processes the files
func (d *Directory) Scan(fset *token.FileSet, dirQueue chan<- interface{}) {
	// buildPkg, err := build.Import(d.root, d.path, 0)
	buildPkg, err := build.ImportDir(d.path, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			fmt.Printf("NO GO CODE: %s\n", d.path)
			return
		}
		// fmt.Printf("Oh dear:\n\t%s\n\t%s\n\t%s\n", d.root, d.path, err.Error())
		fmt.Printf("Oh dear:\n\t%s\n\t%s\n", d.path, err.Error())
	}
	d.buildPkg = buildPkg

	var wg sync.WaitGroup
	imports := map[string]bool{}

	count := len(buildPkg.GoFiles) + len(buildPkg.TestGoFiles)
	wg.Add(count)

	for _, v := range buildPkg.GoFiles {
		fpath := filepath.Join(buildPkg.Dir, v)
		go d.processFile(imports, fset, fpath, &wg)
	}

	for _, v := range buildPkg.TestGoFiles {
		fpath := filepath.Join(buildPkg.Dir, v)
		go d.processFile(imports, fset, fpath, &wg)
	}

	wg.Wait()

	// Take all the imports, and announce them back for processing
	for dpath := range imports {
		absPath := findPackagePath(dpath, buildPkg.ImportPath)
		dirQueue <- &importDir{
			path: absPath,
		}
	}
}

func (d *Directory) processFile(imports map[string]bool, fset *token.FileSet, fpath string, wg *sync.WaitGroup) {
	astf, err := parser.ParseFile(fset, fpath, nil, 0)
	if err != nil {
		fmt.Printf("Got error while parsing file '%s':\n\t%s\n", fpath, err.Error())
		// l.ls.errs = append(l.ls.errs, err)
		return
	}

	pkgName := astf.Name.Name
	key := buildKey(pkgName, fpath)

	d.m.Lock()

	var pkg *Package
	for _, v := range d.pkgs {
		if v.Key() == key {
			pkg = v
			break
		}
	}
	if pkg == nil {
		pkg = &Package{
			astPkg: &ast.Package{
				Name:  pkgName,
				Files: map[string]*ast.File{},
			},
			path:    fpath,
			pkgName: pkgName,
		}
		d.pkgs = append(d.pkgs, pkg)
	}

	for _, decl := range astf.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.IMPORT {
			continue
		}

		for _, spec := range decl.Specs {
			spec := spec.(*ast.ImportSpec)

			// NB: do not assume the program is well-formed!
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path == "C" {
				// Ignore the error and skip the C psuedo package
				continue
			}
			imports[path] = true
		}
	}

	d.m.Unlock()

	wg.Done()
}
