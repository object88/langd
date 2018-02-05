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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/gobwas/glob"
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

// Loader is a Go code loader
type Loader struct {
	mReady sync.Mutex
	closer chan bool
	ready  chan bool

	done bool

	// caravanMu sync.Mutex
	caravan *collections.Caravan

	stateChange chan string

	checkerMu sync.Mutex
	conf      *types.Config
	context   *build.Context

	openedFiles map[string]*rope.Rope

	unsafePath    string
	filteredPaths []glob.Glob

	startDir string

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
	absPath   string
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
func (p *Package) Key() string {
	return p.absPath
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
	return nil
}

var cgoRe = regexp.MustCompile(`[/\\:]`)

// LoaderOption provides a hook for NewLoader to set or modify the new loader's
// context
type LoaderOption func(loader *Loader)

// NewLoader creates a new loader
func NewLoader(options ...LoaderOption) *Loader {
	// Skipping directories that start with "." (i.e., .git) and testdata.
	globs := make([]glob.Glob, 2)
	globs[0] = glob.MustCompile(filepath.Join("**", ".*"))
	globs[1] = glob.MustCompile(filepath.Join("**", "testdata"))

	l := &Loader{
		caravan:       collections.CreateCaravan(),
		closer:        make(chan bool),
		done:          false,
		filteredPaths: globs,
		openedFiles:   map[string]*rope.Rope{},
		stateChange:   make(chan string),
	}

	for _, opt := range options {
		opt(l)
	}

	if l.context == nil {
		l.context = &build.Default
	}

	if l.context.IsDir == nil {
		l.context.IsDir = func(path string) bool {
			fi, err := os.Stat(path)
			return err == nil && fi.IsDir()
		}
	}
	if l.context.OpenFile == nil {
		l.context.OpenFile = func(path string) (io.ReadCloser, error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
	}
	if l.context.ReadDir == nil {
		l.context.ReadDir = func(dir string) ([]os.FileInfo, error) {
			return ioutil.ReadDir(dir)
		}
	}

	l.unsafePath = filepath.Join(l.context.GOROOT, "src", "unsafe")

	i := &Importer{
		l: l,
	}
	c := &types.Config{
		Error: func(e error) {
			if terror, ok := e.(types.Error); ok {
				position := terror.Fset.Position(terror.Pos)
				absPath := filepath.Dir(position.Filename)
				node, ok := l.caravan.Find(absPath)

				if !ok {
					l.Log.Debugf("ERROR: (missing) No package for %s\n", absPath)
					return
				}

				baseFilename := filepath.Base(position.Filename)
				ferr := FileError{
					Position: position,
					Message:  terror.Msg,
					Warning:  terror.Soft,
				}
				p := node.Element.(*Package)

				files := p.currentFiles()
				f, ok := files[baseFilename]
				if !ok {
					l.Log.Debugf("ERROR: (missing file) %s\n", position.Filename)
				} else {
					f.errs = append(f.errs, ferr)
					l.Log.Debugf("ERROR: (types error) %s\n", terror.Error())
				}
			} else {
				l.Log.Debugf("ERROR: (unknown) %#v\n", e)
			}
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
		stop := false
		for !stop {
			select {
			case <-l.closer:
				stop = true
			case dPath := <-l.stateChange:
				go l.processStateChange(dPath)
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
	l.caravan.Iter(func(key string, node *collections.Node) bool {
		p := node.Element.(*Package)
		for fname, f := range p.files {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.absPath, fname), f.errs)
			}
		}
		for fname, f := range p.testFiles {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.absPath, fname), f.errs)
			}
		}
		return true
	})
}

// LoadDirectory adds the contents of a directory to the Loader
func (l *Loader) LoadDirectory(absPath string) error {
	if !l.context.IsDir(absPath) {
		return fmt.Errorf("Argument '%s' is not a directory\n", absPath)
	}

	l.startDir = absPath

	l.readDir(absPath)
	return nil
}

func (l *Loader) readDir(absPath string) {
	for _, g := range l.filteredPaths {
		if g.Match(absPath) {
			// We are looking at a filtered out path.
			return
		}
	}

	l.ensurePackage(absPath)

	fis, err := l.context.ReadDir(absPath)
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

		l.readDir(filepath.Join(absPath, fi.Name()))
	}
}

func (l *Loader) processStateChange(absPath string) {
	n, _ := l.caravan.Find(absPath)
	p := n.Element.(*Package)

	loadState := p.loadState.get()

	switch loadState {
	case queued:
		l.Log.Debugf("PSC: %s: current state: %d\n", p, loadState)

		l.processDirectory(p)

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- absPath
	case unloaded:
		l.Log.Debugf("PSC: %s: current state: %d\n", p, loadState)

		haveGo := l.processGoFiles(p)
		haveCgo := l.processCgoFiles(p)
		if (haveGo || haveCgo) && p.buildPkg != nil {
			imports := importPathMapToArray(p.importPaths)
			l.processPackages(p, imports, false)
			l.processComplete(p)
		}

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- absPath
	case loadedGo:
		l.Log.Debugf("PSC: %s: current state: %d\n", p, loadState)

		haveTestGo := l.processTestGoFiles(p)
		if haveTestGo && p.buildPkg != nil {
			imports := importPathMapToArray(p.testImportPaths)
			l.processPackages(p, imports, true)
			l.processComplete(p)
		}

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- absPath
	case loadedTest:
		// Short circuiting directly to next state.  Will add external test
		// packages later.

		p.loadState.increment()
		p.c.Broadcast()
		l.stateChange <- absPath
	case done:
		complete := true

		l.caravan.Iter(func(_ string, n *collections.Node) bool {
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

func (l *Loader) processComplete(p *Package) {
	if p.absPath == l.unsafePath {
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

		p.typesPkg = types.NewPackage(p.absPath, p.buildPkg.Name)
		p.checker = types.NewChecker(l.conf, p.Fset, p.typesPkg, info)
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

	l.checkerMu.Lock()
	err := p.checker.Files(astFiles)
	l.checkerMu.Unlock()

	if err != nil {
		l.Log.Debugf("Error while checking %s:\n\t%s\n\n", p.absPath, err.Error())
	}
	if !p.typesPkg.Complete() {
		l.Log.Debugf("Incomplete package %s\n", p.absPath)
	}
}

func (l *Loader) processDirectory(p *Package) {
	if l.processUnsafe(p) {
		return
	}

	buildPkg, err := l.context.Import(".", p.absPath, 0)
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

func (l *Loader) processGoFiles(p *Package) bool {
	if p.absPath == l.unsafePath {
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
		fpath := filepath.Join(p.absPath, fname)

		var r io.Reader
		if of, ok := l.openedFiles[fpath]; ok {
			r = of.NewReader()
		} else {
			var err error
			r, err = l.context.OpenFile(fpath)
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

func (l *Loader) processCgoFiles(p *Package) bool {
	if p.absPath == l.unsafePath {
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
		fpaths[k] = filepath.Join(p.absPath, v)
	}

	tmpdir, _ := ioutil.TempDir("", strings.Replace(p.absPath, "/", "_", -1)+"_C")
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
	cmd.Dir = p.absPath
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

func (l *Loader) processTestGoFiles(p *Package) bool {
	if p.absPath == l.unsafePath || p.buildPkg == nil {
		return false
	}

	// If we are in vendor; exclude test files.  It is possible for imports to
	// contain references for packages which are not available.  May want to
	// revisit this later; loading as much as possible for completion sake,
	// but not reporting them as complete errors.
	for _, part := range strings.Split(p.absPath, string(filepath.Separator)) {
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
		fpath := filepath.Join(p.absPath, fname)

		r, err := l.context.OpenFile(fpath)
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

func (l *Loader) processUnsafe(p *Package) bool {
	if p.absPath != l.unsafePath {
		return false
	}
	l.Log.Debugf("*** Loading `%s`, replacing with types.Unsafe\n", p)
	p.typesPkg = types.Unsafe

	l.caravan.Insert(p)

	return true
}

func (l *Loader) processPackages(p *Package, importPaths []string, testing bool) {
	loadState := p.loadState.get()
	l.Log.Debugf(" PP: %s: %d: started\n", p, loadState)

	importedPackages := map[string]bool{}

	for _, importPath := range importPaths {
		targetPath, err := l.findImportPath(importPath, p.absPath)
		if err != nil {
			l.Log.Debugf(" PP: %s: %d: Failed to find import %s\n\t%s\n", p, loadState, importPath, err.Error())
			continue
		}
		l.ensurePackage(targetPath)

		importedPackages[targetPath] = true
	}

	// TEMPORARY
	func() {
		imprts := []string{}
		for _, importPath := range importPaths {
			targetPath, err := l.findImportPath(importPath, p.absPath)
			if err != nil {
				continue
			}
			targetP := l.ensurePackage(targetPath)
			imprts = append(imprts, targetP.String())
		}
		allImprts := strings.Join(imprts, ", ")
		l.Log.Debugf(" PP: %s: %d: -> %s\n", p, loadState, allImprts)
	}()

	for importPath := range importedPackages {
		n, ok := l.caravan.Find(importPath)
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
			panic(fmt.Sprintf(" PP: %s: %d: [weak] connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", p, loadState, p.Key(), targetP.Key(), err.Error()))
		}
	}
	// All dependencies are loaded; can proceed.
	l.Log.Debugf(" PP: %s: %d: all imports fulfilled.\n", p, loadState)
}

func (l *Loader) checkImportReady(sourceLoadState loadState, targetP *Package) bool {
	// return targetD.loadState == done || sourceD.loadState < targetD.loadState

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

func (l *Loader) ensurePackage(absPath string) *Package {
	n, created := l.caravan.Ensure(absPath, func() collections.Keyer {
		shortPath := absPath
		root := runtime.GOROOT()
		if strings.HasPrefix(absPath, root) {
			shortPath = fmt.Sprintf("(stdlib) %s", absPath[utf8.RuneCountInString(root)+5:])
		} else {
			// Shorten the canonical name for logging purposes.
			n := utf8.RuneCountInString(l.startDir)
			if len(absPath) >= n {
				shortPath = absPath[n:]
			}
		}

		p := &Package{
			absPath:         absPath,
			shortPath:       shortPath,
			Fset:            token.NewFileSet(),
			importPaths:     map[string]bool{},
			testImportPaths: map[string]bool{},
		}
		p.c = sync.NewCond(&p.m)
		return p
	})
	p := n.Element.(*Package)

	if created {
		l.stateChange <- absPath
	}

	return p

	// l.caravanMu.Lock()
	// var p *Package
	// n, ok := l.caravan.Find(absPath)
	// if !ok {
	// 	shortPath := absPath
	// 	root := runtime.GOROOT()
	// 	if strings.HasPrefix(absPath, root) {
	// 		shortPath = fmt.Sprintf("(stdlib) %s", absPath[utf8.RuneCountInString(root)+5:])
	// 	} else {
	// 		// Shorten the canonical name for logging purposes.
	// 		n := utf8.RuneCountInString(l.startDir)
	// 		if len(absPath) >= n {
	// 			shortPath = absPath[n:]
	// 		}
	// 	}

	// 	p = &Package{
	// 		absPath:         absPath,
	// 		shortPath:       shortPath,
	// 		Fset:            token.NewFileSet(),
	// 		importPaths:     map[string]bool{},
	// 		testImportPaths: map[string]bool{},
	// 	}
	// 	p.c = sync.NewCond(&p.m)
	// 	l.caravan.Insert(p)
	// } else {
	// 	p = n.Element.(*Package)
	// }
	// l.caravanMu.Unlock()

	// if !ok {
	// 	l.stateChange <- absPath
	// }

	// return p
}

func (l *Loader) findImportPath(path, src string) (string, error) {
	buildPkg, err := l.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Failed to find import path:\n\tAttempted build.Import('%s', '%s', build.FindOnly)\n\t%s\n", path, src, err.Error())
		return "", errors.New(msg)
	}
	return buildPkg.Dir, nil
}
