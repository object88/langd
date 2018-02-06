package requests

import (
	"github.com/dgryski/go-mph"
)

type requestMap struct {
	t *mph.Table
	k []string
	f []IniterFunc
}

func newRequestMap() *requestMap {
	m := getIniterFuncs()

	keys := make([]string, len(m))
	funcs := make([]IniterFunc, len(m))

	i := 0
	for k, v := range m {
		keys[i] = k
		funcs[i] = v
		i++
	}

	t := mph.New(keys)

	return &requestMap{
		t: t,
		k: keys,
		f: funcs,
	}
}

func (rq *requestMap) Lookup(name string) (IniterFunc, bool) {
	i := rq.t.Query(name)
	if rq.k[i] != name {
		return nil, false
	}
	return rq.f[i], true
}
