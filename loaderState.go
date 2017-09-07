package langd

import (
	"fmt"
	"go/build"
	"go/token"
	"strings"
)

type loaderState struct {
	base    string
	org     string
	orgPath string
	fset    *token.FileSet
	pkgs    map[string]*Package
	spacers *[]string
}

func newLoaderState(pkgName string) *loaderState {
	spacers := make([]string, 8)
	for i := 0; i < 8; i++ {
		spacers[i] = strings.Repeat("  ", i)
	}

	org := ""
	orgPath := ""
	s := strings.Split(pkgName, "/")
	if strings.Index(s[0], ".") != -1 {
		org = fmt.Sprintf("%s/%s", s[0], s[1])
		orgPath = fmt.Sprintf("%s/src/%s/%s", build.Default.GOPATH, s[0], s[1])
	}

	ls := &loaderState{
		pkgName,
		org,
		orgPath,
		token.NewFileSet(),
		map[string]*Package{},
		&spacers,
	}

	return ls
}

func (ls *loaderState) getSpacer(depth int) string {
	initialSize := len(*ls.spacers)
	if depth >= initialSize {
		// Expand the contents
		targetSize := nextPowerOfTwo(depth + 1)
		target := make([]string, targetSize)
		for i := 0; i < initialSize; i++ {
			target[i] = (*ls.spacers)[i]
		}
		for i := initialSize; i < targetSize; i++ {
			target[i] = strings.Repeat("  ", i)
		}

		ls.spacers = &target
	}

	return (*ls.spacers)[depth]
}
