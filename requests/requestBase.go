package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

type requestBase struct {
	_ctx context.Context
	_id  jsonrpc2.ID
	h    *Handler
}

func createRequestBase(ctx context.Context, h *Handler, id jsonrpc2.ID) requestBase {
	return requestBase{
		_ctx: ctx,
		_id:  id,
		h:    h,
	}
}

func (rh *requestBase) ctx() context.Context {
	return rh._ctx
}

func (rh *requestBase) id() jsonrpc2.ID {
	return rh._id
}
