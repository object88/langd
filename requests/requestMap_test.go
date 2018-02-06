package requests

import (
	"context"
	"testing"

	"github.com/sourcegraph/jsonrpc2"
)

func Test_RequestMap2(t *testing.T) {
	aFunc := func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
		return nil
	}
	bFunc := func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
		t.Fatalf("NOPE")
		return nil
	}
	cFunc := func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
		t.Fatalf("NOPE2")
		return nil
	}
	m := map[string]IniterFunc{
		"a": aFunc,
		"b": bFunc,
		"c": cFunc,
	}

	rq := newRequestMap(m)
	i1, ok := rq.Lookup("a")
	if !ok {
		t.Fatalf("Got !ok for value key")
	}
	if i1 == nil {
		t.Fatalf("Got nil method back")
	}
	i1(context.Background(), nil, nil)

	i2, ok := rq.Lookup("d")
	if ok {
		t.Fatalf("Got ok for invalid key")
	}
	if i2 != nil {
		t.Fatalf("Got non-nil method back")
	}
}
