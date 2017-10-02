package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

type requestBase struct {
	_cid   int
	_ctx   context.Context
	req    *jsonrpc2.Request
	h      *Handler
	writes bool
}

func createRequestBase(ctx context.Context, h *Handler, req *jsonrpc2.Request, writes bool) requestBase {
	cid := h.NextCid()
	return requestBase{
		_cid:   cid,
		_ctx:   ctx,
		req:    req,
		h:      h,
		writes: writes,
	}
}

func (rh *requestBase) ID() int {
	return rh._cid
}

func (rh *requestBase) ctx() context.Context {
	return rh._ctx
}

func (rh *requestBase) reqID() jsonrpc2.ID {
	return rh.req.ID
}

func (rh *requestBase) method() string {
	return rh.req.Method
}

func (rh *requestBase) Params() *json.RawMessage {
	return rh.req.Params
}

func (rh *requestBase) Replies() bool {
	return !rh.req.Notif
}

func (rh *requestBase) requireWriteLock() bool {
	return rh.writes
}
