package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"

	"github.com/object88/langd"
	"github.com/object88/langd/health"
	"github.com/object88/langd/log"
	"github.com/object88/langd/sigqueue"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/spf13/viper"
)

type handleReqFunc func(ctx context.Context, req *jsonrpc2.Request)

// Handler implements jsonrpc2.Handle.  There is one Handler per client
// connection.  The Handler has a Workspace to service the incoming LSP
// requests.
type Handler struct {
	conn          *jsonrpc2.Conn
	rq            *requestMap
	workspace     *langd.Workspace
	log           *log.Log
	incomingQueue chan int
	outgoingQueue <-chan int

	// The base of the workspace
	rootURI string

	rm map[int]requestHandler
	sq *sigqueue.Sigqueue

	// call count; a monotonically increasing counter of calls made from the
	// client to this handler
	ccount int

	// hFunc will be either uninitedHandler or initedHandler, depending on
	// whether the connection's `initialize` request has been made.
	hFunc handleReqFunc

	load *health.Load
}

type requestHandler interface {
	preprocess(p *json.RawMessage) error
	work() error

	Ctx() context.Context
	ID() int
	Params() *json.RawMessage
	Method() string
	Replies() bool
	ReqID() jsonrpc2.ID
	RequireWriteLock() bool
}

type replyHandler interface {
	requestHandler
	reply() (interface{}, error)
}

// NewHandler creates a new Handler.
func NewHandler(load *health.Load, loader *langd.Loader) *Handler {
	// Hopefully these queues are sufficiently deep.  Otherwise, the handler
	// will start blocking.
	lc := langd.NewLoaderContext(loader, runtime.GOOS, runtime.GOARCH, runtime.GOROOT(), func(lc *langd.LoaderContext) {
		lc.Log = loader.Log
	})
	loader.Log.SetLevel(log.Verbose)

	outgoingQueue := make(chan int, 256)
	h := &Handler{
		incomingQueue: make(chan int, 1024),
		load:          load,
		log:           loader.Log,
		outgoingQueue: outgoingQueue,
		rm:            map[int]requestHandler{},
		rq:            newRequestMap(getIniterFuncs()),
		sq:            sigqueue.CreateSigqueue(outgoingQueue),

		workspace: langd.CreateWorkspace(loader, lc, loader.Log),
	}

	h.hFunc = h.uninitedHandler

	// Start a routine to process requests
	// h.startProcessingQueue()

	return h
}

// ConfigureLoader will instantiate the loader if its not present, and provide
// the GOROOT specified by the settings
func (h *Handler) ConfigureLoader(settings *viper.Viper) {
	root := settings.GetString("go.goroot")
	if root == "" {
		root = runtime.GOROOT()
	}
	loaderContext := langd.NewLoaderContext(h.workspace.Loader, runtime.GOOS, runtime.GOARCH, root, func(lc *langd.LoaderContext) {
		lc.Log = h.log
	})

	// FIXME: This loses the RWMutex on the existing workspace and creates a new one,
	// which is then unlocked without getting locked, and panics.
	h.workspace = langd.CreateWorkspace(h.workspace.Loader, loaderContext, h.log)
}

// NextCid returns the next call id
func (h *Handler) NextCid() int {
	cid := h.ccount
	h.ccount++
	return cid
}

// SetConnection assigns a JSONRPC2 connection and connects the handler
// to its log
func (h *Handler) SetConnection(conn *jsonrpc2.Conn) {
	h.conn = conn
	h.log.AssignSender(h)
}

func (h *Handler) startProcessingQueue() {
	go func() {
		for {
			select {
			case rhid := <-h.incomingQueue:
				h.startProcessing(rhid)
			case rhid := <-h.outgoingQueue:
				h.startResponding(rhid)
			}
		}
	}()
}

// Handle invokes the correct method handler based on the JSONRPC2 method
func (h *Handler) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.hFunc(ctx, req)
}

func (h *Handler) uninitedHandler(ctx context.Context, req *jsonrpc2.Request) {
	meth := req.Method

	switch {
	case meth == initializeMethod:
		// The moment we've been waiting for; initialize.
		result, err := h.processInit(req.Params)
		if err != nil {
			// TODO: set up jsonrpc2.Error{}
			e := &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: err.Error(),
			}
			h.conn.ReplyWithError(ctx, req.ID, e)
			return
		}
		h.conn.Reply(ctx, req.ID, result)

	// case initializedHandler:
	// 	result, err :=
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

	f, ok := h.rq.Lookup(meth)
	if !ok {
		h.log.Verbosef("Unhandled method '%s'\n", meth)
		return
	}

	rh := f(ctx, h, req)
	h.rm[rh.ID()] = rh

	// NOTE: This should probably be removed after all handlers have been
	// implemented.
	_, isReplyHandler := rh.(replyHandler)
	if req.Notif && isReplyHandler {
		h.log.Errorf("Request handler is also a reply handler, but client does not listen for replies for method '%s'\n", meth)
	} else if !req.Notif && !isReplyHandler {
		h.log.Errorf("Request handler is not a reply handler, but client expects a reply for method '%s'\n", meth)
	}

	// We are queueing up here because when the server has received its init,
	// it will respond immediately and asynchronously start processing the source
	// code.  The client can start sending more requests, and we need to keep
	// them on hand for after our source loading has completed.
	h.incomingQueue <- rh.ID()
}

func (h *Handler) startProcessing(rhid int) {
	rh := h.rm[rhid]

	h.log.Verbosef("%d: Starting request type %s\n", rhid, reflect.TypeOf(rh))

	err := rh.preprocess(rh.Params())
	if err != nil {
		// Bad news...
		// TODO: determine what to do here.
		h.log.Errorf("%d: Error preprocessing request type %s: %+v\n", rhid, err)
		return
	}

	if rh.Replies() {
		h.log.Verbosef("%d: Waiting...\n", rhid)

		h.sq.WaitOn(rh.ID())
	}

	// h.log.Verbosef("%d: Locking...\n", rhid)
	// h.workspace.Lock(rh.RequireWriteLock())

	go h.finishProcessing(rh)
}

func (h *Handler) finishProcessing(rh requestHandler) {
	h.log.Verbosef("%d: processing...\n", rh.ID())

	err := rh.work()

	// h.workspace.Unlock(rh.RequireWriteLock())

	if err != nil {
		// Should we respond right away?  Set up with an auto-responder?
		// TODO: Cannot do nothing here; if the request is a method,
		// it wants a response.
		h.log.Errorf("Error handling request type %s: %+v\n", reflect.TypeOf(rh), err)
		return
	}
	if rh.Replies() {
		h.sq.Ready(rh.ID())
	} else {
		delete(h.rm, rh.ID())
	}
}

func (h *Handler) startResponding(rhid int) {
	rh := h.rm[rhid]
	rr := rh.(replyHandler)
	result, err := rr.reply()
	if err != nil {
		// TODO: fill in actual error.
		h.conn.ReplyWithError(rh.Ctx(), rh.ReqID(), nil)
		return
	}
	h.conn.Reply(rh.Ctx(), rh.ReqID(), result)

	delete(h.rm, rhid)
}

// SendMessage implements log.SendMessage, so that the server can
// send a message to the client.
func (h *Handler) SendMessage(lvl log.Level, message string) {
	ctx := context.Background()

	t := ErrorMessageType
	switch lvl {
	case log.Verbose:
		t = LogMessageType
	case log.Info:
		t = InfoMessageType
	case log.Warn:
		t = WarningMessageType
	}

	params := &LogMessageParams{
		Type:    t,
		Message: message,
	}

	logMessage(ctx, h.conn, params)
}
