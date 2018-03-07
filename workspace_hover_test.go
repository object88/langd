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
	expected := "const foo.fooVal int = 0"
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
			expected:  "foo.DoFoo func()",
		},
		{
			name:      "with basic args",
			declFunc:  "(a int, b string) {}",
			usageArgs: "1, \"foo\"",
			expected:  "foo.DoFoo func(a int, b string)",
		},
		{
			name:      "with anonymized arg",
			declFunc:  "(a int, _ string) {}",
			usageArgs: "1, \"foo\"",
			expected:  "foo.DoFoo func(a int, _ string)",
		},
		{
			name:      "with repeated type args",
			declFunc:  "(a, b int) {}",
			usageArgs: "1, 2",
			expected:  "foo.DoFoo func(a, b int)",
		},
		{
			name:      "with struct arg",
			declFunc:  "(a int, b Foo) {}\n\ttype Foo struct {}",
			usageArgs: "1, foo.Foo{}",
			expected:  "foo.DoFoo func(a int, b foo.Foo)",
		},
		{
			name:      "with struct pointer arg",
			declFunc:  "(a int, b *Foo) {}\n\ttype Foo struct {}",
			usageArgs: "1, nil",
			expected:  "foo.DoFoo func(a int, b *foo.Foo)",
		},
		{
			name:      "with repeated type pointer args",
			declFunc:  "(a, b *Foo) {}\n\ttype Foo struct {}",
			usageArgs: "nil, nil",
			expected:  "foo.DoFoo func(a, b *foo.Foo)",
		},
		{
			name:      "with different struct pointer args",
			declFunc:  "(a *Foo1, b *Foo2) {}\n\ttype Foo1 struct {}\n\ttype Foo2 struct {}",
			usageArgs: "nil, nil",
			expected:  "foo.DoFoo func(a *foo.Foo1, b *foo.Foo2)",
		},
		{
			name:      "with a pointer pointer arg",
			declFunc:  "(a **int) {}",
			usageArgs: "nil",
			expected:  "foo.DoFoo func(a **int)",
		},
		{
			name:      "with a blank function arg",
			declFunc:  "(a int, f func()) {}",
			usageArgs: "1, func() {}",
			expected:  "foo.DoFoo func(a int, f func())",
		},
		{
			name:      "with a slice parameter",
			declFunc:  "(a int, b []string) {}",
			usageArgs: "1, nil",
			expected:  "foo.DoFoo func(a int, b []string)",
		},
		{
			name:      "with a slice parameter",
			declFunc:  "(a int, b []string, c []string) {}",
			usageArgs: "1, nil, nil",
			expected:  "foo.DoFoo func(a int, b, c []string)",
		},
		{
			name:      "with a variadic parameter",
			declFunc:  "(a int, b ...string) {}",
			usageArgs: "1",
			expected:  "foo.DoFoo func(a int, b ...string)",
		},
		{
			name:      "with a slice and variadic parameter",
			declFunc:  "(a int, b []string, c ...string) {}",
			usageArgs: "1, nil",
			expected:  "foo.DoFoo func(a int, b []string, c ...string)",
		},
		{
			name:      "with slices and a variadic parameter",
			declFunc:  "(a int, b []string, c []string, d ...string) {}",
			usageArgs: "1, nil, nil",
			expected:  "foo.DoFoo func(a int, b, c []string, d ...string)",
		},
		{
			name:      "with a basic type return",
			declFunc:  "() int { return 0 }",
			usageArgs: "",
			expected:  "foo.DoFoo func() int",
		},
		{
			name:      "with a pointer struct return",
			declFunc:  "() *Foo { return nil }\n\ttype Foo struct {}",
			usageArgs: "",
			expected:  "foo.DoFoo func() *foo.Foo",
		},
		{
			name:      "with a basic type and an error return",
			declFunc:  "() (int, error) { return 0, nil }",
			usageArgs: "",
			expected:  "foo.DoFoo func() (int, error)",
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

func Test_Workspace_Hover_Struct_Func(t *testing.T) {
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
	expected := "foo.ival int"
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
	expected := "type foo.fooer struct {}"
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
	expected := "type foo.fooer struct {\n\ta int\n\tb string\n}"
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
	expected := "type foo.barer struct {\n\tfoo.fooer\n\tc float32\n}"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}
