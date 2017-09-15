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
	imap          map[string]InitializerFunc
	workspace     *langd.Workspace
	log           *log.Log
	openedFiles   map[string]*bytes.Buffer
	incomingQueue chan requestHandler
	outgoingQueue chan replyHandler

	imf       *IniterMapFactory
	initState initState
}

type requestHandler interface {
	preprocess(p *json.RawMessage) error
	work() error

	id() jsonrpc2.ID
	ctx() context.Context
}

type replyHandler interface {
	requestHandler
	reply() (interface{}, error)
}

type initState int

const (
	uninited initState = iota
	initing
	inited
)

type handleFunc func(ctx context.Context, req *jsonrpc2.Request) handleFuncer
type handleFuncer func()

// NewHandler creates a new Handler
func NewHandler(imf *IniterMapFactory) *Handler {
	h := &Handler{
		openedFiles: map[string]*bytes.Buffer{},

		// Hopefully these queues are sufficiently deep.  Otherwise, the handler
		// will start blocking.
		incomingQueue: make(chan requestHandler, 1024),
		outgoingQueue: make(chan replyHandler, 256),

		imf:       imf,
		initState: uninited,
	}

	// h.fmap = map[string]handleFunc{
	// 	didChangeConfigurationMethod:      h.didChangeConfiguration,
	// 	didChangeTextDocumentNotification: h.didChangeTextDocument,
	// 	exitNotification:                  h.exit,
	// 	shutdownMethod:                    h.shutdown,
	// }

	h.imap = map[string]InitializerFunc{
		definitionMethod:        createDefinitionHandler,
		didCloseNotification:    createDidCloseHandler,
		didOpenNotification:     createDidOpenHandler,
		initializedNotification: createInitializedHandler,
		initializeMethod:        createInitializeHandler,
	}

	h.log = log.CreateLog(os.Stdout)
	h.log.SetLevel(log.Verbose)

	// Start a routine to process requests
	go func() {
		for {
			select {
			case work := <-h.incomingQueue:
				err := work.work()
				if err != nil {
					// Should we respond right away?  Set up with an auto-responder?
					// TODO: Cannot do nothing here; if the request is a method,
					// it wants a response.
					h.log.Errorf(err.Error())
					continue
				}
				if replier, ok := work.(replyHandler); ok {
					h.outgoingQueue <- replier
				}

			case work := <-h.outgoingQueue:
				result, err := work.reply()
				if err != nil {
					// TODO: fill in actual error.
					h.conn.ReplyWithError(work.ctx(), work.id(), nil)
					continue
				}
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
	meth := req.Method

	if h.initState == uninited && meth != initializeMethod {
		// We are uninited; must reject or ignore method.
	}

	f, ok := h.imap[meth]
	if !ok {
		h.log.Verbosef("Unhandled method '%s'\n", meth)
		return
	}

	rh := f(ctx, h, req)

	_, isReplyHandler := rh.(replyHandler)
	if req.Notif && isReplyHandler {
		h.log.Errorf("Request handler is also a reply handler, but client does not listen for replies for method '%s'", meth)
	} else if !req.Notif && !isReplyHandler {
		h.log.Errorf("Request handler is not a reply handler, but client expects a reply for method '%s'", meth)
	}

	err := rh.preprocess(req.Params)
	if err != nil {
		// Bad news...
		// TODO: determine what to do here.
	}

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
