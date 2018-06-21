package langd

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/object88/langd/collections"
)

func Test_CalculateHash(t *testing.T) {
	source := make([]byte, 2056)
	rand.Read(source)

	r := bytes.NewReader(source)
	actual := uint64(calculateHash(r))

	hash := xxhash.New64()
	hash.Write(source[0:1024])
	hash.Write(source[1024:2048])
	hash.Write(source[2048:2056])
	expected := hash.Sum64()

	if actual != expected {
		t.Errorf("Expected hash 0x%x does not match actual hash 0x%x", expected, actual)
	}
}

func Test_CombineHashes(t *testing.T) {
	// Want to ensure that combining hashes is consistent.
	h1 := calculateHashFromString("foo")
	h2 := calculateHashFromString("bar")

	attempt1 := combineHashes(h1, h2)
	attempt2 := combineHashes(h1, h2)
	if attempt1 != attempt2 {
		t.Errorf("Two attempts at combining hashes resulted in different hashes: 0x%x / 0x%x", attempt1, attempt2)
	}
}

func Benchmark_SingleHash(b *testing.B) {
	var h collections.Hash
	for n := 0; n < b.N; n++ {
		h = calculateHashFromStrings("foo", "bar", "baz", "quux")
	}
	_ = fmt.Sprintf("Hash: 0x%x\n", h)
}

func Benchmark_CombineHashes(b *testing.B) {
	var h collections.Hash
	for n := 0; n < b.N; n++ {
		h1 := calculateHashFromStrings("foo", "bar")
		h2 := calculateHashFromStrings("baz", "quux")
		h = combineHashes(h1, h2)
	}
	_ = fmt.Sprintf("Hash: 0x%x\n", h)
}
