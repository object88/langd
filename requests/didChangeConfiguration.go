package requests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didChangeConfigurationMethod = "workspace/didChangeConfiguration"
)

type didChangeConfigurationHandler struct {
	requestBase

	settings *viper.Viper
}

func createDidChangeConfigurationHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didChangeConfigurationHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *didChangeConfigurationHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got '%s'\n", didChangeConfigurationMethod)

	var typedParams DidChangeConfigurationParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return err
	}

	rh.settings = viper.New()
	rh.settings.SetConfigType("json")
	rh.settings.Set("go", typedParams.Settings["go"])
	rh.settings.Set("langd", typedParams.Settings["langd"])
	fmt.Printf("DidChangeConfiguration: All settings:\n\t%#v\n", rh.settings.AllSettings())

	return nil
}

func (rh *didChangeConfigurationHandler) work() error {

	// rh.h.workspace.AssignSettings(rh.settings)

	// rh.h.InitLoader("")

	rh.h.ConfigureLoader(rh.settings)

	return nil
}
