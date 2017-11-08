package langd

import (
	"go/build"
	"sync"
)

// Directory represents the Go code in a directory
type Directory struct {
	path     string
	buildPkg *build.Package
	pkgs     []*Package
	files    []string
	m        sync.Mutex

	ready chan struct{}
}

// CreateDirectory creates a new Directory struct
// func CreateDirectory(root, path string) *Directory {
func CreateDirectory(path string) *Directory {
	return &Directory{
		path: path,
		pkgs: []*Package{},
	}
}

// // Scan imports the files in a directory and processes the files
// func (d *Directory) Scan(fset *token.FileSet, packs *collections.Caravan, dirQueue chan<- interface{}) {
// 	buildPkg, err := build.ImportDir(d.path, 0)
// 	if err != nil {
// 		if _, ok := err.(*build.NoGoError); ok {
// 			// There isn't any Go code here.
// 			fmt.Printf("NO GO CODE: %s\n", d.path)
// 			return
// 		}
// 		fmt.Printf("Oh dear:\n\t%s\n\t%s\n", d.path, err.Error())
// 	}
// 	d.buildPkg = buildPkg

// 	var wg sync.WaitGroup
// 	imports := importMap{}

// 	count := len(buildPkg.GoFiles) + len(buildPkg.TestGoFiles)
// 	wg.Add(count)

// 	for _, v := range buildPkg.GoFiles {
// 		fpath := filepath.Join(buildPkg.Dir, v)
// 		go d.processFile(imports, fset, packs, fpath, &wg)
// 	}

// 	for _, v := range buildPkg.TestGoFiles {
// 		fpath := filepath.Join(buildPkg.Dir, v)
// 		go d.processFile(imports, fset, packs, fpath, &wg)
// 	}

// 	wg.Wait()

// 	fmt.Printf("Scanned %s\n", d.path)

// 	// Take all the imports, and announce them back for processing
// 	for src, imps := range imports {
// 		for imp := range imps {
// 			absPath := findPackagePath(imp, buildPkg.ImportPath)
// 			dirQueue <- &importDir{
// 				imp:  imp,
// 				path: absPath,
// 				src:  src,
// 			}
// 		}
// 	}
// }

// func (d *Directory) processFile(imports importMap, fset *token.FileSet, packs *collections.Caravan, fpath string, wg *sync.WaitGroup) {
// 	astf, err := parser.ParseFile(fset, fpath, nil, 0)
// 	if err != nil {
// 		fmt.Printf("Got error while parsing file '%s':\n\t%s\n", fpath, err.Error())
// 		// l.ls.errs = append(l.ls.errs, err)
// 		wg.Done()
// 		return
// 	}

// 	absPath := filepath.Dir(fpath)
// 	pkgName := astf.Name.Name

// 	d.m.Lock()

// 	var pkg *Package
// 	for _, v := range d.pkgs {
// 		if v.path == absPath && v.pkgName == pkgName {
// 			pkg = v
// 			break
// 		}
// 	}
// 	if pkg == nil {
// 		pkg = &Package{
// 			astPkg: &ast.Package{
// 				Name:  pkgName,
// 				Files: map[string]*ast.File{},
// 			},
// 			path:    absPath,
// 			pkgName: pkgName,
// 		}
// 		packs.Insert(pkg)
// 		d.pkgs = append(d.pkgs, pkg)
// 	}

// 	src := &importKey{
// 		pkgName: pkgName,
// 		absPath: absPath,
// 	}

// 	for _, decl := range astf.Decls {
// 		decl, ok := decl.(*ast.GenDecl)
// 		if !ok || decl.Tok != token.IMPORT {
// 			continue
// 		}

// 		for _, spec := range decl.Specs {
// 			spec := spec.(*ast.ImportSpec)

// 			// NB: do not assume the program is well-formed!
// 			path, err := strconv.Unquote(spec.Path.Value)
// 			if err != nil || path == "C" {
// 				// Ignore the error and skip the C psuedo package
// 				continue
// 			}
// 			destMap, ok := imports[src]
// 			if !ok {
// 				destMap = map[string]bool{}
// 				imports[src] = destMap
// 			}
// 			destMap[path] = true
// 		}
// 	}

// 	d.m.Unlock()

// 	wg.Done()
// }
