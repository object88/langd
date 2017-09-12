package server

import (
	"context"
	"os"

	"github.com/object88/langd"
	"github.com/object88/langd/log"
	"github.com/sourcegraph/jsonrpc2"
)

// Handler implements jsonrpc2.Handle
type Handler struct {
	conn      *jsonrpc2.Conn
	fmap      map[string]handleFunc
	workspace *langd.Workspace
	log       *log.Log
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

	h.log = log.CreateLog(os.Stdout)
	h.log.SetLevel(log.Verbose)

	return h
}

func (h *Handler) noopHandleFunc(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// Nothing to do.
}

// Handle invokes the correct method handler based on the JSONRPC2 method
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	f, ok := h.fmap[req.Method]
	if !ok {
		h.log.Verbosef("Unhandled method '%s'\n", req.Method)
		return
	}

	f(ctx, conn, req)
}

// SendMessage implements log.SendMessage, so that the server can
// send a message to the client.
func (h *Handler) SendMessage(lvl log.Level, message string) {
	ctx := context.Background()

	t := Error
	switch lvl {
	case log.Verbose:
		t = Log
	case log.Info:
		t = Info
	case log.Warn:
		t = Warning
	}

	params := &LogMessageParams{
		Type:    t,
		Message: message,
	}

	logMessage(ctx, h.conn, params)
}

// SetConn assigns a JSONRPC2 connection and connects the handler
// to its log
func (h *Handler) SetConn(conn *jsonrpc2.Conn) {
	h.conn = conn
	h.log.AssignSender(h)
}
