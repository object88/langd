package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

type InitializerFunc func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler

type InitFuncMap map[string]InitializerFunc

type IniterMapFactory struct {
	Imap InitFuncMap
}

// CreateIniterMapFactory creates the (theorically, one) factory for initer
// methods.  This should be invoked once when the service starts, and shared
// among all connections, because these initer maps will always be the same.
func CreateIniterMapFactory() *IniterMapFactory {
	imap := InitFuncMap{
		definitionMethod:                  createDefinitionHandler,
		didChangeConfigurationMethod:      createDidChangeConfigurationHandler,
		didChangeTextDocumentNotification: createDidChangeTextDocumentHandler,
		didCloseNotification:              createDidCloseHandler,
		didOpenNotification:               createDidOpenHandler,
		exitNotification:                  createEditHandler,
		initializedNotification:           createNoopNotificationHandler,
		shutdownMethod:                    createShutdownHandler,
	}

	return &IniterMapFactory{
		Imap: imap,
	}
}
