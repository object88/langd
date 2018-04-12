package requests

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/object88/langd"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/object88/langd/health"
)

func Test_Handler_Created(t *testing.T) {
	load := health.StartLoadMonitoring()
	defer load.Close()

	loader := langd.NewLoader()

	h := NewHandler(load, loader)
	if h == nil {
		t.Fatal("Failed to create Handler")
	}

}

type MockConn struct {
	calls      map[string]int
	totalCalls int
}

func NewMockConn() *MockConn {
	mc := &MockConn{
		calls:      map[string]int{},
		totalCalls: 0,
	}

	t := reflect.TypeOf(mc)
	for i := 0; i < t.NumMethod(); i++ {
		mc.calls[t.Method(i).Name] = 0
	}

	return mc
}

func (mc *MockConn) DumpCalls() string {
	var sb strings.Builder
	sb.WriteString("Calls:\n")
	for name, count := range mc.calls {
		sb.WriteString("\t'")
		sb.WriteString(name)
		sb.WriteString("': ")
		sb.WriteString(strconv.Itoa(count))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (mc *MockConn) TrackCall() {
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	name := fn.Name()
	shortname := name[strings.LastIndex(name, ".")+1:]
	mc.calls[shortname]++
	mc.totalCalls++
}

func (mc *MockConn) Call(ctx context.Context, method string, params, result interface{}, opt ...jsonrpc2.CallOption) error {
	mc.TrackCall()
	return nil
}

func (mc *MockConn) Notify(ctx context.Context, method string, params interface{}, opt ...jsonrpc2.CallOption) error {
	mc.TrackCall()
	return nil
}

func (mc *MockConn) Close() error {
	mc.TrackCall()
	return nil
}

func (mc *MockConn) Reply(ctx context.Context, id jsonrpc2.ID, result interface{}) error {
	mc.TrackCall()
	return nil
}

func (mc *MockConn) ReplyWithError(ctx context.Context, id jsonrpc2.ID, respErr *jsonrpc2.Error) error {
	mc.TrackCall()
	return nil
}

func Test_Handler_Uninited_Init(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	initParams := InitializeParams{}
	req := makeRequest(t, "initialize", 0, initParams)
	h.Handle(context.Background(), nil, req)

	if conn.calls["Reply"] != 1 {
		t.Errorf("Incorrect number of calls to Reply; expected 1, got %d\n%s", conn.calls["Reply"], conn.DumpCalls())
	}
}

func Test_Handler_Uninited_ExitNotification(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	req := makeRequest(t, "exit", 0, nil)
	h.Handle(context.Background(), nil, req)

	if conn.totalCalls != 0 {
		t.Errorf("Incorrect number of calls to Conn; expected 0, got %d", conn.totalCalls)
	}
}

func Test_Handler_Uninited_InvalidRequest(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	req := makeRequest(t, "textDocument/didOpen", 0, nil)
	h.Handle(context.Background(), nil, req)

	if conn.calls["ReplyWithError"] != 1 {
		t.Errorf("Incorrect number of calls to ReplyWithError; expected 1, got %d\n%s", conn.calls["ReplyWithError"], conn.DumpCalls())
	}
}

func setupHandler() (*MockConn, *Handler, func()) {
	load := health.StartLoadMonitoring()

	loader := langd.NewLoader()
	h := NewHandler(load, loader)
	conn := NewMockConn()
	h.SetConnection(conn)

	return conn, h, func() {
		load.Close()
	}
}

func makeID(id int) jsonrpc2.ID {
	return jsonrpc2.ID{
		Num:      0,
		Str:      "",
		IsString: false,
	}
}

func makeRequest(t *testing.T, meth string, id int, params interface{}) *jsonrpc2.Request {
	bytes, err := json.Marshal(params)
	if err != nil {
		t.Errorf("Failed to marshal InitializeParams: %s", err.Error())
	}
	p := json.RawMessage(bytes)

	req := &jsonrpc2.Request{
		Method: meth,
		Params: &p,
		ID:     makeID(id),
		Notif:  false,
		Meta:   nil,
	}

	return req
}
