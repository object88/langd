package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/object88/langd/collections"
)

type importDir struct {
	buildPkg *build.Package
	imp      string
	path     string
	src      *importKey
}

type importKey struct {
	pkgName string
	absPath string
}

func (ik *importKey) Key() collections.Key {
	k := fmt.Sprintf("%s:%s", ik.pkgName, ik.absPath)
	return collections.Key(k)
}

// importMap tracks the imports of a directory
type importMap map[*importKey]map[string]bool

type Loader struct {
	srcDirs     []string
	fset        *token.FileSet
	importGraph *collections.Caravan
	dirs        map[string]*dir
}

type dir struct {
	absPath string
	loaded  bool
	m       sync.Mutex
}

func NewLoader() *Loader {
	return &Loader{
		dirs:        map[string]*dir{},
		fset:        token.NewFileSet(),
		importGraph: collections.CreateCaravan(),
		srcDirs:     build.Default.SrcDirs(),
	}
}

func (l *Loader) LoadDirectories(dpaths ...string) {
	for _, dpath := range dpaths {
		l.loadDirectory(dpath)
	}
}

func (l *Loader) Start() (chan bool, error) {
	return nil, nil
}

func (l *Loader) Close() {

}

func (l *Loader) loadDirectory(dpath string) error {
	fmt.Printf("loadDirectory :: %s\n", dpath)

	prefix := ""
	for _, v := range l.srcDirs {
		if strings.HasPrefix(dpath, v) {
			prefix = dpath[len(v)+1:]
		}
	}
	if prefix == "" {
		return fmt.Errorf("%s is not in gopath or goroot", dpath)
	}

	filepath.Walk(dpath, func(dpath string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			return nil
		}

		// Skipping directories that start with "." (i.e., .git)
		if strings.HasPrefix(filepath.Base(info.Name()), ".") {
			return filepath.SkipDir
		}

		fmt.Printf("Processing %s...\n", dpath)

		// dpath: /Users/bropa18/work/src/github.com/object88/langd/examples/echo
		importDir := &importDir{
			path: dpath,
		}
		l.processDirectory(importDir)

		time.Sleep(5 * time.Second)

		return nil
	})

	return nil
}

func (l *Loader) processDirectory(imp *importDir) {
	fmt.Printf("Processing directory %s\n", imp.path)
	d := l.ensureDir(".", imp.path)
	if d.loaded {
		return
	}

	buildPkg, err := build.ImportDir(d.absPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			fmt.Printf("NO GO CODE: %s\n", d.absPath)
			return
		}
		fmt.Printf("Oh dear:\n\t%s\n\t%s\n", d.absPath, err.Error())
		return
	}

	d.loaded = true

	imports := importMap{}

	// count := len(buildPkg.GoFiles) + len(buildPkg.TestGoFiles)

	for _, v := range buildPkg.GoFiles {
		fpath := filepath.Join(buildPkg.Dir, v)
		l.processFile(buildPkg, fpath, imports)
	}

	for _, v := range buildPkg.TestGoFiles {
		fpath := filepath.Join(buildPkg.Dir, v)
		l.processFile(buildPkg, fpath, imports)
	}

	fmt.Printf("Have %d imports\n", len(imports))

	// Take all the imports, and announce them back for processing
	for src, imps := range imports {
		fmt.Printf("%s...\n", src.Key())
		for imp := range imps {
			fmt.Printf("\t%s\n", imp)
			impPkg := l.ensure(buildPkg, imp)
			importDir := &importDir{
				imp:  impPkg.path,
				path: d.absPath,
				src:  src,
			}
			l.processDirectory(importDir)
		}
	}
}

func (l *Loader) processFile(buildPkg *build.Package, fpath string, imports importMap) {
	fmt.Printf("Processing file %s\n", fpath)
	dpath := filepath.Dir(fpath)
	astf, _ := parser.ParseFile(l.fset, fpath, nil, 0)

	pkgName := astf.Name.Name
	pkg := l.ensure(buildPkg, pkgName)
	pkg.astPkg.Files[fpath] = astf

	src := &importKey{
		pkgName: pkgName,
		absPath: dpath,
	}

	for _, decl := range astf.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.IMPORT {
			continue
		}

		for _, spec := range decl.Specs {
			spec := spec.(*ast.ImportSpec)

			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path == "C" {
				// Ignore the error and skip the C pseudo package
				continue
			}
			destMap, ok := imports[src]
			if !ok {
				destMap = map[string]bool{}
				imports[src] = destMap
			}
			destMap[path] = true
		}
	}
}

func (l *Loader) ensureDir(path, src string) *dir {
	fmt.Printf("Ensuring dir %s, %s\n", path, src)
	buildPkg, err := build.Import(path, src, build.FindOnly)
	if err != nil {
		fmt.Printf("Oh dir dear:\n\t%s\n", err.Error())
		return nil
	}

	absPath := buildPkg.Dir
	d, ok := l.dirs[absPath]
	if !ok {
		d = &dir{
			absPath: absPath,
		}
		l.dirs[absPath] = d
	}
	return d
}

func (l *Loader) ensure(buildPkg *build.Package, pkgName string) *Package {
	fmt.Printf("Ensuring %s\n", pkgName)
	// if path == "main" {
	// 	fmt.Printf("Convered to '.'\n")
	// 	path = "."
	// }
	pkgPath := buildPkg.Dir
	if pkgPath == "" {
		// If pkgPath is the empty string, this is a stdlib package?
		fmt.Printf("Ensured %s; No Dir; using %s\n", pkgName, pkgName)
		pkgPath = pkgName
	} else {
		fmt.Printf("Ensured %s; got Dir %s\n", pkgName, pkgPath)
	}

	var pkg *Package
	k := buildKey(pkgName, pkgPath)
	keyer, ok := l.importGraph.Find(k)
	if !ok {
		pkg = &Package{
			astPkg: &ast.Package{
				Name:  pkgName,
				Files: map[string]*ast.File{},
			},
			checked: false,
			path:    pkgPath,
			pkgName: pkgName,
		}

		l.importGraph.Insert(pkg)
	} else {
		pkg = keyer.(*Package)
	}

	return pkg
}
