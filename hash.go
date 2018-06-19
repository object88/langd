package langd

import (
	"io"

	"github.com/OneOfOne/xxhash"
	"github.com/object88/langd/collections"
)

func calculateHash(r io.Reader) collections.Hash {
	h := xxhash.New64()
	in := make([]byte, 1024)
	for {
		c, err := r.Read(in)
		if err == io.EOF {
			break
		}
		h.Write(in[0:c])
	}
	hash := h.Sum64()

	return collections.Hash(hash)
}

func calculateHashFromString(s string) collections.Hash {
	h := xxhash.New64()
	h.WriteString(s)
	hash := h.Sum64()

	return collections.Hash(hash)
}

func calculateHashFromStrings(s ...string) collections.Hash {
	h := xxhash.New64()
	for _, s1 := range s {
		h.WriteString(s1)
	}
	hash := h.Sum64()

	return collections.Hash(hash)
}
