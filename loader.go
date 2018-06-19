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
	"sync/atomic"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
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
	lc   LoaderContext
	hash collections.Hash
}

// Loader is a Go code loader
type Loader interface {
	io.Closer

	// Start() chan bool
	Errors(lc LoaderContext, handleErrs func(file string, errs []FileError))
	LoadDirectory(lc LoaderContext, path string) error
	InvalidatePackage(lc LoaderContext, p *Package)

	Caravan() *collections.Caravan
	OpenedFiles() *OpenedFiles
}

type loader struct {
	closer chan bool

	caravan     *collections.Caravan
	openedFiles *OpenedFiles

	stateChange chan *stateChangeEvent

	Log *log.Log
}

var cgoRe = regexp.MustCompile(`[/\\:]`)

// NewLoader creates a new loader
func NewLoader() Loader {
	l := &loader{
		caravan:     collections.CreateCaravan(),
		closer:      make(chan bool),
		Log:         log.Stdout(),
		openedFiles: NewOpenedFiles(),
		stateChange: make(chan *stateChangeEvent),
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
	}()

	return l
}

func (l *loader) Caravan() *collections.Caravan {
	return l.caravan
}

func (l *loader) OpenedFiles() *OpenedFiles {
	return l.openedFiles
}

func (l *loader) InvalidatePackage(lc LoaderContext, p *Package) {
	p.Invalidate()

	l.stateChange <- &stateChangeEvent{
		hash: BuildPackageHash(p.AbsPath),
		lc:   lc,
	}
}

// Close stops the loader processing
func (l *loader) Close() error {
	l.closer <- true
	return nil
}

// Errors exposes problems with code found during compilation on a file-by-file
// basis.
func (l *loader) Errors(lc LoaderContext, handleErrs func(file string, errs []FileError)) {
	l.caravan.Iter(func(key collections.Hash, node *collections.Node) bool {
		p := node.Element.(*Package)
		dp, ok := p.distincts[lc.GetDistinctHash()]
		if !ok {
			return true
		}
		for fname, f := range dp.files {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.AbsPath, fname), f.errs)
			}
		}
		for fname, f := range dp.testFiles {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(p.AbsPath, fname), f.errs)
			}
		}
		return true
	})
}

// LoadDirectory adds the contents of a directory to the Loader
func (l *loader) LoadDirectory(lc LoaderContext, path string) error {
	fmt.Printf("loader::LoadDirectory: entered\n")
	if !lc.IsDir(path) {
		return fmt.Errorf("Argument '%s' is not a directory", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("Could not get absolute path for '%s'", absPath)
	}

	fmt.Printf("loader::LoadDirectory: reading dir '%s'\n", absPath)
	l.readDir(lc, absPath)

	fmt.Printf("loader::LoadDirectory: done\n")
	return nil
}

func (l *loader) readDir(lc LoaderContext, absPath string) {
	if !lc.IsAllowed(absPath) {
		l.Log.Verbosef("readDir: directory '%s' is not allowed\n", absPath)
		return
	}

	l.Log.Debugf("readDir: queueing '%s'...\n", absPath)

	l.ensurePackage(lc, absPath)

	fis, err := lc.ReadDir(absPath)
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

func (l *loader) processStateChange(sce *stateChangeEvent) {
	n, _ := l.caravan.Find(sce.hash)
	p := n.Element.(*Package)
	dp := p.distincts[sce.lc.GetDistinctHash()]
	// fmt.Printf("Processing %s::%s\n", p, dp)

	loadState := dp.loadState.get()

	l.Log.Debugf("PSC: %s: current state: %d\n", p, loadState)

	switch loadState {
	case queued:
		l.processDirectory(sce.lc, p, dp)

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case unloaded:
		haveGo := l.processGoFiles(sce.lc, p, dp)
		haveCgo := l.processCgoFiles(sce.lc, p, dp)
		if (haveGo || haveCgo) && dp.buildPkg != nil {
			imports := importPathMapToArray(dp.importPaths)
			l.processPackages(sce.lc, p, imports, false)
			l.processComplete(sce.lc, p)
		}

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case loadedGo:
		haveTestGo := l.processTestGoFiles(sce.lc, p, dp)
		if haveTestGo && dp.buildPkg != nil {
			imports := importPathMapToArray(dp.testImportPaths)
			l.processPackages(sce.lc, p, imports, true)
			l.processComplete(sce.lc, p)
		}

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case loadedTest:
		// Short circuiting directly to next state.  Will add external test
		// packages later.

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case done:
		fmt.Printf("Completed %s\n", p.AbsPath)
		p.m.Lock()
		for lc := range p.loaderContexts {
			complete := lc.AreAllPackagesComplete()
			if complete {
				l.Log.Debugf("All packages are loaded\n")
				lc.Signal()
			}
		}
		p.m.Unlock()
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

func (l *loader) processComplete(lc LoaderContext, p *Package) {
	if lc.IsUnsafe(p) {
		l.Log.Debugf(" PC: %s: Checking unsafe (skipping)\n", p)
		return
	}

	err := lc.CheckPackage(p)

	if err != nil {
		l.Log.Debugf("Error while checking %s:\n\t%s\n\n", p.AbsPath, err.Error())
	}
}

func (l *loader) processDirectory(lc LoaderContext, p *Package, dp *DistinctPackage) {
	if l.processUnsafe(lc, p) {
		return
	}

	dp.buildPkg = lc.ImportBuildPackage(p)
}

func (l *loader) getFileReader(lc LoaderContext, absFilepath string) (io.Reader, bool) {
	var r io.Reader
	if of, err := l.openedFiles.Get(absFilepath); err != nil {
		r = lc.OpenFile(absFilepath)
		if r == nil {
			return nil, false
		}
	} else {
		r = of.NewReader()
	}

	return r, true
}

func (l *loader) processGoFiles(lc LoaderContext, p *Package, dp *DistinctPackage) bool {
	if lc.IsUnsafe(p) {
		return true
	}

	if dp.buildPkg == nil {
		return false
	}

	fnames := dp.buildPkg.GoFiles
	if len(fnames) == 0 {
		return false
	}

	for _, fname := range fnames {
		fpath := filepath.Join(p.AbsPath, fname)
		r, ok := l.getFileReader(lc, fpath)
		if !ok {
			continue
		}

		hash := calculateHash(r)
		if s, ok := r.(io.Seeker); ok {
			s.Seek(0, io.SeekStart)
		} else {
			r, _ = l.getFileReader(lc, fpath)
		}

		astf, err := parser.ParseFile(p.Fset, fpath, r, parser.AllErrors)

		if c, ok := r.(io.Closer); ok {
			c.Close()
		}

		if err != nil {
			l.Log.Debugf(" GF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
		}

		l.processAstFile(astf, dp.importPaths)

		p.fileHashes[fname] = hash

		files := dp.currentFiles()
		files[fname] = &File{
			errs: []FileError{},
			file: astf,
		}
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

func (l *loader) processCgoFiles(lc LoaderContext, p *Package, dp *DistinctPackage) bool {
	if lc.IsUnsafe(p) {
		return true
	}

	if dp.buildPkg == nil {
		return false
	}

	fnames := dp.buildPkg.CgoFiles
	if len(fnames) == 0 {
		return false
	}

	cgoCPPFLAGS, _, _, _ := cflags(dp.buildPkg, true)
	_, cgoexeCFLAGS, _, _ := cflags(dp.buildPkg, false)

	if len(dp.buildPkg.CgoPkgConfig) > 0 {
		pcCFLAGS, err := pkgConfigFlags(dp.buildPkg)
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
	if dp.buildPkg.Goroot && dp.buildPkg.ImportPath == "runtime/cgo" {
		cgoflags = append(cgoflags, "-import_runtime_cgo=false")
	}
	if dp.buildPkg.Goroot && dp.buildPkg.ImportPath == "runtime/race" || dp.buildPkg.ImportPath == "runtime/cgo" {
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

		hash := calculateHash(f)
		fmt.Printf("Generated hash 0x%x for '%s'\n", hash, fname)

		f.Seek(0, io.SeekStart)
		astf, err := parser.ParseFile(p.Fset, displayFiles[i], f, 0)

		f.Close()

		if err != nil {
			l.Log.Debugf("CGO: %s: ERROR: Failed to parse %s\n\t%s\n", p, fname, err.Error())
		}

		l.processAstFile(astf, dp.importPaths)

		p.fileHashes[fname] = hash

		files := dp.currentFiles()
		files[fname] = &File{
			errs: []FileError{},
			file: astf,
		}
	}
	l.Log.Debugf("CGO: %s: Done processing\n", p)

	return true
}

func (l *loader) processTestGoFiles(lc LoaderContext, p *Package, dp *DistinctPackage) bool {
	if lc.IsUnsafe(p) || dp.buildPkg == nil {
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

	fnames := dp.buildPkg.TestGoFiles
	if len(fnames) == 0 {
		// No test files; continue on.
		return false
	}

	l.Log.Debugf("TFG: %s: processing %d test Go files\n", p, len(fnames))
	for _, fname := range fnames {
		fpath := filepath.Join(p.AbsPath, fname)
		r, ok := l.getFileReader(lc, fpath)
		if !ok {
			continue
		}

		hash := calculateHash(r)
		fmt.Printf("Generated hash 0x%x for '%s'\n", hash, fpath)

		astf, err := parser.ParseFile(p.Fset, fpath, r, parser.AllErrors)

		if c, ok := r.(io.Closer); ok {
			c.Close()
		}

		if err != nil {
			l.Log.Debugf("TGF: ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
		}

		l.processAstFile(astf, dp.testImportPaths)

		p.fileHashes[fname] = hash

		files := dp.currentFiles()
		files[fname] = &File{
			errs: []FileError{},
			file: astf,
		}
	}

	l.Log.Debugf("TFG: %s: processing complete\n", p)
	return true
}

func (l *loader) processAstFile(astf *ast.File, importPaths map[string]bool) {
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
}

func (l *loader) processUnsafe(lc LoaderContext, p *Package) bool {
	if !lc.IsUnsafe(p) {
		return false
	}
	l.Log.Debugf("*** Loading `%s`, replacing with types.Unsafe\n", p)
	dp := p.distincts[lc.GetDistinctHash()]
	dp.typesPkg = types.Unsafe

	l.caravan.Insert(p)

	return true
}

func (l *loader) processPackages(lc LoaderContext, p *Package, importPaths []string, testing bool) {
	dp := p.distincts[lc.GetDistinctHash()]
	loadState := dp.loadState.get()
	l.Log.Debugf(" PP: %s: %d: started\n", p, loadState)

	importedPackages := map[string]bool{}

	for _, importPath := range importPaths {
		targetPath, err := lc.FindImportPath(p, importPath)
		if err != nil {
			l.Log.Debugf(" PP: %s: %d: %s\n\t%s\n", p, loadState, err.Error())
			continue
		}
		l.ensurePackage(lc, targetPath)

		importedPackages[targetPath] = true
	}

	// TEMPORARY
	func() {
		imprts := []string{}
		for importedPackage := range importedPackages {
			n, ok := l.caravan.Find(BuildPackageHash(importedPackage))
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
		n, ok := l.caravan.Find(BuildPackageHash(importPath))
		if !ok {
			l.Log.Debugf(" PP: %s: %d: import path is missing: %s\n", p, loadState, importPath)
			continue
		}
		targetP := n.Element.(*Package)
		targetDp := targetP.distincts[lc.GetDistinctHash()]

		targetDp.m.Lock()
		for !targetDp.CheckReady(loadState) {
			l.Log.Debugf(" PP: %s: %d: *** still waiting on %s ***\n", p, loadState, targetP)
			targetDp.c.Wait()
		}
		targetDp.m.Unlock()

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

func (l *loader) ensurePackage(lc LoaderContext, absPath string) *Package {
	p, _, created := lc.EnsurePackage(absPath)

	if created {
		l.stateChange <- &stateChangeEvent{
			lc:   lc,
			hash: p.Hash(),
		}
	}

	return p
}
