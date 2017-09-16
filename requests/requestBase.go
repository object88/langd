package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

type requestBase struct {
	_ctx context.Context
	_id  jsonrpc2.ID
	h    *Handler
	meth string
}

func createRequestBase(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestBase {
	return requestBase{
		_ctx: ctx,
		_id:  req.ID,
		h:    h,
		meth: req.Method,
	}
}

func (rh *requestBase) ctx() context.Context {
	return rh._ctx
}

func (rh *requestBase) id() jsonrpc2.ID {
	return rh._id
}