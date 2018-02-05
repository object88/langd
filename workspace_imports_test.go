package langd

const identImportsTestProgram1 = `package foo

import (
	"../bar"
)

func add1(param1 int) int {
	bar.CountCall("add1")

	if bar.TotalCalls == 100 {
		// That's a lot of calls...
		return param1
	}

	param1++
	return param1
}
`

const identImportsTestProgram2 = `package bar

var calls = map[string]int{}
var TotalCalls = 0

func CountCall(source string) {
	TotalCalls++

	call, ok := calls[source]
	if !ok {
		call = 1
	} else {
		call++
	}
	calls[source] = call
}
`

// func Test_FromImports_LocateReferences_AsFunc(t *testing.T) {
// 	w := setup2(t)

// 	callCountInvokeOffset := nthIndex(identImportsTestProgram1, "CountCall", 0)
// 	callCountInvokePosition := w.Loader.Fset.Position(token.Pos(callCountInvokeOffset + 1))
// 	callCountIdent, _ := w.LocateIdent(&callCountInvokePosition)

// 	refPositions := w.LocateReferences(callCountIdent)
// 	if nil == refPositions {
// 		t.Fatalf("Returned nil from LocateReferences")
// 	}
// }
