package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

// IniterFunc creates a request handler for a particular method
type IniterFunc func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler

// IniterFuncMap maps method names to their initializer functions
type IniterFuncMap map[string]IniterFunc

// IniterMapFactory contains the one instance of IniterFuncMaps
type IniterMapFactory struct {
	Imap IniterFuncMap
}

// CreateIniterMapFactory creates the (theorically, one) factory for initer
// methods.  This should be invoked once when the service starts, and shared
// among all connections, because these initer maps will always be the same.
func CreateIniterMapFactory() *IniterMapFactory {
	imap := IniterFuncMap{
		definitionMethod:                  createDefinitionHandler,
		didChangeConfigurationMethod:      createDidChangeConfigurationHandler,
		didChangeTextDocumentNotification: createDidChangeTextDocumentHandler,
		didCloseNotification:              createDidCloseHandler,
		didOpenNotification:               createDidOpenHandler,
		didSaveNotification:               createDidSaveHandler,
		exitNotification:                  createEditHandler,
		initializedNotification:           createNoopNotificationHandler,
		shutdownMethod:                    createShutdownHandler,
	}

	return &IniterMapFactory{
		Imap: imap,
	}
}
