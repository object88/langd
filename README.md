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
