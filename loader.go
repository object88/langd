package langd

import (
	"fmt"
	"go/build"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
)

type importDir struct {
	path string
}

// Loader will load code into an AST
type Loader struct {
	config  *types.Config
	srcDirs []string
	stderr  *log.Log

	fset  *token.FileSet
	packs *collections.Caravan

	dirQueue *collections.InfiniteQueue

	m           sync.Mutex
	directories map[string]*Directory

	done chan bool
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

	return &Loader{
		config:      config,
		srcDirs:     srcDirs,
		stderr:      l,
		fset:        token.NewFileSet(),
		packs:       collections.CreateCaravan(),
		dirQueue:    collections.CreateInfiniteQueue(),
		directories: map[string]*Directory{},
		done:        make(chan bool),
	}
}

// LoadDirectory reads in the file of a given directory.  LoadDirectory will
// not read directories that begin with a "." (i.e., ".git"), and it will not
// follow symbolic links.
func (l *Loader) LoadDirectory(dpath string) error {
	prefix := ""
	for _, v := range l.srcDirs {
		if strings.HasPrefix(dpath, v) {
			prefix = dpath[len(v)+1:]
		}
	}
	if prefix == "" {
		return fmt.Errorf("Failed to find '%s'", dpath)
	}

	filepath.Walk(dpath, func(dpath string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			return nil
		}

		// Skipping directories that start with "." (i.e., .git)
		if strings.HasPrefix(filepath.Base(info.Name()), ".") {
			return filepath.SkipDir
		}

		// dpath: /Users/bropa18/work/src/github.com/object88/langd/examples/echo
		l.dirQueue.In() <- &importDir{
			path: dpath,
		}

		return nil
	})

	return nil
}

// Start initializes the dispatcher for file and directory load events.  The
// dispatch is stopped by passing a bool (any value) into the returned
// channel.
func (l *Loader) Start(base string) (<-chan bool, error) {
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

	done := make(chan bool)
	importsDone := make(chan bool)

	go func() {
		for {
			select {
			case _, ok := <-l.done:
				if !ok {
					return
				}

			case pimportDir := <-l.dirQueue.Out():
				imp := pimportDir.(*importDir)
				l.processDir(imp)

			case <-importsDone:
				fmt.Printf("*** Reported imports done...\n")
				ready := true
				l.packs.Walk(collections.WalkDown, func(k collections.Keyer, _, _ bool) {
					if k.(*Package).astPkg == nil {
						ready = false
					}
				})
				if ready {
					fmt.Printf("*** *** Completely done\n")
					done <- true
				}
			}
		}
	}()

	return done, nil
}

// Close will stop monitoring the files
func (l *Loader) Close() {
	close(l.done)
}

func (l *Loader) processDir(imp *importDir) {
	l.m.Lock()

	absPath := findPackagePath(".", imp.path)

	if _, ok := l.directories[absPath]; ok {
		l.m.Unlock()
		return
	}

	d := CreateDirectory(absPath)
	l.directories[absPath] = d

	l.m.Unlock()

	go d.Scan(l.fset, l.dirQueue.In())
}

func findPackagePath(path, src string) string {
	buildPkg, err := build.Import(path, src, build.FindOnly)
	if err != nil {
		fmt.Printf("Oh dear:\n\t%s\n", err.Error())
	}
	if buildPkg.Dir == "" {
		// If Dir is the empty string, this is a stdlib package?
		return path
	}
	return buildPkg.Dir
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
