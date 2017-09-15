# langd

## Notes

### Server behavior

When scanning the root URI for folders with Go code, we are skipping:

* any directory that begins with "." (i.e., `.git`, `.vscode`).
* any directory that is symlinked.  See [filepath.Walk](https://golang.org/pkg/path/filepath/#Walk) description.

### gRPC and Proto

(Should go into deps?)
`go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`

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