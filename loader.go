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
	"strconv"
	"strings"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
)

type stateChangeEvent struct {
	lc   *LoaderContext
	hash collections.Hash
}

// Loader is a Go code loader
type Loader struct {
	closer chan bool

	caravan     *collections.Caravan
	openedFiles *OpenedFiles
	packages    map[string]*Package

	stateChange chan *stateChangeEvent

	Log *log.Log
}

// NewLoader creates a new loader
func NewLoader() *Loader {
	l := &Loader{
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

func (l *Loader) InvalidatePackage(absPath string) {
	p, ok := l.packages[absPath]
	if !ok {
		fmt.Printf("No package at %s\n", absPath)
		return
	}

	nodesMap := map[*collections.Node]bool{}
	for lc := range p.loaderContexts {
		n, _ := l.caravan.Find(lc.calculateDistinctPackageHash(absPath))
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
func (l *Loader) Close() error {
	l.closer <- true
	return nil
}

// ensurePackage will check for a package at the given path, and if one does
// not exist, create it.
func (l *Loader) ensurePackage(absPath string) (*Package, bool) {
	p, ok := l.packages[absPath]
	if !ok {
		p = NewPackage(absPath)
		l.packages[absPath] = p
	}
	return p, !ok
}

func (l *Loader) readDir(lc *LoaderContext, absPath string) {
	if !lc.isAllowed(absPath) {
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

func (l *Loader) processStateChange(sce *stateChangeEvent) {
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
		importPaths := map[string]bool{}
		haveGo := l.processGoFiles(sce.lc, dp, importPaths)
		haveCgo := l.processCgoFiles(sce.lc, dp, importPaths)
		if haveGo || haveCgo {
			imports := importPathMapToArray(importPaths)
			l.processPackages(sce.lc, dp, imports, false)
			l.processComplete(sce.lc, dp)
		}

		dp.loadState.increment()
		dp.c.Broadcast()
		l.stateChange <- sce
	case loadedGo:
		importPaths := map[string]bool{}
		haveTestGo := l.processTestGoFiles(sce.lc, dp, importPaths)
		if haveTestGo {
			imports := importPathMapToArray(importPaths)
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
		if sce.lc.areAllPackagesComplete() {
			l.Log.Debugf("All packages are loaded\n")
			sce.lc.Signal()
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

func (l *Loader) processComplete(lc *LoaderContext, dp *DistinctPackage) {
	if lc.isUnsafe(dp) {
		l.Log.Debugf(" PC: %s: Checking unsafe (skipping)\n", dp)
		return
	}

	err := lc.checkPackage(dp)
	if err != nil {
		l.Log.Debugf(" PC: %s: Error while checking %s:\n\t%s\n\n", dp, dp.Package.AbsPath, err.Error())
	}
}

func (l *Loader) processDirectory(lc *LoaderContext, dp *DistinctPackage) {
	if lc.isUnsafe(dp) {
		l.Log.Debugf("*** Loading `%s`, replacing with types.Unsafe\n", dp)
		dp.typesPkg = types.Unsafe

		l.caravan.Insert(dp)
	} else {
		err := dp.generateBuildPackage()
		if err != nil {
			l.Log.Debugf("importBuildPackage: %s\n", err.Error())
		}
	}
}

func (l *Loader) processGoFiles(lc *LoaderContext, dp *DistinctPackage, importPaths map[string]bool) bool {
	if lc.isUnsafe(dp) || dp.buildPkg == nil {
		return false
	}

	fnames := dp.buildPkg.GoFiles
	if len(fnames) == 0 {
		return false
	}

	for _, fname := range fnames {
		fpath := filepath.Join(dp.Package.AbsPath, fname)
		l.processFile(lc, dp, fname, fpath, fpath, importPaths)
	}

	return true
}

func (l *Loader) processCgoFiles(lc *LoaderContext, dp *DistinctPackage, importPaths map[string]bool) bool {
	if lc.isUnsafe(dp) || dp.buildPkg == nil {
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
		l.processFile(lc, dp, fname, fpath, displayFiles[i], importPaths)
	}
	l.Log.Debugf("CGO: %s: Done processing\n", dp)

	return true
}

func (l *Loader) processTestGoFiles(lc *LoaderContext, dp *DistinctPackage, importPaths map[string]bool) bool {
	if lc.isUnsafe(dp) || dp.buildPkg == nil {
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
		l.processFile(lc, dp, fname, fpath, fpath, importPaths)
	}

	l.Log.Debugf("TFG: %s: processing complete\n", dp)
	return true
}

func (l *Loader) processFile(lc *LoaderContext, dp *DistinctPackage, fname, fpath, displayPath string, importPaths map[string]bool) {
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

	dp.files[fname] = &File{
		errs: []FileError{},
		file: astf,
	}
}

func (l *Loader) processPackages(lc *LoaderContext, dp *DistinctPackage, importPaths []string, testing bool) {
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

func (l *Loader) ensureDistinctPackage(lc *LoaderContext, absPath string) {
	dp, created := lc.ensureDistinctPackage(absPath)

	if created {
		l.stateChange <- &stateChangeEvent{
			lc:   lc,
			hash: dp.Hash(),
		}
	}
}

func (l *Loader) findImportPathsFromAst(astf *ast.File, importPaths map[string]bool) {
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

func (l *Loader) getFileReader(lc *LoaderContext, absFilepath string) (io.Reader, bool) {
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
