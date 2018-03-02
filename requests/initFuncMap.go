package requests

import (
	"context"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

// IniterFunc creates a request handler for a particular method
type IniterFunc func(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler

// IniterFuncMap maps method names to their initializer functions
type initerFuncMap map[string]IniterFunc

var initerFuncs initerFuncMap
var initerFuncsSync sync.Once

// CreateIniterMapFactory creates the (theorically, one) factory for initer
// methods.  This should be invoked once when the service starts, and shared
// among all connections, because these initer maps will always be the same.
func getIniterFuncs() initerFuncMap {
	initerFuncsSync.Do(func() {
		initerFuncs = initerFuncMap{
			definitionMethod:                  createDefinitionHandler,
			didChangeConfigurationMethod:      createDidChangeConfigurationHandler,
			didChangeTextDocumentNotification: createDidChangeTextDocumentHandler,
			didCloseNotification:              createDidCloseHandler,
			didOpenNotification:               createDidOpenHandler,
			didSaveNotification:               createDidSaveHandler,
			exitNotification:                  createExitHandler,
			healthMethod:                      createHealthHandler,
			hoverMethod:                       createHoverHandler,
			initializedNotification:           createInitializedHandler,
			referencesMethod:                  createReferencesHandler,
			shutdownMethod:                    createShutdownHandler,
			willSaveNotification:              createNoopNotificationHandler,
		}
	})

	return initerFuncs
}
