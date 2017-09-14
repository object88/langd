package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	"github.com/object88/langd"
	"github.com/object88/langd/log"
	"github.com/sourcegraph/jsonrpc2"
)

// Handler implements jsonrpc2.Handle
type Handler struct {
	conn          *jsonrpc2.Conn
	imap          map[string]initializerFunc
	workspace     *langd.Workspace
	log           *log.Log
	openedFiles   map[string]*bytes.Buffer
	incomingQueue chan requestHandler
	outgoingQueue chan replyHandler
}

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

type requestHandler interface {
	preprocess(p *json.RawMessage)
	work()

	id() jsonrpc2.ID
	ctx() context.Context
}

type replyHandler interface {
	requestHandler
	reply() interface{}
}

type initializerFunc func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler

type handleFunc func(ctx context.Context, req *jsonrpc2.Request) handleFuncer
type handleFuncer func()

// NewHandler creates a new Handler
func NewHandler() *Handler {
	h := &Handler{}

	// h.fmap = map[string]handleFunc{
	// 	definitionMethod:                  h.definition,
	// 	didChangeConfigurationMethod:      h.didChangeConfiguration,
	// 	didChangeTextDocumentNotification: h.didChangeTextDocument,
	// 	didCloseNotification:              h.didClose,
	// 	didOpenNotification:               h.didOpen,
	// 	exitNotification:                  h.exit,
	// 	initializeMethod:                  h.initialize,
	// 	"initialized":                     h.noopHandleFunc,
	// 	shutdownMethod:                    h.shutdown,
	// }

	h.imap = map[string]initializerFunc{
		definitionMethod:        createDefinitionHandler,
		initializedNotification: createInitializedHandler,
		initializeMethod:        createInitializeHandler,
	}

	h.log = log.CreateLog(os.Stdout)
	h.log.SetLevel(log.Verbose)

	h.openedFiles = map[string]*bytes.Buffer{}

	// Hopefully these queues are sufficiently deep.  Otherwise, the handler
	// will start blocking.
	h.incomingQueue = make(chan requestHandler, 1024)
	h.outgoingQueue = make(chan replyHandler, 256)

	// Start a routine to process requests
	go func() {
		for {
			select {
			case work := <-h.incomingQueue:
				work.work()
				if replier, ok := work.(replyHandler); ok {
					h.outgoingQueue <- replier
				}

			case work := <-h.outgoingQueue:
				result := work.reply()
				h.conn.Reply(work.ctx(), work.id(), result)
			}
		}
	}()

	return h
}

// SetConn assigns a JSONRPC2 connection and connects the handler
// to its log
func (h *Handler) SetConn(conn *jsonrpc2.Conn) {
	h.conn = conn
	h.log.AssignSender(h)
}

// Handle invokes the correct method handler based on the JSONRPC2 method
func (h *Handler) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) {
	f, ok := h.imap[req.Method]
	if !ok {
		h.log.Verbosef("Unhandled method '%s'\n", req.Method)
		return
	}

	rh := f(ctx, h, req)

	_, isReplyHandler := rh.(replyHandler)
	if req.Notif && isReplyHandler {
		h.log.Errorf("Request handler is also a reply handler, but client does not listen for replies for method '%s'", req.Method)
	} else if !req.Notif && !isReplyHandler {
		h.log.Errorf("Request handler is not a reply handler, but client expects a reply for method '%s'", req.Method)
	}

	rh.preprocess(req.Params)

	h.incomingQueue <- rh
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

func (h *Handler) noopHandleFunc(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
	// Nothing to do.
	return noopHandleFuncer
}

func noopHandleFuncer() {
	// Nothing to do here either.
}
