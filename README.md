# langd

## Notes

### Server behavior

When scanning the root URI for folders with Go code, we are skipping:

* any directory that begins with "." (i.e., `.git`, `.vscode`).
* any directory that is symlinked.  See [filepath.Walk](https://golang.org/pkg/path/filepath/#Walk) description.

### gRPC and Proto

If the gRPC API is changed (specifically, `/proto/langd.proto`), use the following command to rebuild the generated files.

`protoc -I proto proto/langd.proto --go_out=plugins=grpc:proto`

### Initialization

Noting [the LSP spec for initialization](https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#initialize-request), `initialize` should be the first request on a connection.  In this implementation, this request will be responded to before workspace initialization is complete.  The initialize responce will announce what capabilities the server has, which should include the `openClose` option.  If this workspace has been opened before, some files may automatically open, and the server may receive a `didOpen` request before initialiation is completed.

Connection & handler initialization has three stages: `uninitialized`, `initializing`, and `initialized`.  Expectations for handling requests are detailed in the [Initialize Request](https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#initialize-request) documentation in the LSP spec.

In the uninitialized state...

``` text
* for a request the respond should be errored with code: -32002. The message can be picked by the server.
* notifications should be dropped, except for the exit notification. This will allow the exit a server without an initialize request.
```

After the client has sent an `initialize` request, the client is expected to not send any further requests until it receives the `InitializeResult` response.

The server may return an `InitializeResult` response before it is ready to process requests.  This is the `initializing` state.  During this time, the client may send requests, and the server must queue them up for processing.  The queue is _not_ being processed at this time.

Once internal initialization is complete, the server is in the `initialized` stage, and will begin processing the queue.  New requests are still queued up, but the server is free to process them.

### Loading the workspace

The `initialize` request points to a filesystem path, which will be the root of the workspace.  That path and every directory under it will be scanned for Go code.  Additionally, and imported packages are loaded, down to the standard library.

During the loading process, the loader engine keeps a map of discovered distinct packages; a distinct package is a set of package files differentiated by OS, architecture, or other build flags.  Each time that a distinct package is encountered, the map be checked, and if the entry is missing, a goroutine will be started to load the AST.

As each package is loaded (and all dependencies are loaded), that package is fed to `config.Check` to build the map of uses, definitions, etc.

### Processing requests

Incoming requests are asynchronously processed by a connection handler.  A connection handler has two queues: `incomingQueue` and `outgoingQueue`. As requests are received from the JSONRPC2 server, they are handed off to the connection handler, which looks up and instantiates a request handler by method name. The request handler immediately performs some preprocessing on the request to unmarshal arguments and perform any other setup. Once the preprocessing is complete, the request handler is placed on the `incomingQueue`.

The `incomingQueue` and `outgoingQueue` are processed in a GoRoutine. When a request handler is pulled off the `incomingQueue`, the `work` method is invoked, which is expected to perform the processing of the actual request. Once this is complete, the request handler is checked to see if it is also a reply handler, and if so, the request / reply handler is placed on the `outgoingQueue`. Notifications do not reply, so those requests are not placed on the `outgoingQueue`.

Replies are supposed to be sent in same order as the requests. However, if the requests are processed asynchronously and some are faster to complete than others, then there is potenial for out-of-order replies. Additionally, some requests may require some inherent synchronous processing. For example, if the client sends a sequence of `didChange` notifications, those will need to be processed in order, and before a `definition` request is processed (as the `didChange` may have some bearing on the request of a definition).  For this reason, we use the Sigqueue struct to control the order of operations.

TODO: Describe `Sigqueue` in greater detail, as it applies here

The connection handler may need a RWMutex to handle requests. Requests which do not alter state (`textDocument/definition`, `textDocument/references`, etc) enter with a Read lock, allowing any other non-altering requests to enter as well. Once a request which would alter state is processed (`textDocument/didChange`, `textDocument/rename`, etc), a Write lock is requested. All currently running read operations will need to complete before the write can proceed, and each write operation will need to proceed synchronously. (Conceivably, some write operations could be performed asynchronously, but that is out of scope for an initial implementation.)

Because replies may be generated out of order with asynchronous processing, they must be queued up in the `outgoingQueue`.

### Changes to files

When the IDE opens a file for the user, a `textDocument/didOpen` request is sent to the server.  The server creates a `rope` representation of the file contents, which is altered with `textDocument/didChange` requests.

## Extensions to the Language Server Protocol

### Server Health request

The client may request some basic health metrics from the server; including instanteous CPU and memory usage.

_Request_

* method: 'health/instant'
* params: none

_Response_

* result: Instant defined as follows:

``` typescript
interface HealthInformation {
	/**
	 *
	 */
	cpu: number;

	/**
	 *
	 */
	memory: number;
}

```
