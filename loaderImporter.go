package langd

import (
	"fmt"
	"go/types"

	"github.com/pkg/errors"
)

type loaderImporter struct {
	l *Loader
}

// Import is the implementation of types.Importer
func (li *loaderImporter) Import(path string) (*types.Package, error) {
	dp, err := li.locatePackages(path)
	if err != nil {
		return nil, err
	}
	if dp == nil {
		return nil, fmt.Errorf("Path parsed, but does not contain package %s", path)
	}

	return dp.typesPkg, nil
}

// ImportFrom is the implementation of types.ImporterFrom
func (li *loaderImporter) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	absPath, err := li.l.findImportPath(path, srcDir)
	if err != nil {
		msg := fmt.Sprintf("Failed to locate import path for %s, %s", path, srcDir)
		return nil, errors.Wrap(err, msg)
	}

	dp, err := li.locatePackages(absPath)
	if err != nil {
		msg := fmt.Sprintf("Failed to locate package %s\n\tfrom %s, %s", absPath, path, srcDir)
		return nil, errors.Wrap(err, msg)
	}

	if dp.typesPkg == nil {
		return nil, fmt.Errorf("Got nil in packages map")
	}

	return dp.typesPkg, nil
}

func (li *loaderImporter) locatePackages(path string) (*DistinctPackage, error) {
	phash := calculateHashFromString(path)
	chash := combineHashes(phash, li.l.hash)
	n, ok := li.l.le.caravan.Find(chash)
	if !ok {
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	dp := n.Element.(*DistinctPackage)
	return dp, nil
}
