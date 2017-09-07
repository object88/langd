package server

import (
	"context"
	"fmt"

	"github.com/object88/langd"
	"github.com/sourcegraph/jsonrpc2"
)

// Handler implements jsonrpc2.Handle
type Handler struct {
	fmap    map[string]handleFunc
	program *langd.Program
}

type handleFunc func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request)

// NewHandler creates a new Handler
func NewHandler() *Handler {
	h := &Handler{}

	h.fmap = map[string]handleFunc{
		definitionMethod:             h.definition,
		didChangeConfigurationMethod: h.didChangeConfiguration,
		initializeMethod:             h.initialize,
		"initialized":                h.noopHandleFunc,
		shutdownMethod:               h.shutdown,
	}

	return h
}

func (h *Handler) noopHandleFunc(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// Nothing to do.
}

// Handle invokes the correct method handler based on the JSONRPC2 method
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	f, ok := h.fmap[req.Method]
	if !ok {
		fmt.Printf("Unhandled method '%s'\n", req.Method)
		return
	}

	f(ctx, conn, req)
}
