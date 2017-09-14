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

To address this problem, if initialization is not complete, requests should be queued up.