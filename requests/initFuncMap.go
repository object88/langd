package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

// Request-func mapping depends on the initialization state of the server.
// The connection may be uninitialized, initializing, or initialized.

type InitializerFunc func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler

type InitFuncMap map[string]InitializerFunc

type IniterMapFactory struct {
	preinit  InitFuncMap
	postinit InitFuncMap
}

// CreateIniterMapFactory creates the (theorically, one) factory for initer
// methods.  This should be invoked once when the service starts, and shared
// among all connections, because these initer maps will always be the same.
func CreateIniterMapFactory() *IniterMapFactory {
	preinit := InitFuncMap{
		definitionMethod:        createUninitedMethodHandler,
		didCloseNotification:    createNoopNotificationHandler,
		didOpenNotification:     createNoopNotificationHandler,
		initializedNotification: createNoopNotificationHandler,
		initializeMethod:        createInitializedHandler,
	}
	postinit := InitFuncMap{
		definitionMethod:        createDefinitionHandler,
		didCloseNotification:    createDidCloseHandler,
		didOpenNotification:     createDidOpenHandler,
		initializedNotification: createNoopNotificationHandler,
		initializeMethod:        createInitializeHandler,
	}

	return &IniterMapFactory{
		preinit:  preinit,
		postinit: postinit,
	}
}
