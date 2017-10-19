package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
)

// Loader will load code into an AST
type Loader struct {
	config  *types.Config
	srcDirs []string
	stderr  *log.Log
	ls      *loaderState

	importer types.ImporterFrom

	m sync.Mutex
	// filesInDir  map[string][]string
	directories map[string]*Directory
}

type Directory struct {
	buildPkg *build.Package
	files    []string
}

// NewLoader constructs a new Loader struct
func NewLoader() *Loader {
	l := log.Stderr()
	l.SetLevel(log.Verbose)
	config := &types.Config{
		Error: func(e error) {
			l.Warnf("%s\n", e.Error())
		},
		Importer: importer.Default(),
	}

	srcDirs := build.Default.SrcDirs()

	importer := config.Importer.(types.ImporterFrom)

	return &Loader{
		config:   config,
		srcDirs:  srcDirs,
		stderr:   l,
		importer: importer,
		// filesInDir:  map[string][]string{},
		directories: map[string]*Directory{},
	}
}

// Start initializes the dispatcher for file and directory load events.  The
// dispatch is stopped by passing a bool (any value) into the returned
// channel.
func (l *Loader) Start(base string) (chan<- bool, error) {
	abs, err := validateInitialPath(base)
	if err != nil {
		return nil, err
	}

	pkgName := ""
	for _, v := range l.srcDirs {
		if strings.HasPrefix(abs, v) {
			pkgName = abs[len(v)+1:]
		}
	}
	if pkgName == "" {
		return nil, fmt.Errorf("Failed to find '%s'", base)
	}

	l.ls = newLoaderState(pkgName)

	done := make(chan bool)
	packLoaded := make(chan *Package)

	go func() {
		for {
			select {
			case pkg := <-packLoaded:
				// A pack is loaded.  Check for imports and process.
				fmt.Printf("Got message that %s id ready to check imports\n", pkg.path)
				go l.LoadImports(pkg)

			case fpath := <-l.ls.fileQueue.Out():
				go l.LoadFile(*fpath, packLoaded)

			case <-done:
				close(packLoaded)
				break
			}
		}
	}()

	return done, nil
}

// LoadDirectory reads in the file of a given directory.  If recurse is true,
// it will read in nested directories.  LoadDirectory will not read directories
// that begin with a "." (i.e., ".git"), and it will not follow symbolic
// links.
func (l *Loader) LoadDirectory(dpath string, recurse bool) <-chan bool {
	done := make(chan bool)

	if recurse {
		filepath.Walk(dpath, l.checkAndQueueDirectory)
		fmt.Printf("Walked all file paths\n")
	} else {
		fmt.Printf("Queueing up %s\n", dpath)
		info, err := os.Lstat(dpath)
		if err != nil {
			fmt.Printf("ERR trying to get info on '%s': %s\n", dpath, err.Error())
			l.ls.errs = append(l.ls.errs, err)
		}
		l.checkAndQueueDirectory(dpath, info, nil)
	}
	return done
}

func (l *Loader) checkAndQueueDirectory(dpath string, info os.FileInfo, _ error) error {
	if !info.IsDir() {
		return nil
	}

	// Skipping directories that start with "." (i.e., .git)
	if strings.HasPrefix(filepath.Base(info.Name()), ".") {
		return filepath.SkipDir
	}

	l.m.Lock()

	if _, ok := l.directories[dpath]; ok {
		fmt.Printf("Already processed '%s'\n", dpath)
		l.m.Unlock()
		return nil
	}

	fmt.Printf("Starting to process %s\n", dpath)
	l.directories[dpath] = &Directory{}

	l.m.Unlock()

	go l.processDirectory(dpath)

	return nil
}

func (l *Loader) processDirectory(dpath string) {
	// Do something with the directory
	buildP, err := build.ImportDir(dpath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			fmt.Printf("NO GO CODE: %s\n", dpath)
			return
		}
		l.stderr.Errorf("Got error when attempting import on dir '%s': %s\n", dpath, err.Error())
		l.ls.errs = append(l.ls.errs, err)
		return
	}

	l.queueFilesFromBuildPackage(dpath, buildP)
}

// queueFile assumes that the lock has already been aquired.
func (l *Loader) queueFile(buildP *build.Package, filename string) string {
	fpath := path.Join(buildP.Dir, filename)
	// fmt.Printf("Dir %s, adding %s\n", buildP.Dir, fpath)

	_, ok := l.ls.files[fpath]
	if !ok {
		l.ls.files[fpath] = nil
	}

	if ok {
		return fpath
	}

	l.ls.fileQueue.In() <- &fpath
	return fpath
}

// LoadFile reads in a single file.  Is assumes that this file is new and has
// not already been added to a package.
func (l *Loader) LoadFile(fpath string, done chan<- *Package) {
	fmt.Printf("PROCESSING: %s\n", fpath)

	dpath := filepath.Dir(fpath)
	astf, err := parser.ParseFile(l.ls.fset, fpath, nil, 0)
	if err != nil {
		l.stderr.Verbosef("Got error while parsing file '%s':\n%s\n", fpath, err.Error())
		l.ls.errs = append(l.ls.errs, err)
		return
	}

	l.m.Lock()

	l.ls.files[fpath] = astf

	var pkg *Package
	keyer, found := l.ls.packs.Find(collections.Key(dpath))
	if !found {
		pkg = &Package{
			astPkg: &ast.Package{
				Name:  astf.Name.Name,
				Files: map[string]*ast.File{},
			},
			path: dpath,
		}
		l.ls.packs.Insert(pkg)
	} else {
		pkg = keyer.(*Package)
	}

	pkg.astPkg.Files[fpath] = astf

	// Check to see if this was the last file for this package to need processing
	directory := l.directories[dpath]
	// fmt.Printf("In package %s, checking through %d files\n", dpath, len(files))
	tcount := len(directory.files)
	nrcount := 0
	for _, v := range directory.files {
		if _, ok := pkg.astPkg.Files[v]; !ok {
			nrcount++
		}
	}
	if nrcount == 0 {
		fmt.Printf("COMPLETED %s\n", pkg.path)
		done <- pkg
	} else {
		fmt.Printf("INCOMPLETE: %03d of %03d: %s\n", tcount-nrcount, tcount, pkg.path)
	}

	l.m.Unlock()
}

func scanImports(imports map[string]bool, file *ast.File) {
	for _, decl := range file.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.IMPORT {
			continue
		}

		for _, spec := range decl.Specs {
			spec := spec.(*ast.ImportSpec)

			// NB: do not assume the program is well-formed!
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path == "C" {
				// Ignore the error and skip the C psuedo package
				continue
			}
			imports[path] = true
		}
	}
}

func (l *Loader) LoadImports(pkg *Package) {
	imports := map[string]bool{}

	l.m.Lock()
	for _, astf := range pkg.astPkg.Files {
		scanImports(imports, astf)
	}
	for dpath := range imports {
		l.directories[dpath] = &Directory{}
	}
	l.m.Unlock()

	for dpath := range imports {
		l.m.Lock()
		_, ok := l.directories[dpath]
		l.m.Unlock()

		if ok {
			// This directory has already been scanned.
			continue
		}

		buildP, err := build.Default.Import(dpath, pkg.path, 0)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			continue
		}

		l.queueFilesFromBuildPackage(dpath, buildP)
	}
}

func (l *Loader) queueFilesFromBuildPackage(dpath string, buildP *build.Package) {
	l.m.Lock()

	l.directories[dpath].buildPkg = buildP

	count := len(buildP.GoFiles) + len(buildP.TestGoFiles)
	filesInDir := make([]string, 0, count)

	for _, v := range buildP.GoFiles {
		filesInDir = append(filesInDir, l.queueFile(buildP, v))
	}

	for _, v := range buildP.TestGoFiles {
		filesInDir = append(filesInDir, l.queueFile(buildP, v))
	}

	l.directories[dpath].files = filesInDir

	l.m.Unlock()
}

func validateInitialPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("Provided path '%s' must be a directory", p)
	}

	return abs, nil
}
