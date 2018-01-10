package langd

import (
	"fmt"
	"go/types"

	"github.com/object88/langd/collections"
)

// Importer implements types.Importer for use in Loader
type Importer struct {
	l *Loader
}

// Import is the implementation of types.Importer
func (i *Importer) Import(path string) (*types.Package, error) {
	// fmt.Printf("Importer looking for '%s'\n", path)
	p, err := i.locatePackages(path)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("Directory parsed, but does not contain package %s", path)
	}

	return p.typesPkg, nil
}

// ImportFrom is the implementation of types.ImporterFrom
func (i *Importer) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	absPath, err := i.l.findImportPath(path, srcDir)
	if err != nil {
		fmt.Printf("Failed to locate import path for %s, %s:\n\t%s", path, srcDir, err.Error())
		return nil, fmt.Errorf("Failed to locate import path for %s, %s:\n\t%s", path, srcDir, err.Error())
	}

	p, err := i.locatePackages(absPath)
	if err != nil {
		fmt.Printf("Failed to locate package %s\n\tfrom %s, %s:\n\t%s\n", absPath, path, srcDir, err.Error())
		return nil, err
	}

	if p.typesPkg == nil {
		fmt.Printf("\t%s (nil)\n", p.name)
		return nil, fmt.Errorf("Got nil in packages map")
	}

	return p.typesPkg, nil
}

func (i *Importer) locatePackages(path string) (*Package, error) {
	i.l.caravanMutex.Lock()
	n, ok := i.l.caravan.Find(collections.Key(path))
	i.l.caravanMutex.Unlock()
	if !ok {
		fmt.Printf("**** Not found! *****\n")
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	p := n.Element.(*Package)

	return p, nil
}
