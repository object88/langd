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
	"github.com/object88/langd/health"
	"github.com/sourcegraph/jsonrpc2"
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

func (mc *MockConn) ResetCalls() {
	for name := range mc.calls {
		mc.calls[name] = 0
	}
	mc.totalCalls = 0
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

type MockLoader struct {
}

func NewMockLoader() langd.Loader {
	return nil
}

func Test_Handler_Uninited_Init(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	initParams := InitializeParams{
		RootURI: DocumentURI("foo"),
	}
	req := makeRequest(t, "initialize", 0, initParams, true)
	h.Handle(context.Background(), nil, req)

	if conn.calls["Reply"] != 1 {
		t.Errorf("Incorrect number of calls to Reply; expected 1, got %d\n%s", conn.calls["Reply"], conn.DumpCalls())
	}

	if h.rootURI != "foo" {
		t.Errorf("Incorrect RootURI property: expected 'foo', got '%s'", h.rootURI)
	}
}

func Test_Handler_Uninited_ExitNotification(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	req := makeRequest(t, "exit", 0, nil, true)
	h.Handle(context.Background(), nil, req)

	if conn.totalCalls != 0 {
		t.Errorf("Incorrect number of calls to Conn; expected 0, got %d", conn.totalCalls)
	}
}

func Test_Handler_Uninited_InvalidRequest(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	referencesParam := &referencesHandler{}
	req := makeRequest(t, referencesMethod, 0, referencesParam, false)
	h.Handle(context.Background(), nil, req)

	if conn.calls["ReplyWithError"] != 1 {
		t.Errorf("Incorrect number of calls to ReplyWithError; expected 1, got %d\n%s", conn.calls["ReplyWithError"], conn.DumpCalls())
	}
}

func Test_Handler_Inited_TextDocument_didOpen(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	initHandler(t, h, conn)

	req := makeRequest(t, didOpenNotification, h.NextCid(), DidOpenTextDocumentParams{}, true)

	cid := h.ccount
	h.Handle(context.Background(), nil, req)

	rh := h.rm[cid]
	if rh == nil {
		t.Errorf("RequestHandler is nil")
	}

	// Checking too soon; must be able to wait until we know that the request is processed.
	rhID := <-h.incomingQueue

	if rhID != cid {
		t.Errorf("Bad ID %d from incoming queue", rhID)
	}

	// if conn.calls["Reply"] != 1 {
	// 	t.Errorf("Incorrect number of calls to Reply; expected 1, got %d\n%s", conn.calls["Reply"], conn.DumpCalls())
	// }
}

func Test_Handler_Timing(t *testing.T) {
	conn, h, def := setupHandler()
	defer def()

	initHandler(t, h, conn)

	h.startProcessingQueue()

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

func initHandler(t *testing.T, h *Handler, conn *MockConn) {
	initParams := InitializeParams{
		RootURI: DocumentURI("foo"),
	}
	req := makeRequest(t, initializeMethod, h.NextCid(), initParams, false)
	h.Handle(context.Background(), nil, req)

	conn.ResetCalls()
}

func makeID(id int) jsonrpc2.ID {
	return jsonrpc2.ID{
		Num:      uint64(id),
		Str:      "",
		IsString: false,
	}
}

func makeRequest(t *testing.T, meth string, id int, params interface{}, isNotif bool) *jsonrpc2.Request {
	bytes, err := json.Marshal(params)
	if err != nil {
		t.Errorf("Failed to marshal InitializeParams: %s", err.Error())
	}
	p := json.RawMessage(bytes)

	req := &jsonrpc2.Request{
		Method: meth,
		Params: &p,
		ID:     makeID(id),
		Notif:  isNotif,
		Meta:   nil,
	}

	return req
}
