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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/object88/langd/collections"
)

type loadState int

const (
	queued loadState = iota
	unloaded
	loadedGo
	loadedTest
	done
)

// Loader is a Go code loader
type Loader struct {
	mReady sync.Mutex
	closer chan bool
	ready  chan bool

	done bool

	caravan *collections.Caravan

	mDirectories sync.Mutex
	directories  map[string]*Directory

	stateChange chan string

	conf    *types.Config
	context *build.Context
	fset    *token.FileSet
	info    *types.Info

	unsafePath string
}

// Directory is the collection of packages in a directory
type Directory struct {
	buildPkg *build.Package

	absPath string

	pm       packageMap
	packages map[string]*Package

	loadState loadState
	m         sync.Mutex
	c         *sync.Cond
}

// PackageKey contains the canonical absolute path and package name
type PackageKey struct {
	absPath string
	name    string
}

// Package is the contents of a package
type Package struct {
	PackageKey
	files       map[string]*ast.File
	importPaths map[string]bool
	key         collections.Key
	typesPkg    *types.Package
}

// CreatePackage creates a new Package with the given canonical path and name
func CreatePackage(absPath, name string) *Package {
	return &Package{
		PackageKey: PackageKey{
			absPath: absPath,
			name:    name,
		},
		files:       map[string]*ast.File{},
		importPaths: map[string]bool{},
		key:         collections.Key(fmt.Sprintf("%s:%s", absPath, name)),
	}
}

// Key returns the collections.Key for the given Package
func (p *Package) Key() collections.Key {
	return p.key
}

type packageMap map[string]*packageMapItem

type packageMapItem struct {
	files       map[string]*ast.File
	imports     map[string]bool
	m           sync.Mutex
	p           *Package
	testFiles   map[string]*ast.File
	testImports map[string]bool
}

var cgoRe = regexp.MustCompile(`[/\\:]`)

// NewLoader creates a new loader
func NewLoader() *Loader {
	ctx := build.Default

	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}

	l := &Loader{
		caravan:     collections.CreateCaravan(),
		closer:      make(chan bool),
		context:     &ctx,
		directories: map[string]*Directory{},
		done:        false,
		fset:        token.NewFileSet(),
		info:        info,
		stateChange: make(chan string),
		unsafePath:  filepath.Join(runtime.GOROOT(), "src", "unsafe"),
	}

	fmt.Printf("unsafe path: %s\n", l.unsafePath)

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
	l.mReady.Lock()
	if l.ready != nil {
		l.mReady.Unlock()
		return l.ready
	}
	go func() {
		fmt.Printf("Start: starting anon go func\n")
		stop := false
		for !stop {
			select {
			case <-l.closer:
				stop = true
			case dPath := <-l.stateChange:
				go l.processStateChange(dPath)
			}
		}

		fmt.Printf("Start: ending anon go func\n")
		close(l.ready)
	}()

	l.ready = make(chan bool)

	l.mReady.Unlock()
	return l.ready
}

// Close stops the loader processing
func (l *Loader) Close() {
	l.closer <- true
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

		fmt.Printf("LoadDirectory: queueing %s\n", l.shortName(dpath))
		l.mDirectories.Lock()
		l.ensureDirectory(dpath)
		l.mDirectories.Unlock()

		return nil
	})
}

func (l *Loader) processStateChange(absPath string) {
	l.mDirectories.Lock()
	d, _ := l.directories[absPath]
	loadState := d.loadState
	l.mDirectories.Unlock()

	fmt.Printf("PSC: %s: current state %q...\n", l.shortName(absPath), d.loadState)

	switch loadState {
	case queued:
		l.processDirectory(d)
		d.loadState++
		d.c.Broadcast()
		l.stateChange <- absPath
	case unloaded:
		l.processGoFiles(d)
		l.processCgoFiles(d)
		l.processPackages(d, false)
		d.loadState++
		fmt.Printf("PSC: %s: broadcasting done!!\n", l.shortName(d.absPath))
		d.c.Broadcast()
		l.stateChange <- d.absPath
	case loadedGo:
		l.processTestGoFiles(d)
		l.processPackages(d, true)
		d.loadState++
		fmt.Printf("PSC: %s: broadcasting done on test!!\n", l.shortName(d.absPath))
		d.c.Broadcast()
		l.stateChange <- d.absPath
	case loadedTest:
		// Short circuiting directly to next state.
		d.loadState++
		d.c.Broadcast()
		l.stateChange <- absPath
	case done:
		fmt.Printf("PSC: %s: done\n", l.shortName(absPath))

		complete := true
		l.mDirectories.Lock()

		for _, d := range l.directories {
			if d.loadState != done {
				fmt.Printf("PSC: %s: Incomplete: %s has state %q\n", l.shortName(absPath), l.shortName(d.absPath), d.loadState)
				complete = false
				break
			}
		}

		l.mDirectories.Unlock()

		if !complete {
			return
		}

		fmt.Printf("DONE DONE\n")

		complete = !l.done
		l.done = true

		if complete {
			fmt.Printf("DONE DONE DONE DONE DONE\n")
			l.processComplete()
		}
	}
}

func (l *Loader) processComplete() {
	// Loop over packages in reverse order of imports and inspect
	checked := map[string]bool{}
	l.caravan.Walk(collections.WalkUp, func(node *collections.Node) {
		p, ok := node.Element.(*Package)
		if !ok {
			panic("Oops, not a package pointer...\n")
		}
		if p.name == "unsafe" {
			fmt.Printf("Checking unsafe (skipping)\n")
			return
		}
		fmt.Printf("Checking %s\n", p.name)
		fmap := l.directories[p.absPath].pm[p.name].files
		files := make([]*ast.File, len(fmap))
		i := 0
		for _, v := range fmap {
			f := v
			files[i] = f
			i++
		}

		key := fmt.Sprintf("%s:%s", p.absPath, p.name)
		if _, ok := checked[key]; ok {
			fmt.Printf("Double checking %s...\n", key)
			return
		}

		typesPkg, err := l.conf.Check(p.absPath, l.fset, files, l.info)
		if err != nil {
			fmt.Printf("Error while checking %s:\n\t%s\n\n", key, err.Error())
			return
		}
		p.typesPkg = typesPkg
		checked[key] = true
	})

	l.ready <- true
}

func (l *Loader) processDirectory(d *Directory) {
	fmt.Printf(" PD: %s\n", l.shortName(d.absPath))

	if l.processUnsafe(d) {
		return
	}

	buildPkg, err := l.context.Import(".", d.absPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			fmt.Printf(" PD: %s: no go code", l.shortName(d.absPath))
			return
		}
		fmt.Printf(" PD: %s: proc error:\n\t%s\n", l.shortName(d.absPath), err.Error())
		return
	}

	d.buildPkg = buildPkg
	d.pm = packageMap{}
}

func (l *Loader) processGoFiles(d *Directory) {
	if d.absPath == l.unsafePath || d.buildPkg == nil {
		return
	}

	fnames := d.buildPkg.GoFiles
	if len(fnames) == 0 {
		return
	}

	for _, fname := range fnames {
		fpath := filepath.Join(d.absPath, fname)
		astf, err := parser.ParseFile(l.fset, fpath, nil, parser.AllErrors)
		if err != nil {
			fmt.Printf(" GF: ERROR: While parsing %s:\n\t%s", fpath, err.Error())
			return
		}

		l.processAstFile(fname, astf, d.pm)
	}
}

func (l *Loader) processCgoFiles(d *Directory) {
	if d.absPath == l.unsafePath || d.buildPkg == nil {
		return
	}

	fnames := d.buildPkg.CgoFiles
	if len(fnames) == 0 {
		return
	}

	fpaths := make([]string, len(fnames))
	for k, v := range fnames {
		fmt.Printf("CGO: %s\n", v)
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

	fmt.Printf("importPath = %s\n", d.buildPkg.ImportPath)
	var cgoflags []string
	if d.buildPkg.Goroot && d.buildPkg.ImportPath == "runtime/cgo" {
		cgoflags = append(cgoflags, "-import_runtime_cgo=false")
	}
	if d.buildPkg.Goroot && d.buildPkg.ImportPath == "runtime/race" || d.buildPkg.ImportPath == "runtime/cgo" {
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
	cmd.Stdout = os.Stdout // os.Stderr
	cmd.Stderr = os.Stdout // os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("CGO: ERROR: cgo failed: %s: %s", args, err)
		return
	}

	fmt.Printf("CGO: Processing %d cgo based files\n", len(fpaths))
	for i := range fpaths {
		f, err := os.Open(files[i])
		if err != nil {
			fmt.Printf("CGO: ERROR: failed to open file %s\n\t%s\n", files[i], err.Error())
			continue
		}
		astf, err := parser.ParseFile(l.fset, displayFiles[i], f, 0)
		f.Close()
		if err != nil {
			fmt.Printf("CGO: ERROR: Failed to open %s\n\t%s\n", fpaths[i], err.Error())
			return
		}

		l.processAstFile(fnames[i], astf, d.pm)
	}
}

func (l *Loader) processTestGoFiles(d *Directory) {
	if d.absPath == l.unsafePath || d.buildPkg == nil {
		return
	}

	fnames := d.buildPkg.TestGoFiles
	if len(fnames) == 0 {
		// No test files; continue on.
		fmt.Printf("TFG: %s: no test Go files\n", l.shortName(d.absPath))
		return
	}

	fmt.Printf("TFG: %s: processing %d test Go files\n", l.shortName(d.absPath), len(fnames))
	for _, fname := range fnames {
		fpath := filepath.Join(d.absPath, fname)
		astf, err := parser.ParseFile(l.fset, fpath, nil, parser.AllErrors)
		if err != nil {
			fmt.Printf("TGF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
			return
		}

		l.processAstFile(fname, astf, d.pm)
	}
	fmt.Printf("TFG: %s: processing complete\n", l.shortName(d.absPath))
}

func (l *Loader) processAstFile(fname string, astf *ast.File, pm packageMap) {
	pkgName := filepath.Base(astf.Name.Name)

	pmi, ok := pm[pkgName]
	if !ok {
		pmi = &packageMapItem{
			files:   map[string]*ast.File{},
			imports: map[string]bool{},
		}
		pm[pkgName] = pmi
	}

	pmi.m.Lock()

	for _, decl := range astf.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.IMPORT {
			continue
		}

		for _, spec := range decl.Specs {
			spec := spec.(*ast.ImportSpec)

			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path == "C" {
				// Ignore any error and skip the C pseudo package
				continue
			}
			pmi.imports[path] = true
		}
	}

	pmi.files[fname] = astf

	pmi.m.Unlock()
}

func (l *Loader) processUnsafe(d *Directory) bool {
	absPath := d.absPath
	if strings.Compare(absPath, l.unsafePath) != 0 {
		return false
	}
	fmt.Printf("*** Loading `%s`, replacing with types.Unsafe\n", l.shortName(d.absPath))
	p := CreatePackage(l.unsafePath, "unsafe")
	p.typesPkg = types.Unsafe
	d.packages = map[string]*Package{
		"unsafe": p,
	}
	l.caravan.Insert(p)

	return true
}

func (l *Loader) processPackages(d *Directory, testing bool) {
	fmt.Printf(" PP: %s: %d: started\n", l.shortName(d.absPath), d.loadState)
	importMaps := map[*PackageKey]map[*PackageKey]bool{}

	for pkgName, pmi := range d.pm {
		thisPkgName := filepath.Base(pkgName)
		p, ok := d.packages[thisPkgName]
		if !ok {
			fmt.Printf(" PP: %s: %d: creating package w/ %s:%s\n", l.shortName(d.absPath), d.loadState, d.absPath, thisPkgName)
			p = CreatePackage(d.absPath, thisPkgName)
			l.caravan.Insert(p)
			d.packages[thisPkgName] = p
		}
		pmi.p = p

		fmt.Printf(" PP: %s: %d: package %s\n", l.shortName(d.absPath), d.loadState, pmi.p.name)
		pmi.m.Lock()
		sourceKey := &PackageKey{
			absPath: pmi.p.absPath,
			name:    pmi.p.name,
		}
		for importPkgName := range pmi.imports {
			importPath, err := l.findImportPath(importPkgName, d.absPath)
			if err != nil {
				fmt.Printf(err.Error())
				continue
			}

			fmt.Printf(" PP: %s: %d: -> %s\n", l.shortName(d.absPath), d.loadState, l.shortName(importPath))
			pmi.p.importPaths[importPath] = false
			destinationKey := &PackageKey{
				absPath: importPath,
				name:    filepath.Base(importPkgName),
			}
			destinationKeys, ok := importMaps[sourceKey]
			if !ok {
				destinationKeys = map[*PackageKey]bool{}
				importMaps[sourceKey] = destinationKeys
			}
			destinationKeys[destinationKey] = true

			l.mDirectories.Lock()
			l.ensureDirectory(importPath)
			l.mDirectories.Unlock()
		}
		pmi.m.Unlock()
	}

	for sourceKey, destinationKeys := range importMaps {
		l.mDirectories.Lock()
		sourcePackage := l.directories[sourceKey.absPath].packages[sourceKey.name]
		l.mDirectories.Unlock()

		for destinationKey := range destinationKeys {
			l.mDirectories.Lock()
			targetD, _ := l.directories[destinationKey.absPath]
			l.mDirectories.Unlock()

			targetD.m.Lock()

			for !l.checkImportReady(d, targetD) {
				fmt.Printf(" PP: %s: %d: *** still waiting on %s ***\n", l.shortName(d.absPath), d.loadState, l.shortName(targetD.absPath))
				targetD.c.Wait()
			}

			targetD.m.Unlock()

			targetPackage, ok := targetD.packages[destinationKey.name]
			if !ok {
				panic(fmt.Sprintf(" PP: %s: %d: target package %s:%s is !ok\n", l.shortName(d.absPath), d.loadState, destinationKey.absPath, destinationKey.name))
			}

			if testing {
				fmt.Printf(" PP: %s: %d: weak connecting to %s:%s\n", l.shortName(d.absPath), d.loadState, destinationKey.absPath, destinationKey.name)
				if err := l.caravan.WeakConnect(sourcePackage, targetPackage); err != nil {
					panic(fmt.Sprintf(" PP: %s: %d: weak connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", l.shortName(d.absPath), d.loadState, sourcePackage.key, targetPackage.key, err.Error()))
				}
			} else {
				fmt.Printf(" PP: %s: %d: connecting to %s:%s\n", l.shortName(d.absPath), d.loadState, destinationKey.absPath, destinationKey.name)
				if err := l.caravan.Connect(sourcePackage, targetPackage); err != nil {
					panic(fmt.Sprintf(" PP: %s: %d: connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", l.shortName(d.absPath), d.loadState, sourcePackage.key, targetPackage.key, err.Error()))
				}
			}

			// All dependencies are loaded; can proceed.
			fmt.Printf(" PP: %s: %d: dep %s OK\n", l.shortName(d.absPath), d.loadState, l.shortName(targetD.absPath))
		}
	}
	fmt.Printf(" PP: %s: %d: all imports fulfilled.\n", l.shortName(d.absPath), d.loadState)
}

// ensureDirectory assumes that the caller has the mDirectories mutex
func (l *Loader) checkImportReady(sourceD *Directory, targetD *Directory) bool {
	// return targetD.loadState == done || sourceD.loadState < targetD.loadState

	switch sourceD.loadState {
	case queued:
		// Does not make sense that the source loadState would be here.
	case unloaded:
		return targetD.loadState > unloaded
	case loadedGo:
		return targetD.loadState > unloaded
	case loadedTest:
		// Should pass through here.
	default:
		// Should never get here.
	}

	return false
}

func (l *Loader) ensureDirectory(absPath string) *Directory {
	d, ok := l.directories[absPath]
	if !ok {
		d = &Directory{
			absPath:   absPath,
			loadState: queued,
			packages:  map[string]*Package{},
		}
		d.c = sync.NewCond(&d.m)
		l.directories[absPath] = d
		l.stateChange <- absPath
	}

	return d
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

func (l *Loader) shortName(path string) string {
	root := runtime.GOROOT()
	if strings.HasPrefix(path, root) {
		return path[utf8.RuneCountInString(root)+5:]
	}
	return filepath.Base(path)
}
