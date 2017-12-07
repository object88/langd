package langd

import (
	"fmt"
	"go/types"
	"path/filepath"
)

// Importer implements types.Importer for use in Loader
type Importer struct {
	l *Loader
}

// Import is the implementation of types.Importer
func (i *Importer) Import(path string) (*types.Package, error) {
	// fmt.Printf("Importer looking for '%s'\n", path)
	ps, err := i.locatePackages(path)
	if err != nil {
		return nil, err
	}
	// d, ok := i.l.directories[path]
	// if !ok {
	// 	fmt.Printf("Not found!\n")
	// 	i.l.processDirectory(&importDirective{
	// 		absPath: path,
	// 	}, -1)
	// 	d, ok = i.l.directories[path]
	// 	if !ok {
	// 		fmt.Printf("Failed to import '%s'\n", path)
	// 		return nil, fmt.Errorf("Failed to import %s", path)
	// 	}
	// }
	pkgName := filepath.Base(path)
	// p, ok := d.packages[pkgName]
	p, ok := ps[pkgName]
	if !ok {
		return nil, fmt.Errorf("Directory parsed, but does not contain package %s", pkgName)
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

	ps, err := i.locatePackages(absPath)
	if err != nil {
		fmt.Printf("Failed to locate package %s\n\tfrom %s, %s:\n\t%s\n", absPath, path, srcDir, err.Error())
		return nil, err
	}

	base := filepath.Base(path)
	p, ok := ps[base]
	if !ok {
		fmt.Printf("Failed to find package from %s, %s", path, srcDir)
		return nil, fmt.Errorf("Failed to find package from %s, %s", path, srcDir)
	}

	if p.typesPkg == nil {
		fmt.Printf("Found package map; has nil for %s\nAt %s\nStarting from %s, %s.\n", base, absPath, path, srcDir)
		fmt.Printf("Have...\n")
		for _, v := range ps {
			fmt.Printf("\t%s\n", v.name)
		}
		return nil, fmt.Errorf("Got nil in packages map")
	}
	return p.typesPkg, nil
	// return i.Import(absPath)
}

func (i *Importer) locatePackages(path string) (map[string]*Package, error) {
	d, ok := i.l.directories[path]
	if !ok {
		fmt.Printf("**** Not found! *****\n")
		// i.l.processDirectory(&importDirective{
		// 	absPath: path,
		// }, -1)
		// d, ok = i.l.directories[path]
		// if !ok {
		// 	fmt.Printf("Failed to import '%s'\n", path)
		// 	return nil, fmt.Errorf("Failed to import %s", path)
		// }
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	return d.packages, nil
}
