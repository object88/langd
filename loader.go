package langd

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
	"github.com/object88/rope"
)

type loadState int32

const (
	queued loadState = iota
	unloaded
	loadedGo
	loadedTest
	done
)

func (ls *loadState) increment() int32 {
	return atomic.AddInt32((*int32)(ls), 1)
}

func (ls *loadState) get() loadState {
	return loadState(atomic.LoadInt32((*int32)(ls)))
}

type stateChangeEvent struct {
	lc  *LoaderContext
	key collections.Key
}

// Loader is a Go code loader
type Loader struct {
	mReady sync.Mutex
	closer chan bool
	ready  chan bool

	done bool

	caravan *collections.Caravan

	stateChange chan *stateChangeEvent

	openedFiles map[string]*rope.Rope

	Log *log.Log
}

// FileError is a translation of the types.Error struct
type FileError struct {
	token.Position
	Message string
	Warning bool
}

// File is an AST file and any errors that types.Config.Check discovers
type File struct {
	file *ast.File
	errs []FileError
}

// Package is the contents of a package
type Package struct {
	key       collections.Key
	AbsPath   string
	GOARCH    string
	GOOS      string
	Tags      string
	shortPath string

	buildPkg *build.Package
	checker  *types.Checker
	Fset     *token.FileSet
	typesPkg *types.Package

	files           map[string]*File
	importPaths     map[string]bool
	testFiles       map[string]*File
	testImportPaths map[string]bool

	loadState loadState
	m         sync.Mutex
	c         *sync.Cond
}

// Key returns the collection key for the given Package
func (p *Package) Key() collections.Key {
	return p.key
}

// ResetChecker sets the checker to nil
func (p *Package) ResetChecker() {
	p.checker = nil
}

func (p *Package) String() string {
	return p.shortPath
}

func (p *Package) currentFiles() map[string]*File {
	loadState := p.loadState.get()
	switch loadState {
	case unloaded:
		if p.files == nil {
			p.files = map[string]*File{}
		}
		return p.files
	case loadedGo:
		if p.testFiles == nil {
			p.testFiles = map[string]*File{}
		}
		return p.testFiles
	}
	fmt.Printf("Package '%s' has loadState %d; no files.\n", p.AbsPath, loadState)
	return nil
}

var cgoRe = regexp.MustCompile(`[/\\:]`)

// NewLoader creates a new loader
func NewLoader() *Loader {
	l := &Loader{
		caravan:     collections.CreateCaravan(),
		closer:      make(chan bool),
		done:        false,
		Log:         log.Stdout(),
		openedFiles: map[string]*rope.Rope{},
		stateChange: make(chan *stateChangeEvent),
	}

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
		stop := false
		for !stop {
			select {
			case <-l.closer:
				stop = true
			case e := <-l.stateChange:
				go l.processStateChange(e)
			}
		}

		l.Log.Debugf("Start: ending anon go func\n")
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

// Errors exposes problems with code found during compilation on a file-by-file
// basis.
func (l *Loader) Errors(handleErrs func(file string, errs []FileError)) {
	l.caravan.Iter(func(key collections.Key, node *collections.Node) bool {
		p := node.Element.(*Package)
		for fname, f := range p.files {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.AbsPath, fname), f.errs)
			}
		}
		for fname, f := range p.testFiles {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.AbsPath, fname), f.errs)
			}
		}
		return true
	})
}

// LoadDirectory adds the contents of a directory to the Loader
func (l *Loader) LoadDirectory(lc *LoaderContext, path string) error {
	if !lc.context.IsDir(path) {
		return fmt.Errorf("Argument '%s' is not a directory", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("Could not get absolute path for '%s'", absPath)
	}

	lc.startDir = absPath

	l.readDir(lc, absPath)
	return nil
}

func (l *Loader) readDir(lc *LoaderContext, absPath string) {
	for _, g := range lc.filteredPaths {
		if g.Match(absPath) {
			// We are looking at a filtered out path.
			return
		}
	}

	l.Log.Debugf("readDir: queueing '%s'...\n", absPath)

	l.ensurePackage(lc, absPath)

	fis, err := lc.context.ReadDir(absPath)
	if err != nil {
		panic(fmt.Sprintf("Dang:\n\t%s", err.Error()))
	}
	for _, fi := range fis {
		// Ignore individual files
		if !fi.IsDir() {
			continue
		}

		if fi.Name() == "vendor" {
			continue
		}

		l.readDir(lc, filepath.Join(absPath, fi.Name()))
	}
}

func (l *Loader) processStateChange(sce *stateChangeEvent) {
	n, _ := l.caravan.Find(sce.key)
	p := n.Element.(*Package)

	loadState := p.loadState.get()

	l.Log.Debugf("PSC: %s: current state: %d\n", p, loadState)

	switch loadState {
	case queued:
		l.processDirectory(sce.lc, p)

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- sce
	case unloaded:
		haveGo := l.processGoFiles(sce.lc, p)
		haveCgo := l.processCgoFiles(sce.lc, p)
		if (haveGo || haveCgo) && p.buildPkg != nil {
			imports := importPathMapToArray(p.importPaths)
			l.processPackages(sce.lc, p, imports, false)
			l.processComplete(sce.lc, p)
		}

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- sce
	case loadedGo:
		haveTestGo := l.processTestGoFiles(sce.lc, p)
		if haveTestGo && p.buildPkg != nil {
			imports := importPathMapToArray(p.testImportPaths)
			l.processPackages(sce.lc, p, imports, true)
			l.processComplete(sce.lc, p)
		}

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- sce
	case loadedTest:
		// Short circuiting directly to next state.  Will add external test
		// packages later.

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- sce
	case done:
		complete := true

		l.caravan.Iter(func(_ collections.Key, n *collections.Node) bool {
			targetP := n.Element.(*Package)
			targetLoadState := targetP.loadState.get()
			if targetLoadState != done {
				complete = false
			}
			return complete
		})

		if !complete {
			return
		}

		complete = !l.done
		l.done = true

		if complete {
			l.Log.Debugf("All packages are loaded\n")
			l.ready <- true
		}
	}
}

func importPathMapToArray(imports map[string]bool) []string {
	results := make([]string, len(imports))
	i := 0
	for k := range imports {
		results[i] = k
		i++
	}
	return results
}

func (l *Loader) processComplete(lc *LoaderContext, p *Package) {
	if lc.IsUnsafe(p) {
		l.Log.Debugf(" PC: %s: Checking unsafe (skipping)\n", p)
		return
	}

	if p.checker == nil {
		info := &types.Info{
			Defs:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Scopes:     map[ast.Node]*types.Scope{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Types:      map[ast.Expr]types.TypeAndValue{},
			Uses:       map[*ast.Ident]types.Object{},
		}

		p.typesPkg = types.NewPackage(p.AbsPath, p.buildPkg.Name)
		p.checker = types.NewChecker(lc.config, p.Fset, p.typesPkg, info)
	}

	// Clear previous errors; all will be rechecked.
	files := p.currentFiles()

	// Loop over packages
	astFiles := make([]*ast.File, len(files))
	i := 0
	for _, v := range files {
		f := v
		f.errs = []FileError{}
		astFiles[i] = f.file
		i++
	}

	lc.checkerMu.Lock()
	err := p.checker.Files(astFiles)
	lc.checkerMu.Unlock()

	if err != nil {
		l.Log.Debugf("Error while checking %s:\n\t%s\n\n", p.AbsPath, err.Error())
	}
	if !p.typesPkg.Complete() {
		l.Log.Debugf("Incomplete package %s\n", p.AbsPath)
	}
}

func (l *Loader) processDirectory(lc *LoaderContext, p *Package) {
	if l.processUnsafe(lc, p) {
		return
	}

	buildPkg, err := lc.context.Import(".", p.AbsPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			return
		}
		l.Log.Debugf(" PD: %s: proc error:\n\t%s\n", p, err.Error())
		return
	}

	p.buildPkg = buildPkg
}

func (l *Loader) processGoFiles(lc *LoaderContext, p *Package) bool {
	if lc.IsUnsafe(p) {
		return true
	}

	if p.buildPkg == nil {
		return false
	}

	fnames := p.buildPkg.GoFiles
	if len(fnames) == 0 {
		return false
	}

	for _, fname := range fnames {
		fpath := filepath.Join(p.AbsPath, fname)

		var r io.Reader
		if of, ok := l.openedFiles[fpath]; ok {
			r = of.NewReader()
		} else {
			var err error
			r, err = lc.context.OpenFile(fpath)
			if err != nil {
				l.Log.Debugf(" GF: ERROR: Failed to read file %s:\n\t%s\n", fpath, err.Error())
				continue
			}
		}

		astf, err := parser.ParseFile(p.Fset, fpath, r, parser.AllErrors)

		if c, ok := r.(io.Closer); ok {
			c.Close()
		}

		if err != nil {
			l.Log.Debugf(" GF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
		}

		l.processAstFile(p, fname, astf, p.importPaths)
	}

	return true
}

// Return the flags to use when invoking the C or C++ compilers, or cgo.
func cflags(p *build.Package, def bool) (cppflags, cflags, cxxflags, ldflags []string) {
	var defaults string
	if def {
		defaults = "-g -O2"
	}

	cppflags = stringList(envList("CGO_CPPFLAGS", ""), p.CgoCPPFLAGS)
	cflags = stringList(envList("CGO_CFLAGS", defaults), p.CgoCFLAGS)
	cxxflags = stringList(envList("CGO_CXXFLAGS", defaults), p.CgoCXXFLAGS)
	ldflags = stringList(envList("CGO_LDFLAGS", defaults), p.CgoLDFLAGS)
	return
}

// envList returns the value of the given environment variable broken
// into fields, using the default value when the variable is empty.
func envList(key, def string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = def
	}
	return strings.Fields(v)
}

// stringList's arguments should be a sequence of string or []string values.
// stringList flattens them into a single []string.
func stringList(args ...interface{}) []string {
	var x []string
	for _, arg := range args {
		switch arg := arg.(type) {
		case []string:
			x = append(x, arg...)
		case string:
			x = append(x, arg)
		default:
			panic("stringList: invalid argument")
		}
	}
	return x
}

// pkgConfig runs pkg-config with the specified arguments and returns the flags it prints.
func pkgConfig(mode string, pkgs []string) (flags []string, err error) {
	cmd := exec.Command("pkg-config", append([]string{mode}, pkgs...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s := fmt.Sprintf("%s failed: %v", strings.Join(cmd.Args, " "), err)
		if len(out) > 0 {
			s = fmt.Sprintf("%s: %s", s, out)
		}
		return nil, errors.New(s)
	}
	if len(out) > 0 {
		flags = strings.Fields(string(out))
	}
	return
}

// pkgConfigFlags calls pkg-config if needed and returns the cflags
// needed to build the package.
func pkgConfigFlags(p *build.Package) (cflags []string, err error) {
	if len(p.CgoPkgConfig) == 0 {
		return nil, nil
	}
	return pkgConfig("--cflags", p.CgoPkgConfig)
}

func (l *Loader) processCgoFiles(lc *LoaderContext, p *Package) bool {
	if lc.IsUnsafe(p) {
		return true
	}

	if p.buildPkg == nil {
		return false
	}

	fnames := p.buildPkg.CgoFiles
	if len(fnames) == 0 {
		return false
	}

	cgoCPPFLAGS, _, _, _ := cflags(p.buildPkg, true)
	_, cgoexeCFLAGS, _, _ := cflags(p.buildPkg, false)

	if len(p.buildPkg.CgoPkgConfig) > 0 {
		pcCFLAGS, err := pkgConfigFlags(p.buildPkg)
		if err != nil {
			l.Log.Debugf("CGO: %s: Failed to get flags: %s\n", p, err.Error())
			return false
		}
		cgoCPPFLAGS = append(cgoCPPFLAGS, pcCFLAGS...)
	}

	fpaths := make([]string, len(fnames))
	for k, v := range fnames {
		fpaths[k] = filepath.Join(p.AbsPath, v)
	}

	tmpdir, _ := ioutil.TempDir("", strings.Replace(p.AbsPath, "/", "_", -1)+"_C")
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

	var cgoflags []string
	if p.buildPkg.Goroot && p.buildPkg.ImportPath == "runtime/cgo" {
		cgoflags = append(cgoflags, "-import_runtime_cgo=false")
	}
	if p.buildPkg.Goroot && p.buildPkg.ImportPath == "runtime/race" || p.buildPkg.ImportPath == "runtime/cgo" {
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
	for _, v := range cgoCPPFLAGS {
		args = append(args, v)
	}
	for _, v := range cgoexeCFLAGS {
		args = append(args, v)
	}
	for _, f := range fnames {
		args = append(args, f)
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = p.AbsPath
	cmd.Stdout = os.Stdout // os.Stderr
	cmd.Stderr = os.Stdout // os.Stderr
	if err := cmd.Run(); err != nil {
		l.Log.Debugf("CGO: %s: ERROR: cgo failed: %s\n\t%s\n", p, args, err.Error())
		return false
	}

	for i, fname := range files {
		f, err := os.Open(fname)
		if err != nil {
			l.Log.Debugf("CGO: %s: ERROR: failed to open file %s\n\t%s\n", p, fname, err.Error())
			continue
		}

		astf, err := parser.ParseFile(p.Fset, displayFiles[i], f, 0)

		f.Close()

		if err != nil {
			l.Log.Debugf("CGO: %s: ERROR: Failed to parse %s\n\t%s\n", p, fname, err.Error())
		}

		l.processAstFile(p, fname, astf, p.importPaths)
	}
	l.Log.Debugf("CGO: %s: Done processing\n", p)

	return true
}

func (l *Loader) processTestGoFiles(lc *LoaderContext, p *Package) bool {
	if lc.IsUnsafe(p) || p.buildPkg == nil {
		return false
	}

	// If we are in vendor; exclude test files.  It is possible for imports to
	// contain references for packages which are not available.  May want to
	// revisit this later; loading as much as possible for completion sake,
	// but not reporting them as complete errors.
	for _, part := range strings.Split(p.AbsPath, string(filepath.Separator)) {
		if part == "vendor" {
			return false
		}
	}

	fnames := p.buildPkg.TestGoFiles
	if len(fnames) == 0 {
		// No test files; continue on.
		return false
	}

	l.Log.Debugf("TFG: %s: processing %d test Go files\n", p, len(fnames))
	for _, fname := range fnames {
		fpath := filepath.Join(p.AbsPath, fname)

		r, err := lc.context.OpenFile(fpath)
		if err != nil {
			l.Log.Debugf("TGF: ERROR: Failed to read file %s:\n\t%s\n", fpath, err.Error())
			continue
		}

		astf, err := parser.ParseFile(p.Fset, fpath, r, parser.AllErrors)

		r.Close()

		if err != nil {
			l.Log.Debugf("TGF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
		}

		l.processAstFile(p, fname, astf, p.testImportPaths)
	}

	l.Log.Debugf("TFG: %s: processing complete\n", p)
	return true
}

func (l *Loader) processAstFile(p *Package, fname string, astf *ast.File, importPaths map[string]bool) {
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

			importPaths[path] = true
		}
	}

	files := p.currentFiles()
	files[fname] = &File{
		errs: []FileError{},
		file: astf,
	}
}

func (l *Loader) processUnsafe(lc *LoaderContext, p *Package) bool {
	if !lc.IsUnsafe(p) {
		return false
	}
	l.Log.Debugf("*** Loading `%s`, replacing with types.Unsafe\n", p)
	p.typesPkg = types.Unsafe

	l.caravan.Insert(p)

	return true
}

func (l *Loader) processPackages(lc *LoaderContext, p *Package, importPaths []string, testing bool) {
	loadState := p.loadState.get()
	l.Log.Debugf(" PP: %s: %d: started\n", p, loadState)

	importedPackages := map[string]bool{}

	for _, importPath := range importPaths {
		targetPath, err := lc.findImportPath(importPath, p.AbsPath)
		if err != nil {
			l.Log.Debugf(" PP: %s: %d: Failed to find import %s\n\t%s\n", p, loadState, importPath, err.Error())
			continue
		}
		if targetPath == p.AbsPath {

			l.Log.Debugf(" PP: %s: %d: Failed due to self-import\n", p, loadState)
			continue
		}
		l.ensurePackage(lc, targetPath)

		importedPackages[targetPath] = true
	}

	// TEMPORARY
	func() {
		imprts := []string{}
		for importedPackage := range importedPackages {
			n, ok := l.caravan.Find(lc.BuildKey(importedPackage))
			if !ok {
				continue
			}
			targetP := n.Element.(*Package)
			imprts = append(imprts, targetP.String())
		}
		allImprts := strings.Join(imprts, ", ")
		l.Log.Debugf(" PP: %s: %d: -> %s\n", p, loadState, allImprts)
	}()

	for importPath := range importedPackages {
		n, ok := l.caravan.Find(lc.BuildKey(importPath))
		if !ok {
			l.Log.Debugf(" PP: %s: %d: import path is missing: %s\n", p, loadState, importPath)
			continue
		}
		targetP := n.Element.(*Package)

		targetP.m.Lock()
		for !l.checkImportReady(loadState, targetP) {
			l.Log.Debugf(" PP: %s: %d: *** still waiting on %s ***\n", p, loadState, targetP)
			targetP.c.Wait()
		}
		targetP.m.Unlock()

		var err error

		if testing {
			err = l.caravan.WeakConnect(p, targetP)
		} else {
			err = l.caravan.Connect(p, targetP)
		}

		if err != nil {
			panic(fmt.Sprintf(" PP: %s: %d: [weak] connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", p, loadState, p, targetP, err.Error()))
		}
	}
	// All dependencies are loaded; can proceed.
	l.Log.Debugf(" PP: %s: %d: all imports fulfilled.\n", p, loadState)
}

func (l *Loader) checkImportReady(sourceLoadState loadState, targetP *Package) bool {
	targetLoadState := targetP.loadState.get()

	switch sourceLoadState {
	case queued:
		// Does not make sense that the source loadState would be here.
	case unloaded:
		return targetLoadState > unloaded
	case loadedGo:
		return targetLoadState > unloaded
	case loadedTest:
		// Should pass through here.
	default:
		// Should never get here.
	}

	return false
}

func (l *Loader) ensurePackage(lc *LoaderContext, absPath string) *Package {
	key := lc.BuildKey(absPath)
	n, created := l.caravan.Ensure(key, func() collections.Keyer {
		shortPath := absPath
		if strings.HasPrefix(absPath, lc.context.GOROOT) {
			shortPath = fmt.Sprintf("(%s, %s, stdlib) %s", lc.context.GOARCH, lc.context.GOOS, absPath[utf8.RuneCountInString(lc.context.GOROOT)+5:])
		} else {
			// Shorten the canonical name for logging purposes.
			n := utf8.RuneCountInString(lc.startDir)
			if len(absPath) >= n {
				shortPath = absPath[n:]
			}
			shortPath = fmt.Sprintf("(%s, %s) %s", lc.context.GOARCH, lc.context.GOOS, shortPath)
		}

		p := &Package{
			AbsPath:         absPath,
			GOARCH:          lc.context.GOARCH,
			GOOS:            lc.context.GOOS,
			key:             key,
			shortPath:       shortPath,
			Fset:            token.NewFileSet(),
			importPaths:     map[string]bool{},
			testImportPaths: map[string]bool{},
		}
		p.c = sync.NewCond(&p.m)

		l.Log.Debugf("ensurePackage: creating package for '%s' at 0x%x.\n", p.String(), p.Key())

		return p
	})
	p := n.Element.(*Package)

	if created {
		l.stateChange <- &stateChangeEvent{
			lc:  lc,
			key: p.Key(),
		}
	}

	return p
}
