package langd

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"net/http"

// 	"github.com/osamingo/jsonrpc"
// )

// type (
// 	EchoHandler struct{}
// 	EchoParams  struct {
// 		Name string `json:"name"`
// 	}
// 	EchoResult struct {
// 		Message string `json:"message"`
// 	}
// )

// func (h *EchoHandler) ServeJSONRPC(c context.Context, params *json.RawMessage) (interface{}, *jsonrpc.Error) {
// 	var p EchoParams
// 	if err := jsonrpc.Unmarshal(params, &p); err != nil {
// 		return nil, err
// 	}

// 	return EchoResult{
// 		Message: "Hello, " + p.Name,
// 	}, nil
// }

// func init() {
// 	jsonrpc.RegisterMethod("Main.Echo", &EchoHandler{}, EchoParams{}, EchoResult{})
// }

// func main() {
// 	http.HandleFunc("/jrpc", func(w http.ResponseWriter, r *http.Request) {
// 		jsonrpc.HandlerFunc(r.Context(), w, r)
// 	})
// 	http.HandleFunc("/jrpc/debug", jsonrpc.DebugHandlerFunc)
// 	if err := http.ListenAndServe(":8080", nil); err != nil {
// 		log.Fatalln(err)
// 	}
// }
