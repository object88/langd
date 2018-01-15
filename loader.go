package langd

import (
	"bytes"
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
	"unicode/utf8"

	"github.com/gobwas/glob"
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

	caravanMutex sync.Mutex
	caravan      *collections.Caravan

	stateChange chan string

	conf    *types.Config
	context *build.Context
	mFset   sync.Mutex
	fset    *token.FileSet
	info    *types.Info

	unsafePath    string
	filteredPaths []glob.Glob
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
	absPath string
	name    string

	buildPkg        *build.Package
	files           map[string]*File
	importPaths     map[string]bool
	testImportPaths map[string]bool
	typesPkg        *types.Package

	loadState loadState
	m         sync.Mutex
	c         *sync.Cond
}

// Key returns the collections.Key for the given Package
func (p *Package) Key() collections.Key {
	return collections.Key(p.absPath)
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
		fset:          token.NewFileSet(),
		info: &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{},
			Defs:  map[*ast.Ident]types.Object{},
			Uses:  map[*ast.Ident]types.Object{},
		},
		stateChange: make(chan string),
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
				l.caravanMutex.Lock()
				node, ok := l.caravan.Find(collections.Key(absPath))
				l.caravanMutex.Unlock()

				if !ok {
					fmt.Printf("ERROR: (missing) No package for %s\n", absPath)
					return
				}

				baseFilename := filepath.Base(position.Filename)
				ferr := FileError{
					Position: position,
					Message:  terror.Msg,
					Warning:  terror.Soft,
				}
				p := node.Element.(*Package)
				f, ok := p.files[baseFilename]
				if !ok {
					fmt.Printf("ERROR: (missing file) %s\n", position.Filename)
				} else {
					f.errs = append(f.errs, ferr)

					fmt.Printf("ERROR: (types error) %s\n", terror.Error())
				}
			} else {
				fmt.Printf("ERROR: (unknown) %#v\n", e)
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

// Errors exposes problems with code found during compilation on a file-by-file
// basis.
func (l *Loader) Errors(handleErrs func(file string, errs []FileError)) {
	l.caravanMutex.Lock()
	for k := range l.caravan.Iter() {
		p := k.(*Package)
		for fname, f := range p.files {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.absPath, fname), f.errs)
			}
		}
	}
	l.caravanMutex.Unlock()
}

// LoadDirectory adds the contents of a directory to the Loader
func (l *Loader) LoadDirectory(absPath string) {
	if !l.context.IsDir(absPath) {
		fmt.Printf("Argument '%s' is not a directory\n", absPath)
		return
	}
	l.readDir(absPath)
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
	l.caravanMutex.Lock()
	n, _ := l.caravan.Find(collections.Key(absPath))
	l.caravanMutex.Unlock()
	p := n.Element.(*Package)

	fmt.Printf("PSC: %s: current state: %d\n", l.shortName(absPath), p.loadState)

	switch p.loadState {
	case queued:
		l.processDirectory(p)
		p.loadState++
		p.c.Broadcast()
		l.stateChange <- absPath
	case unloaded:
		haveGo := l.processGoFiles(p)
		haveCgo := l.processCgoFiles(p)
		if (haveGo || haveCgo) && p.buildPkg != nil {
			imports := make([]string, len(p.importPaths))
			i := 0
			for k := range p.importPaths {
				imports[i] = k
				i++
			}

			fmt.Printf("PSC: %s: haveGo: %t; haveCGo: %t\n", l.shortName(absPath), haveGo, haveCgo)
			l.processPackages(p, imports, false)
			l.processComplete(p)
		}
		p.loadState++
		p.c.Broadcast()
		l.stateChange <- absPath
	case loadedGo:
		// haveTestGo := l.processTestGoFiles(p)
		// if haveTestGo && p.buildPkg != nil {
		// 	imports := make([]string, len(p.testImportPaths))
		// 	i := 0
		// 	for k := range p.testImportPaths {
		// 		imports[i] = k
		// 		i++
		// 	}

		// 	l.processPackages(p, imports, true)
		// 	l.processComplete(p)
		// }
		p.loadState++
		p.c.Broadcast()
		l.stateChange <- absPath
	case loadedTest:
		// Short circuiting directly to next state.  Will add external test
		// packages later.
		p.loadState++
		p.c.Broadcast()
		l.stateChange <- absPath
	case done:
		complete := true
		l.caravanMutex.Lock()

		for n := range l.caravan.Iter() {
			p := n.(*Package)
			if p.loadState != done {
				complete = false
				break
			}
		}

		l.caravanMutex.Unlock()

		if !complete {
			return
		}

		fmt.Printf("DONE DONE\n")

		complete = !l.done
		l.done = true

		if complete {
			fmt.Printf("DONE DONE DONE DONE DONE\n")
			l.ready <- true
		}
	}
}

func (l *Loader) processComplete(p *Package) {
	if p.absPath == l.unsafePath {
		fmt.Printf(" PC: %s: Checking unsafe (skipping)\n", l.shortName(p.absPath))
		return
	}

	// Clear previous errors; all will be rechecked.
	for _, f := range p.files {
		f.errs = []FileError{}
	}

	// Loop over packages
	fmt.Printf(" PC: %s: Checking %s, %d files\n", l.shortName(p.absPath), p.name, len(p.files))
	files := make([]*ast.File, len(p.files))
	i := 0
	for _, v := range p.files {
		// fmt.Fprintf(buf, "\t%s\n", l.fset.Position(v.file.Pos()).Filename)
		f := v
		files[i] = f.file
		i++
	}

	l.mFset.Lock()
	typesPkg, err := l.conf.Check(p.absPath, l.fset, files, l.info)
	l.mFset.Unlock()
	if err != nil {
		fmt.Printf("Error while checking %s:%s:\n\t%s\n\n", p.absPath, p.name, err.Error())
	}
	if !typesPkg.Complete() {
		fmt.Printf("Incomplete package %s:%s\n", p.absPath, p.name)
	}
	p.typesPkg = typesPkg
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
		fmt.Printf(" PD: %s: proc error:\n\t%s\n", l.shortName(p.absPath), err.Error())
		return
	}

	p.buildPkg = buildPkg
	if p.name == "" {
		p.name = buildPkg.Name
	}
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

		r, err := l.context.OpenFile(fpath)
		if err != nil {
			fmt.Printf(" GF: ERROR: Failed to read file %s:\n\t%s\n", fpath, err.Error())
			continue
		}

		l.mFset.Lock()
		astf, err := parser.ParseFile(l.fset, fpath, r, parser.AllErrors)
		l.mFset.Unlock()

		if err != nil {
			fmt.Printf(" GF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
		}

		// fmt.Printf(" GF: %s: Processing AST %s\n", l.shortName(p.absPath), fpath)

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

	buf := bytes.NewBufferString("")
	// fmt.Fprintf(buf, "CGO: %s: have %d cgo files\n", l.shortName(p.absPath), len(fnames))

	cgoCPPFLAGS, _, _, _ := cflags(p.buildPkg, true)
	_, cgoexeCFLAGS, _, _ := cflags(p.buildPkg, false)

	if len(p.buildPkg.CgoPkgConfig) > 0 {
		pcCFLAGS, err := pkgConfigFlags(p.buildPkg)
		if err != nil {
			fmt.Fprintf(buf, "CGO: %s: Failed to get flags: %s\n", l.shortName(p.absPath), err.Error())
			return false
		}
		cgoCPPFLAGS = append(cgoCPPFLAGS, pcCFLAGS...)
	}
	// fmt.Fprintf(buf, "CGO: %s: CgoPkgConfig: %#v\n", l.shortName(p.absPath), p.buildPkg.CgoPkgConfig)

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

	fmt.Fprintf(buf, "CGO: %s: importPath = %s\n", l.shortName(p.absPath), p.buildPkg.ImportPath)
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
		fmt.Fprintf(buf, "CGO: %s: ERROR: cgo failed: %s\n\t%s\n", l.shortName(p.absPath), args, err.Error())
		return false
	}

	// fmt.Fprintf(buf, "CGO: %s: Processing %d cgo based files\n", l.shortName(p.absPath), len(fpaths))
	for i, fname := range files {
		f, err := os.Open(fname)
		if err != nil {
			fmt.Fprintf(buf, "CGO: %s: ERROR: failed to open file %s\n\t%s\n", l.shortName(p.absPath), fname, err.Error())
			continue
		}

		// fmt.Fprintf(buf, "CGO: %s: About to parse %s / %s\n", l.shortName(p.absPath), fname, displayFiles[i])

		l.mFset.Lock()
		astf, err := parser.ParseFile(l.fset, displayFiles[i], f, 0)
		l.mFset.Unlock()

		f.Close()

		if err != nil {
			fmt.Fprintf(buf, "CGO: %s: ERROR: Failed to parse %s\n\t%s\n", l.shortName(p.absPath), fname, err.Error())
		}

		// fmt.Fprintf(buf, "CGO: %s: Processing AST %s\n", l.shortName(p.absPath), fname)
		l.processAstFile(p, fname, astf, p.importPaths)
	}
	fmt.Fprintf(buf, "CGO: %s: Done processing\n", l.shortName(p.absPath))

	fmt.Printf(buf.String())

	return true
}

func (l *Loader) processTestGoFiles(p *Package) bool {
	if p.absPath == l.unsafePath || p.buildPkg == nil {
		return false
	}

	fnames := p.buildPkg.TestGoFiles
	if len(fnames) == 0 {
		// No test files; continue on.
		// fmt.Printf("TFG: %s: no test Go files\n", l.shortName(d.absPath))
		return false
	}

	fmt.Printf("TFG: %s: processing %d test Go files\n", l.shortName(p.absPath), len(fnames))
	for _, fname := range fnames {
		fpath := filepath.Join(p.absPath, fname)

		r, err := l.context.OpenFile(fpath)
		if err != nil {
			fmt.Printf("TGF: ERROR: Failed to read file %s:\n\t%s\n", fpath, err.Error())
			continue
		}

		l.mFset.Lock()
		astf, err := parser.ParseFile(l.fset, fpath, r, parser.AllErrors)
		l.mFset.Unlock()

		if err != nil {
			fmt.Printf("TGF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
		}

		l.processAstFile(p, fname, astf, p.testImportPaths)
	}

	fmt.Printf("TFG: %s: processing complete\n", l.shortName(p.absPath))
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

	p.files[fname] = &File{
		errs: []FileError{},
		file: astf,
	}
}

func (l *Loader) processUnsafe(p *Package) bool {
	absPath := p.absPath
	if strings.Compare(absPath, l.unsafePath) != 0 {
		return false
	}
	fmt.Printf("*** Loading `%s`, replacing with types.Unsafe\n", l.shortName(p.absPath))
	p.name = "unsafe"
	p.typesPkg = types.Unsafe
	l.caravan.Insert(p)

	return true
}

func (l *Loader) processPackages(p *Package, importPaths []string, testing bool) {
	buf := bytes.NewBufferString("")
	fmt.Fprintf(buf, " PP: %s: %d: started\n", l.shortName(p.absPath), p.loadState)

	imprts := []string{}
	importedPackages := map[string]bool{}

	for _, importPath := range importPaths {
		targetPkgName := filepath.Base(importPath)
		targetPath, err := l.findImportPath(importPath, p.absPath)
		if err != nil {
			fmt.Printf(err.Error())
			continue
		}
		targetP := l.ensurePackage(targetPath)
		targetP.name = targetPkgName

		imprts = append(imprts, l.shortName(importPath))
		importedPackages[targetPath] = true
	}

	allImprts := strings.Join(imprts, ", ")
	fmt.Fprintf(buf, " PP: %s: %d: -> %s\n", l.shortName(p.absPath), p.loadState, allImprts)
	fmt.Printf(buf.String())

	for importPath := range importedPackages {
		l.caravanMutex.Lock()
		n, _ := l.caravan.Find(collections.Key(importPath))
		l.caravanMutex.Unlock()
		targetP := n.Element.(*Package)

		targetP.m.Lock()

		for !l.checkImportReady(p, targetP) {
			fmt.Printf(" PP: %s: %d: *** still waiting on %s ***\n", l.shortName(p.absPath), p.loadState, l.shortName(targetP.absPath))
			targetP.c.Wait()
		}

		targetP.m.Unlock()

		if testing {
			if err := l.caravan.WeakConnect(p, targetP); err != nil {
				panic(fmt.Sprintf(" PP: %s: %d: weak connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", l.shortName(p.absPath), p.loadState, p.Key(), targetP.Key(), err.Error()))
			}
		} else {
			if err := l.caravan.Connect(p, targetP); err != nil {
				panic(fmt.Sprintf(" PP: %s: %d: connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", l.shortName(p.absPath), p.loadState, p.Key(), targetP.Key(), err.Error()))
			}
		}
	}
	// All dependencies are loaded; can proceed.
	fmt.Printf(" PP: %s: %d: all imports fulfilled.\n", l.shortName(p.absPath), p.loadState)
}

func (l *Loader) checkImportReady(sourceP *Package, targetP *Package) bool {
	// return targetD.loadState == done || sourceD.loadState < targetD.loadState

	switch sourceP.loadState {
	case queued:
		// Does not make sense that the source loadState would be here.
	case unloaded:
		return targetP.loadState > unloaded
	case loadedGo:
		return targetP.loadState > unloaded
	case loadedTest:
		// Should pass through here.
	default:
		// Should never get here.
	}

	return false
}

func (l *Loader) ensurePackage(absPath string) *Package {
	l.caravanMutex.Lock()
	var p *Package
	n, ok := l.caravan.Find(collections.Key(absPath))
	if !ok {
		p = &Package{
			absPath:         absPath,
			files:           map[string]*File{},
			importPaths:     map[string]bool{},
			testImportPaths: map[string]bool{},
		}
		p.c = sync.NewCond(&p.m)
		l.caravan.Insert(p)
	} else {
		p = n.Element.(*Package)
	}
	l.caravanMutex.Unlock()

	if !ok {
		l.stateChange <- absPath
	}

	return p
}

func (l *Loader) findImportPath(path, src string) (string, error) {
	buildPkg, err := l.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Oh dear:\n\tAttempted build.Import('%s', '%s', build.FindOnly)\n\t%s\n", path, src, err.Error())
		fmt.Printf("ERROR: %s", msg)
		return "", errors.New(msg)
	}
	return buildPkg.Dir, nil
}

func (l *Loader) shortName(path string) string {
	root := runtime.GOROOT()
	if strings.HasPrefix(path, root) {
		return path[utf8.RuneCountInString(root):]
	}
	return filepath.Base(path)
}
