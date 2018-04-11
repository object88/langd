package requests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/spf13/viper"
)

const (
	initializedNotification = "initialized"
)

type initializedHandler struct {
	requestBase
}

func createInitializedHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &initializedHandler{
		requestBase: createRequestBase(ctx, h, req, true),
	}

	return rh
}

func (rh *initializedHandler) preprocess(params *json.RawMessage) error {
	fmt.Printf("InitializedHandler::preprocess\n")

	return nil
}

func (rh *initializedHandler) work() error {
	fmt.Printf("InitializedHandler::work\n")
	goString := "go"
	langdString := "langd"
	cParams := &ConfigurationParams{
		Items: []ConfigurationItem{
			ConfigurationItem{
				Section: &goString,
			},
			ConfigurationItem{
				Section: &langdString,
			},
		},
	}
	// result := &[]ConfigurationItem{}
	result := &[]interface{}{}
	fmt.Printf("InitializedHandler::work: requesting configuration\n")
	err := rh.h.conn.Call(context.Background(), "workspace/configuration", cParams, result)
	if err != nil {
		fmt.Printf("InitializedHandler::work: Error: %#v\n", err)
	}

	fmt.Printf("InitializedHandler::work: Result:\n\t%#v\n", result)

	s := viper.New()
	s.SetConfigType("json")
	s.Set(goString, (*result)[0])
	s.Set(langdString, (*result)[1])
	rh.h.workspace.AssignSettings(s)

	// rh.h.InitLoader("")

	rh.h.ConfigureLoader(s)

	return nil
}
