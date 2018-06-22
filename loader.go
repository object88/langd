package langd

import (
	"fmt"
	"go/ast"
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

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
)

type stateChangeEvent struct {
	lc   LoaderContext
	hash collections.Hash
}

// Loader is a Go code loader
type Loader interface {
	io.Closer

	EnsurePackage(absPath string) (*Package, bool)
	Errors(lc LoaderContext, handleErrs func(file string, errs []FileError))
	LoadDirectory(lc LoaderContext, path string) error
	InvalidatePackage(absPath string)

	Caravan() *collections.Caravan
	OpenedFiles() *OpenedFiles
}

type loader struct {
	closer chan bool

	caravan     *collections.Caravan
	openedFiles *OpenedFiles
	packages    map[string]*Package

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
		packages:    map[string]*Package{},
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

func (l *loader) InvalidatePackage(absPath string) {
	p, ok := l.packages[absPath]
	if !ok {
		fmt.Printf("No package at %s\n", absPath)
		return
	}

	nodesMap := map[*collections.Node]bool{}
	for lc := range p.loaderContexts {
		n, _ := l.caravan.Find(lc.CalculateDistinctPackageHash(absPath))
		nodesMap[n] = true
	}

	nodes := make([]*collections.Node, len(nodesMap))
	i := 0
	for n := range nodesMap {
		nodes[i] = n
		i++
	}

	dps := flattenAscendants(true, nodes...)

	fmt.Printf("Have %d distinct packages\n", len(dps))

	for _, dp := range dps {
		fmt.Printf("Invalidating %s\n", dp)
		dp.Invalidate()
		lc := dp.lc

		l.stateChange <- &stateChangeEvent{
			hash: dp.Hash(),
			lc:   lc,
		}
	}
}

// Close stops the loader processing
func (l *loader) Close() error {
	l.closer <- true
	return nil
}

// EnsurePackage will check for a package at the given path, and if one does
// not exist, create it.
func (l *loader) EnsurePackage(absPath string) (*Package, bool) {
	p, ok := l.packages[absPath]
	if !ok {
		fmt.Printf("EnsurePackage: Creating new package for %s\n", absPath)
		p = NewPackage(absPath)
		l.packages[absPath] = p
	}
	return p, !ok
}

// Errors exposes problems with code found during compilation on a file-by-file
// basis.
func (l *loader) Errors(lc LoaderContext, handleErrs func(file string, errs []FileError)) {
	l.caravan.Iter(func(key collections.Hash, node *collections.Node) bool {
		dp := node.Element.(*DistinctPackage)
		if dp.lc != lc {
			return true
		}
		for fname, f := range dp.files {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(dp.Package.AbsPath, fname), f.errs)
			}
		}
		for fname, f := range dp.testFiles {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(dp.Package.AbsPath, fname), f.errs)
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

	l.ensureDistinctPackage(lc, absPath)

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
	dp := n.Element.(*DistinctPackage)

	loadState := dp.loadState.get()

	l.Log.Debugf("PSC: %s: current state: %d\n", dp, loadState)

	switch loadState {
	case queued:
		l.processDirectory(sce.lc, dp)

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case unloaded:
		haveGo := l.processGoFiles(sce.lc, dp)
		haveCgo := l.processCgoFiles(sce.lc, dp)
		if (haveGo || haveCgo) && dp.buildPkg != nil {
			imports := importPathMapToArray(dp.importPaths)
			l.processPackages(sce.lc, dp, imports, false)
			l.processComplete(sce.lc, dp)
		}

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case loadedGo:
		haveTestGo := l.processTestGoFiles(sce.lc, dp)
		if haveTestGo && dp.buildPkg != nil {
			imports := importPathMapToArray(dp.testImportPaths)
			l.processPackages(sce.lc, dp, imports, true)
			l.processComplete(sce.lc, dp)
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
		fmt.Printf("Completed %s\n", dp.Package.AbsPath)
		dp.m.Lock()
		for lc := range dp.Package.loaderContexts {
			complete := lc.AreAllPackagesComplete()
			if complete {
				l.Log.Debugf("All packages are loaded\n")
				lc.Signal()
			}
		}
		dp.m.Unlock()
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

func (l *loader) processComplete(lc LoaderContext, dp *DistinctPackage) {
	if lc.IsUnsafe(dp) {
		l.Log.Debugf(" PC: %s: Checking unsafe (skipping)\n", dp)
		return
	}

	err := lc.CheckPackage(dp)

	if err != nil {
		l.Log.Debugf("Error while checking %s:\n\t%s\n\n", dp.Package.AbsPath, err.Error())
	}
}

func (l *loader) processDirectory(lc LoaderContext, dp *DistinctPackage) {
	if lc.IsUnsafe(dp) {
		l.Log.Debugf("*** Loading `%s`, replacing with types.Unsafe\n", dp)
		dp.typesPkg = types.Unsafe

		l.caravan.Insert(dp)
	} else {
		lc.ImportBuildPackage(dp)
	}
}

func (l *loader) processGoFiles(lc LoaderContext, dp *DistinctPackage) bool {
	if lc.IsUnsafe(dp) {
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
		fpath := filepath.Join(dp.Package.AbsPath, fname)
		l.processFile(lc, dp, fname, fpath, fpath, dp.importPaths)
	}

	return true
}

func (l *loader) processCgoFiles(lc LoaderContext, dp *DistinctPackage) bool {
	if lc.IsUnsafe(dp) {
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
			l.Log.Debugf("CGO: %s: Failed to get flags: %s\n", dp, err.Error())
			return false
		}
		cgoCPPFLAGS = append(cgoCPPFLAGS, pcCFLAGS...)
	}

	fpaths := make([]string, len(fnames))
	for k, v := range fnames {
		fpaths[k] = filepath.Join(dp.Package.AbsPath, v)
	}

	tmpdir, _ := ioutil.TempDir("", strings.Replace(dp.Package.AbsPath, "/", "_", -1)+"_C")
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
	cmd.Dir = dp.Package.AbsPath
	cmd.Stdout = os.Stdout // os.Stderr
	cmd.Stderr = os.Stdout // os.Stderr
	if err := cmd.Run(); err != nil {
		l.Log.Debugf("CGO: %s: ERROR: cgo failed: %s\n\t%s\n", dp, args, err.Error())
		return false
	}

	for i, fpath := range files {
		fname := filepath.Base(fpath)
		l.processFile(lc, dp, fname, fpath, displayFiles[i], dp.importPaths)
	}
	l.Log.Debugf("CGO: %s: Done processing\n", dp)

	return true
}

func (l *loader) processTestGoFiles(lc LoaderContext, dp *DistinctPackage) bool {
	if lc.IsUnsafe(dp) || dp.buildPkg == nil {
		return false
	}

	// If we are in vendor; exclude test files.  It is possible for imports to
	// contain references for packages which are not available.  May want to
	// revisit this later; loading as much as possible for completion sake,
	// but not reporting them as complete errors.
	for _, part := range strings.Split(dp.Package.AbsPath, string(filepath.Separator)) {
		if part == "vendor" {
			return false
		}
	}

	fnames := dp.buildPkg.TestGoFiles
	if len(fnames) == 0 {
		// No test files; continue on.
		return false
	}

	l.Log.Debugf("TFG: %s: processing %d test Go files\n", dp, len(fnames))
	for _, fname := range fnames {
		fpath := filepath.Join(dp.Package.AbsPath, fname)
		l.processFile(lc, dp, fname, fpath, fpath, dp.testImportPaths)
	}

	l.Log.Debugf("TFG: %s: processing complete\n", dp)
	return true
}

func (l *loader) processFile(lc LoaderContext, dp *DistinctPackage, fname, fpath, displayPath string, importPaths map[string]bool) {
	r, ok := l.getFileReader(lc, fpath)
	if !ok {
		return
	}

	hash := calculateHash(r)
	if s, ok := r.(io.Seeker); ok {
		s.Seek(0, io.SeekStart)
	} else {
		if c, ok := r.(io.Closer); ok {
			c.Close()
		}
		r, _ = l.getFileReader(lc, fpath)
	}

	astf, err := parser.ParseFile(dp.Package.Fset, displayPath, r, parser.AllErrors)

	if c, ok := r.(io.Closer); ok {
		c.Close()
	}

	if err != nil {
		l.Log.Debugf("ERROR: While parsing %s:\n\t%s\n", fpath, err.Error())
	}

	l.findImportPathsFromAst(astf, importPaths)

	dp.Package.m.Lock()
	dp.Package.fileHashes[fname] = hash
	dp.Package.m.Unlock()

	files := dp.currentFiles()
	files[fname] = &File{
		errs: []FileError{},
		file: astf,
	}
}

func (l *loader) processPackages(lc LoaderContext, dp *DistinctPackage, importPaths []string, testing bool) {
	loadState := dp.loadState.get()
	l.Log.Debugf(" PP: %s: %d: started\n", dp, loadState)

	importedPackages := map[string]bool{}

	for _, importPath := range importPaths {
		targetPath, err := lc.FindImportPath(dp, importPath)
		if err != nil {
			l.Log.Debugf(" PP: %s: %d: %s\n\t%s\n", dp, loadState, err.Error())
			continue
		}
		l.ensureDistinctPackage(lc, targetPath)

		importedPackages[targetPath] = true
	}

	// TEMPORARY
	func() {
		imprts := []string{}
		for importedPackage := range importedPackages {
			if targetDp, err := lc.FindDistinctPackage(importedPackage); err != nil {
				continue
			} else {
				imprts = append(imprts, targetDp.String())
			}
		}
		allImprts := strings.Join(imprts, ", ")
		l.Log.Debugf(" PP: %s: %d: -> %s\n", dp, loadState, allImprts)
	}()

	for importPath := range importedPackages {
		targetDp, err := lc.FindDistinctPackage(importPath)
		if err != nil {
			l.Log.Debugf(" PP: %s: %d: import path is missing: %s\n", dp, loadState, importPath)
			continue
		}

		targetDp.WaitUntilReady(loadState)

		if testing {
			err = l.caravan.WeakConnect(dp, targetDp)
		} else {
			err = l.caravan.Connect(dp, targetDp)
		}

		if err != nil {
			panic(fmt.Sprintf(" PP: %s: %d: [weak] connect failed:\n\tfrom: %s\n\tto: %s\n\terr: %s\n\n", dp, loadState, dp, targetDp, err.Error()))
		}
	}
	// All dependencies are loaded; can proceed.
	l.Log.Debugf(" PP: %s: %d: all imports fulfilled.\n", dp, loadState)
}

func (l *loader) ensureDistinctPackage(lc LoaderContext, absPath string) {
	dp, created := lc.EnsureDistinctPackage(absPath)

	if created {
		l.stateChange <- &stateChangeEvent{
			lc:   lc,
			hash: dp.Hash(),
		}
	}
}

func (l *loader) findImportPathsFromAst(astf *ast.File, importPaths map[string]bool) {
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
