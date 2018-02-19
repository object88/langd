package langd

import (
	"os"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

var hugoIsAccessible = false

func init() {
	fi, err := os.Stat("../../gohugoio/hugo")
	if err != nil {
		return
	}

	if !fi.IsDir() {
		return
	}

	hugoIsAccessible = true
}

func Test_Load_Own_Package(t *testing.T) {
	src := `package bar
	import "../bar"
	var Bar int = 0`

	packages := map[string]map[string]string{
		"bar": map[string]string{
			"bar.go": src,
		},
	}

	fc := buildutil.FakeContext(packages)
	loader := NewLoader(func(l *Loader) {
		l.context = fc
	})
	w := CreateWorkspace(loader, log.CreateLog(os.Stdout))
	w.log.SetLevel(log.Verbose)

	done := loader.Start()
	err := loader.LoadDirectory("/go/src/bar")
	if err != nil {
		t.Fatalf("Error while loading: %s", err.Error())
	}
	<-done
}

func Test_Load_Relative_Path(t *testing.T) {
	if !hugoIsAccessible {
		t.Skip("Test requires the Hugo repository; skipping.")
	}

	loader := NewLoader()

	done := loader.Start()
	err := loader.LoadDirectory("../../gohugoio/hugo")
	if err != nil {
		t.Fatalf("Error while loading: %s", err.Error())
	}

	<-done
}
