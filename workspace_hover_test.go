package langd

import (
	"fmt"
	"go/token"
	"testing"
)

func Test_Workspace_Hover_Local_Const(t *testing.T) {
	src1 := `package foo
	const fooVal = 0
	func IncFoo() int {
		return fooVal
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   10,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "``` go\nconst foo.fooVal int = 0\n```"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Package_Func(t *testing.T) {
	src1 := `package foo
	func DoFoo%s`

	src2 := `package bar
	import "../foo"
	func Do() {
		foo.DoFoo(%s)
	}`

	tests := []struct {
		name      string
		declFunc  string
		usageArgs string
		expected  string
	}{
		{
			name:      "empty",
			declFunc:  "() {}",
			usageArgs: "",
			expected:  "``` go\nfunc foo.DoFoo()\n```",
		},
		{
			name:      "with basic args",
			declFunc:  "(a int, b string) {}",
			usageArgs: "1, \"foo\"",
			expected:  "``` go\nfunc foo.DoFoo(a int, b string)\n```",
		},
		{
			name:      "with anonymized arg",
			declFunc:  "(a int, _ string) {}",
			usageArgs: "1, \"foo\"",
			expected:  "``` go\nfunc foo.DoFoo(a int, _ string)\n```",
		},
		{
			name:      "with repeated type args",
			declFunc:  "(a, b int) {}",
			usageArgs: "1, 2",
			expected:  "``` go\nfunc foo.DoFoo(a, b int)\n```",
		},
		{
			name:      "with struct arg",
			declFunc:  "(a int, b Foo) {}\n\ttype Foo struct {}",
			usageArgs: "1, foo.Foo{}",
			expected:  "``` go\nfunc foo.DoFoo(a int, b foo.Foo)\n```",
		},
		{
			name:      "with struct pointer arg",
			declFunc:  "(a int, b *Foo) {}\n\ttype Foo struct {}",
			usageArgs: "1, nil",
			expected:  "``` go\nfunc foo.DoFoo(a int, b *foo.Foo)\n```",
		},
		{
			name:      "with repeated type pointer args",
			declFunc:  "(a, b *Foo) {}\n\ttype Foo struct {}",
			usageArgs: "nil, nil",
			expected:  "``` go\nfunc foo.DoFoo(a, b *foo.Foo)\n```",
		},
		{
			name:      "with different struct pointer args",
			declFunc:  "(a *Foo1, b *Foo2) {}\n\ttype Foo1 struct {}\n\ttype Foo2 struct {}",
			usageArgs: "nil, nil",
			expected:  "``` go\nfunc foo.DoFoo(a *foo.Foo1, b *foo.Foo2)\n```",
		},
		{
			name:      "with a pointer pointer arg",
			declFunc:  "(a **int) {}",
			usageArgs: "nil",
			expected:  "``` go\nfunc foo.DoFoo(a **int)\n```",
		},
		{
			name:      "with a blank function arg",
			declFunc:  "(a int, f func()) {}",
			usageArgs: "1, func() {}",
			expected:  "``` go\nfunc foo.DoFoo(a int, f func())\n```",
		},
		{
			name:      "with a slice parameter",
			declFunc:  "(a int, b []string) {}",
			usageArgs: "1, nil",
			expected:  "``` go\nfunc foo.DoFoo(a int, b []string)\n```",
		},
		{
			name:      "with a slice parameter",
			declFunc:  "(a int, b []string, c []string) {}",
			usageArgs: "1, nil, nil",
			expected:  "``` go\nfunc foo.DoFoo(a int, b, c []string)\n```",
		},
		{
			name:      "with a variadic parameter",
			declFunc:  "(a int, b ...string) {}",
			usageArgs: "1",
			expected:  "``` go\nfunc foo.DoFoo(a int, b ...string)\n```",
		},
		{
			name:      "with a slice and variadic parameter",
			declFunc:  "(a int, b []string, c ...string) {}",
			usageArgs: "1, nil",
			expected:  "``` go\nfunc foo.DoFoo(a int, b []string, c ...string)\n```",
		},
		{
			name:      "with slices and a variadic parameter",
			declFunc:  "(a int, b []string, c []string, d ...string) {}",
			usageArgs: "1, nil, nil",
			expected:  "``` go\nfunc foo.DoFoo(a int, b, c []string, d ...string)\n```",
		},
		{
			name:      "with a basic type return",
			declFunc:  "() int { return 0 }",
			usageArgs: "",
			expected:  "``` go\nfunc foo.DoFoo() int\n```",
		},
		{
			name:      "with a pointer struct return",
			declFunc:  "() *Foo { return nil }\n\ttype Foo struct {}",
			usageArgs: "",
			expected:  "``` go\nfunc foo.DoFoo() *foo.Foo\n```",
		},
		{
			name:      "with a basic type and an error return",
			declFunc:  "() (int, error) { return 0, nil }",
			usageArgs: "",
			expected:  "``` go\nfunc foo.DoFoo() (int, error)\n```",
		},
		// More return value tests...
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			packages := map[string]map[string]string{
				"foo": map[string]string{
					"foo.go": fmt.Sprintf(src1, tc.declFunc),
				},
				"bar": map[string]string{
					"bar.go": fmt.Sprintf(src2, tc.usageArgs),
				},
			}

			w := workspaceSetup(t, "/go/src/bar", packages, false)

			p := &token.Position{
				Filename: "/go/src/bar/bar.go",
				Line:     4,
				Column:   7,
			}
			text, err := w.Hover(p)
			if err != nil {
				t.Fatal(err)
			}
			if text != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, text)
			}
		})
	}
}

func Test_Workspace_Hover_Struct_Pointer_Receiver_Func(t *testing.T) {
	src1 := `package foo
	type Foo struct {}
	func (%s *Foo) Do()`

	src2 := `package bar
	import "../foo"
	func Do() {
		f := foo.Foo{}
		f.Do()
	}`

	tests := []struct {
		name         string
		receiverName string
		expected     string
	}{
		{
			name:         "with named receiver",
			receiverName: "f",
			expected:     "``` go\nfunc (f *foo.Foo) Do()\n```",
		},
		{
			name:         "with named receiver",
			receiverName: "_",
			expected:     "``` go\nfunc (_ *foo.Foo) Do()\n```",
		},
		{
			name:         "with named receiver",
			receiverName: "",
			expected:     "``` go\nfunc (*foo.Foo) Do()\n```",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			packages := map[string]map[string]string{
				"foo": map[string]string{
					"foo.go": fmt.Sprintf(src1, tc.receiverName),
				},
				"bar": map[string]string{
					"bar.go": src2,
				},
			}

			w := workspaceSetup(t, "/go/src/bar", packages, false)

			p := &token.Position{
				Filename: "/go/src/bar/bar.go",
				Line:     5,
				Column:   5,
			}
			text, err := w.Hover(p)
			if err != nil {
				t.Fatal(err)
			}
			if text != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, text)
			}
		})
	}
}

func Test_Workspace_Hover_Struct_Value_Receiver_Func(t *testing.T) {
	src1 := `package foo
	type Foo struct {}
	func (%s Foo) Do()`

	src2 := `package bar
	import "../foo"
	func Do() {
		f := foo.Foo{}
		f.Do()
	}`

	tests := []struct {
		name         string
		receiverName string
		expected     string
	}{
		{
			name:         "with named receiver",
			receiverName: "f",
			expected:     "``` go\nfunc (f foo.Foo) Do()\n```",
		},
		{
			name:         "with named receiver",
			receiverName: "_",
			expected:     "``` go\nfunc (_ foo.Foo) Do()\n```",
		},
		{
			name:         "with named receiver",
			receiverName: "",
			expected:     "``` go\nfunc (foo.Foo) Do()\n```",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			packages := map[string]map[string]string{
				"foo": map[string]string{
					"foo.go": fmt.Sprintf(src1, tc.receiverName),
				},
				"bar": map[string]string{
					"bar.go": src2,
				},
			}

			w := workspaceSetup(t, "/go/src/bar", packages, false)

			p := &token.Position{
				Filename: "/go/src/bar/bar.go",
				Line:     5,
				Column:   5,
			}
			text, err := w.Hover(p)
			if err != nil {
				t.Fatal(err)
			}
			if text != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, text)
			}
		})
	}
}

func Test_Workspace_Hover_Local_Var_Basic(t *testing.T) {
	src1 := `package foo
	var ival int = 10
	func foof() int {
		ival += 1
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   3,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "``` go\nfoo.ival int\n```"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Struct_Empty(t *testing.T) {
	src1 := `package foo
	type fooer struct {
	}
	var ival fooer
	func foof() fooer {
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     6,
		Column:   10,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "``` go\ntype foo.fooer struct {}\n```"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Struct_With_Fields(t *testing.T) {
	src1 := `package foo
	type fooer struct {
		a int
		b string
	}
	var ival fooer
	func foof() fooer {
		ival.a += 1
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     8,
		Column:   3,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "``` go\ntype foo.fooer struct {\n\ta int\n\tb string\n}\n```"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Struct_Embedded(t *testing.T) {
	src1 := `package foo
	type fooer struct {
		a int
		b string
	}
	type barer struct {
		fooer
		c float32
	}
	var ival barer
	func foof() barer {
		ival.c += 1
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     12,
		Column:   3,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "``` go\ntype foo.barer struct {\n\tfoo.fooer\n\tc float32\n}\n```"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}
