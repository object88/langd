package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/object88/langd"
	"github.com/object88/langd/log"
	"github.com/sourcegraph/jsonrpc2"
)

type handleReqFunc func(ctx context.Context, req *jsonrpc2.Request)

// Handler implements jsonrpc2.Handle
type Handler struct {
	conn          *jsonrpc2.Conn
	imap          IniterFuncMap
	workspace     *langd.Workspace
	log           *log.Log
	openedFiles   map[string]*bytes.Buffer
	incomingQueue chan requestHandler
	outgoingQueue chan replyHandler

	hFunc handleReqFunc
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

// NewHandler creates a new Handler
func NewHandler(imf *IniterMapFactory) *Handler {
	h := &Handler{
		openedFiles: map[string]*bytes.Buffer{},

		// Hopefully these queues are sufficiently deep.  Otherwise, the handler
		// will start blocking.
		incomingQueue: make(chan requestHandler, 1024),
		outgoingQueue: make(chan replyHandler, 256),

		imap: imf.Imap,
	}

	h.log = log.CreateLog(os.Stdout)
	h.log.SetLevel(log.Verbose)

	h.hFunc = h.unintedHandler

	return h
}

// SetConn assigns a JSONRPC2 connection and connects the handler
// to its log
func (h *Handler) SetConn(conn *jsonrpc2.Conn) {
	h.conn = conn
	h.log.AssignSender(h)
}

func (h *Handler) startProcessingQueue() {
	go func() {
		for {
			select {
			case rh := <-h.incomingQueue:
				err := rh.work()
				if err != nil {
					// Should we respond right away?  Set up with an auto-responder?
					// TODO: Cannot do nothing here; if the request is a method,
					// it wants a response.
					h.log.Errorf(err.Error())
					continue
				}
				if replier, ok := rh.(replyHandler); ok {
					h.outgoingQueue <- replier
				}

			case rh := <-h.outgoingQueue:
				result, err := rh.reply()
				if err != nil {
					// TODO: fill in actual error.
					h.conn.ReplyWithError(rh.ctx(), rh.id(), nil)
					continue
				}
				h.conn.Reply(rh.ctx(), rh.id(), result)
			}
		}
	}()
}

// Handle invokes the correct method handler based on the JSONRPC2 method
func (h *Handler) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.hFunc(ctx, req)
}

func (h *Handler) unintedHandler(ctx context.Context, req *jsonrpc2.Request) {
	meth := req.Method

	switch {
	case meth == initializeMethod:
		// The moment we've been waiting for; initialize.
		result, err := h.processInit(req.Params)
		if err != nil {
			// TODO: set up jsonrpc2.Error{}
			h.conn.ReplyWithError(ctx, req.ID, nil)
			return
		}
		h.conn.Reply(ctx, req.ID, result)

	case meth == exitNotification:
		// Should close down this connection.
		// TODO: handle exit
	case req.Notif:
		// Do nothing.
	default:
		// Respond with uninit'ed error.
		msg := fmt.Sprintf("Request '%s' not allowed until connection is initialized", req.Method)
		err := &jsonrpc2.Error{
			Code:    -32002,
			Message: msg,
		}
		h.conn.ReplyWithError(ctx, req.ID, err)
	}
}

func (h *Handler) initedHandler(ctx context.Context, req *jsonrpc2.Request) {
	meth := req.Method

	f, ok := h.imap[meth]
	if !ok {
		h.log.Verbosef("Unhandled method '%s'\n", meth)
		return
	}

	rh := f(ctx, h, req)

	_, isReplyHandler := rh.(replyHandler)
	if req.Notif && isReplyHandler {
		h.log.Errorf("Request handler is also a reply handler, but client does not listen for replies for method '%s'\n", meth)
	} else if !req.Notif && !isReplyHandler {
		h.log.Errorf("Request handler is not a reply handler, but client expects a reply for method '%s'\n", meth)
	}

	err := rh.preprocess(req.Params)
	if err != nil {
		// Bad news...
		// TODO: determine what to do here.
		return
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
