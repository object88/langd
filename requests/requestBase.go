package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

type requestBase struct {
	ctx    context.Context
	h      *Handler
	id     int
	req    *jsonrpc2.Request
	writes bool
}

func createRequestBase(ctx context.Context, h *Handler, req *jsonrpc2.Request, writes bool) requestBase {
	cid := h.NextCid()
	return requestBase{
		ctx:    ctx,
		h:      h,
		id:     cid,
		req:    req,
		writes: writes,
	}
}

// ID returns the monotonically increasing request count assigned to this
// method by the connection handler
func (rh *requestBase) ID() int {
	return rh.id
}

// Ctx returns the request's context
func (rh *requestBase) Ctx() context.Context {
	return rh.ctx
}

// ReqID returns the LSP request ID
func (rh *requestBase) ReqID() jsonrpc2.ID {
	return rh.req.ID
}

// Method returns the name of the method being invoked
func (rh *requestBase) Method() string {
	return rh.req.Method
}

// Params returns the raw JSONRPC2 request parameters
func (rh *requestBase) Params() *json.RawMessage {
	return rh.req.Params
}

// Replies indicates whether this request handler is also a reply handler
func (rh *requestBase) Replies() bool {
	return !rh.req.Notif
}

// RequireWriteLock indicates whether this method should aquire a write
// lock on the workspace, because it will change some state.
func (rh *requestBase) RequireWriteLock() bool {
	return rh.writes
}
