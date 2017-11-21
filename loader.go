package langd

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/object88/langd/collections"
)

// Directory is the collection of packages in a directory
type Directory struct {
	absPath  string
	packages map[string]*Package
}

type importDirective struct {
	absPath string
	// sourcePackage     *Package
	// targetPackageName string
}

// Loader is a Go code loader
type Loader struct {
	ready chan bool

	conf    *types.Config
	context *build.Context
	fset    *token.FileSet
	info    *types.Info

	directories map[string]*Directory
	importGraph *collections.Caravan
}

// Package is the contents of a package
type Package struct {
	// absPath is the cardinal location of this package
	absPath string

	// files contains the map from files names to parsed ast Files
	files map[string]*ast.File

	// name is the package name
	name string

	// astPkg is the native ast Package
	typesPkg *types.Package

	// checked indicates whether this package has been checked
	checked bool
}

// Key returns the collections.Key for the package, indexing it in a caravan
func (p *Package) Key() collections.Key {
	return collections.Key(fmt.Sprintf("%s:%s", p.absPath, p.name))
}

type packageMap map[string]*packageMapItem

type packageMapItem struct {
	files   map[string]*ast.File
	imports map[string]bool
	p       *Package
}

var cgoRe = regexp.MustCompile(`[/\\:]`)

// NewLoader creates a new loader
func NewLoader() *Loader {
	ctx := build.Default
	fmt.Printf("&build.Default = %p\n", &build.Default)
	fmt.Printf("&ctx = %p\n", &ctx)

	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	l := &Loader{
		// conf:        &types.Config{Importer: i},
		context:     &ctx,
		directories: map[string]*Directory{},
		fset:        token.NewFileSet(),
		importGraph: collections.CreateCaravan(),
		info:        info,
		ready:       make(chan bool),
	}
	i := &Importer{
		l: l,
	}
	c := &types.Config{
		Error: func(e error) {
			fmt.Printf("ERROR: %s\n", e.Error())
		},
		Importer: i,
	}
	l.conf = c
	return l
}

// Start initializes the asynchronous source processing
func (l *Loader) Start() chan bool {
	return l.ready
}

// LoadDirectory adds the contents of a directory to the Loader
func (l *Loader) LoadDirectory(absPath string) {
	filepath.Walk(absPath, func(dpath string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			return nil
		}

		// Skipping directories that start with "." (i.e., .git)
		if strings.HasPrefix(filepath.Base(info.Name()), ".") {
			return filepath.SkipDir
		}

		l.processDirectory(&importDirective{absPath: dpath}, 0)

		return nil
	})

	fmt.Printf("\nListing descendents\n")
	for k, v := range l.directories {
		for k1 := range v.packages {
			key := fmt.Sprintf("%s:%s", k, k1)
			fmt.Printf("* %s\n", key)
			n, ok := l.importGraph.Find(collections.Key(key))
			if !ok {
				fmt.Printf("Cannot find %s\n", key)
				continue
			}
			for k2 := range n.Descendants {
				fmt.Printf("\t --> %s\n", k2)
			}
		}
	}
	// l.importGraph.Walk(collections.WalkUp, func(n *collections.Node) {
	// 	p, _ := n.Element.(*Package)

	// 	fmt.Printf("%s:%s\n", p.absPath, p.name)
	// 	if len(n.Descendants) != 0 {
	// 		for k := range n.Descendants {
	// 			fmt.Printf("--> %s\n", k)
	// 		}
	// 	} else {
	// 		fmt.Printf("LEAF\n")
	// 	}
	// 	// if len(n.Ascendants) != 0 {
	// 	// 	for k := range n.Ascendants {
	// 	// 		fmt.Printf("IMP BY %s\n", k)
	// 	// 	}
	// 	// } else {
	// 	// 	fmt.Printf("ROOT\n")
	// 	// }
	// })
	fmt.Printf("Done.\n\n")

	l.importGraph.Walk(collections.WalkUp, func(n *collections.Node) {
		p, ok := n.Element.(*Package)
		if !ok {
			fmt.Printf("NOT PACKAGE.")
			return
		}

		if p.checked == true {
			fmt.Printf("!!! PACKAGE %s ALREADY CHECKED !!!\n", p.Key())
		}

		if strings.HasSuffix(p.absPath, "/src/unsafe") {
			fmt.Printf("Checking %s... using types.Unsafe... Done.\n", p.absPath)
			p.typesPkg = types.Unsafe
		} else {
			files := make([]*ast.File, len(p.files))
			i := 0
			for _, v := range p.files {
				f := v
				files[i] = f
				i++
			}

			fmt.Printf("Checking %s, %d files... ", p.absPath, len(files))

			for _, n1 := range n.Descendants {
				p1, _ := n1.Element.(*Package)
				if !p1.checked {
					fmt.Printf("\n\t!!! ERROR: imported package %s is not checked !!!\n", p1.name)
				}
			}

			typesPkg, err := l.conf.Check(p.absPath, l.fset, files, l.info)
			if err != nil {
				fmt.Printf("\n\t%s\n", err.Error())
			}
			fmt.Printf("Done.\n")
			p.typesPkg = typesPkg
		}
		p.checked = true
	})

	fmt.Printf("Done.\n\n")

	fmt.Printf("Checking for unchecked packages...\n")

	for keyer := range l.importGraph.Iter() {
		p, _ := keyer.(*Package)
		if !p.checked {
			fmt.Printf("!!! PACKAGE %s IS NEVER CHECKED !!!\n", p.Key())
		}
	}

	fmt.Printf("Done.\n\n")

	l.ready <- true
}

func (l *Loader) processDirectory(impDir *importDirective, depth int) *Directory {
	absPath := impDir.absPath

	if d, ok := l.directories[absPath]; ok {
		// Already processed.
		return d
	}

	if strings.HasSuffix(absPath, "/unsafe") {
		fmt.Printf("*** Loading `unsafe`, replacing with types.Unsafe\n")
		p := &Package{
			absPath:  absPath,
			name:     "unsafe",
			typesPkg: types.Unsafe,
		}
		d := &Directory{
			absPath: absPath,
			packages: map[string]*Package{
				"unsafe": p,
			},
		}
		l.directories[absPath] = d
		l.importGraph.Insert(p)
		return d
	}

	buildPkg, err := l.context.Import(".", absPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			return nil
		}
		fmt.Printf("Proc error for '%s':\n\t%s\n", absPath, err.Error())
		return nil
	}

	d := &Directory{
		absPath:  absPath,
		packages: map[string]*Package{},
	}
	l.directories[absPath] = d

	pm := packageMap{}

	for _, v := range buildPkg.GoFiles {
		f := v
		l.processFile(d, f, pm)
	}

	for _, v := range buildPkg.TestGoFiles {
		f := v
		l.processFile(d, f, pm)
	}

	for _, v := range buildPkg.XTestGoFiles {
		f := v
		l.processFile(d, f, pm)
	}

	if len(buildPkg.CgoFiles) != 0 {
		l.processCgoFile(d, buildPkg.CgoFiles, pm, buildPkg.Goroot, buildPkg.ImportPath)
	}

	for pkgName, pmi := range pm {
		files := pmi.files
		thisPkgName := filepath.Base(pkgName)
		p := &Package{
			absPath: absPath,
			files:   files,
			name:    thisPkgName,
		}
		pmi.p = p
		d.packages[thisPkgName] = p
		l.importGraph.Insert(p)
	}

	for _, pmi := range pm {
		l.reportPath(absPath, pmi.p.name, len(pmi.p.files), depth, false)

		// Find... and import.
		for importPkgName := range pmi.imports {
			importPath, err := l.findImportPath(importPkgName, absPath)
			if err != nil {
				fmt.Printf(err.Error())
				continue
			}

			targetPkgName := filepath.Base(importPkgName)
			targetD, ok := l.directories[importPath]
			if !ok {
				targetD = l.processDirectory(&importDirective{absPath: importPath}, depth+1)
			} else {
				l.reportPath(targetD.absPath, targetPkgName, len(targetD.packages[targetPkgName].files), depth, true)
			}

			targetP, ok := targetD.packages[targetPkgName]
			if !ok {
				fmt.Printf("ERROR: Failed to find package %s in %s\n", targetPkgName, d.absPath)
			}
			err = l.importGraph.Connect(pmi.p, targetP)
			if err != nil {
				fmt.Printf("ERROR: %s\n\tfrom: %s\n\tto: %s\n", err.Error(), pmi.p.name, targetP.name)
			}
		}
	}

	return d
}

func (l *Loader) processFile(d *Directory, fname string, pm packageMap) {
	fpath := filepath.Join(d.absPath, fname)
	astf, err := parser.ParseFile(l.fset, fpath, nil, parser.AllErrors)
	if err != nil {
		fmt.Printf("ERROR: While parsing %s:\n\t%s", fpath, err.Error())
		return
	}

	l.processAstFile(fname, astf, pm)
}

func (l *Loader) processCgoFile(d *Directory, fnames []string, pm packageMap, isFromRoot bool, importPath string) {
	fpaths := make([]string, len(fnames))
	for k, v := range fnames {
		fmt.Printf("cgo: %s\n", v)
		fpaths[k] = filepath.Join(d.absPath, v)
	}

	tmpdir, _ := ioutil.TempDir("", strings.Replace(d.absPath, "/", "_", -1)+"_C")
	var files, displayFiles []string

	// _cgo_gotypes.go (displayed "C") contains the type definitions.
	files = append(files, filepath.Join(tmpdir, "_cgo_gotypes.go"))
	displayFiles = append(displayFiles, "C")
	for _, fname := range fnames {
		// "foo.cgo1.go" (displayed "foo.go") is the processed Go source.
		f := cgoRe.ReplaceAllString(fname[:len(fname)-len("go")], "_")
		files = append(files, filepath.Join(tmpdir, f+"cgo1.go"))
		displayFiles = append(displayFiles, fname)
	}

	fmt.Printf("importPath = %s\n", importPath)
	var cgoflags []string
	if isFromRoot && importPath == "runtime/cgo" {
		cgoflags = append(cgoflags, "-import_runtime_cgo=false")
	}
	if isFromRoot && importPath == "runtime/race" || importPath == "runtime/cgo" {
		cgoflags = append(cgoflags, "-import_syscall=false")
	}

	args := []string{
		"tool",
		"cgo",
		"-objdir",
		tmpdir,
	}
	for _, f := range cgoflags {
		args = append(args, f)
	}
	args = append(args, "--")
	args = append(args, "-I")
	args = append(args, tmpdir)
	for _, f := range fnames {
		args = append(args, f)
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = d.absPath
	// cmd.Stdout = os.Stderr
	// cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Printf("ERROR: cgo failed: %s: %s", args, err)
		return
	}

	fmt.Printf("Processing %d cgo based files\n", len(fpaths))
	for i := range fpaths {
		f, err := os.Open(files[i])
		if err != nil {
			fmt.Printf("ERROR: failed to open file %s\n\t%s\n", files[i], err.Error())
			continue
		}
		astf, err := parser.ParseFile(l.fset, displayFiles[i], f, 0)
		f.Close()
		if err != nil {
			fmt.Printf("ERROR: Failed to open %s\n\t%s\n", fpaths[i], err.Error())
			return
		}

		l.processAstFile(fnames[i], astf, pm)
	}
}

func (l *Loader) processAstFile(fname string, astf *ast.File, pm packageMap) {
	pkgName := astf.Name.Name

	pmi, ok := pm[pkgName]
	if !ok {
		pmi = &packageMapItem{
			files:   map[string]*ast.File{},
			imports: map[string]bool{},
		}
		pm[pkgName] = pmi
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
			pmi.imports[path] = true
		}
	}

	pmi.files[fname] = astf
}

func (l *Loader) findImportPath(path, src string) (string, error) {
	buildPkg, err := l.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Oh dear:\n\tAttemped build.Import('%s', '%s', build.FindOnly)\n\t%s\n", path, src, err.Error())
		fmt.Printf("ERROR: %s", msg)
		return "", errors.New(msg)
	}
	return buildPkg.Dir, nil
}

func (l *Loader) reportPath(absPath, targetPackageName string, fileCount, depth int, pathAlreadyFound bool) {
	packagePath := absPath
	for _, v := range l.context.SrcDirs() {
		if strings.HasPrefix(packagePath, v) {
			packagePath = packagePath[len(v)+1:]
			break
		}
	}
	prefix := ""
	if pathAlreadyFound {
		prefix = "☑ "
	}
	spacer := strings.Repeat("  ", depth)
	fmt.Printf("%s%s%s:%s (%d)\n", prefix, spacer, packagePath, targetPackageName, fileCount)
}
