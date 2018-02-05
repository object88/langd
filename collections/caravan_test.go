package collections

import (
	"testing"
)

type Foo struct {
	key     string
	checked bool
}

func (f *Foo) Key() string {
	return f.key
}

func Test_Caravan_Create(t *testing.T) {
	c := CreateCaravan()
	if c == nil {
		t.Errorf("Got nil")
	}

	if c.nodes == nil {
		t.Errorf("Internal structure nodes was not initialized")
	}

	if c.roots == nil {
		t.Errorf("Internal structure roots was not initialized")
	}
}

func Test_Caravan_Ensure_Existing(t *testing.T) {
	c := CreateCaravan()
	f0 := &Foo{key: "123"}
	c.Insert(f0)

	n, created := c.Ensure("123", func() Keyer {
		return &Foo{key: "123"}
	})
	if created {
		t.Error("Got 'true' created")
	}
	if n == nil {
		t.Error("Got nil node back from Ensure")
	}
	if n.Element == nil {
		t.Error("Got nil element back from Ensure")
	}
	f, ok := n.Element.(*Foo)
	if !ok {
		t.Error("Element is not a Foo")
	}
	if f != f0 {
		t.Error("Returned Foo pointer does not match the original inserted Foo")
	}
}

func Test_Caravan_Ensure_New(t *testing.T) {
	c := CreateCaravan()
	f0 := &Foo{key: "123"}
	c.Insert(f0)

	f1 := &Foo{key: "234"}
	n, created := c.Ensure("234", func() Keyer {
		return f1
	})
	if !created {
		t.Error("Got 'false' created")
	}
	if n == nil {
		t.Error("Got nil node back from Ensure")
	}
	if n.Element == nil {
		t.Error("Got nil element back from Ensure")
	}
	f, ok := n.Element.(*Foo)
	if !ok {
		t.Error("Element is not a Foo")
	}
	if f != f1 {
		t.Error("Returned Foo pointer does not match the newly inserted Foo")
	}
}

func Test_Caravan_Insert(t *testing.T) {
	f0 := &Foo{key: "123"}
	f0a := &Foo{key: "123"}
	f1 := &Foo{key: "456"}

	tests := []struct {
		name    string
		inserts []*Foo
		nodes   []*Foo
		roots   []*Foo
	}{
		{
			name:    "insert",
			inserts: []*Foo{f0},
			nodes:   []*Foo{f0},
			roots:   []*Foo{f0},
		},
		{
			name:    "insert_multiple",
			inserts: []*Foo{f0, f1},
			nodes:   []*Foo{f0, f1},
			roots:   []*Foo{f0, f1},
		},
		{
			name:    "insert_repeated",
			inserts: []*Foo{f0, f0},
			nodes:   []*Foo{f0},
			roots:   []*Foo{f0},
		},
		{
			name:    "insert_same_key",
			inserts: []*Foo{f0, f0a},
			nodes:   []*Foo{f0},
			roots:   []*Foo{f0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CreateCaravan()

			for _, i := range tc.inserts {
				c.Insert(i)
			}

			if len(tc.nodes) != len(c.nodes) {
				t.Errorf("Incorrect number of nodes: expected %d, got %d", len(tc.nodes), len(c.nodes))
			}

			if len(tc.roots) != len(c.roots) {
				t.Errorf("Incorrect number of roots: expected %d, got %d", len(tc.roots), len(c.roots))
			}

			for _, v := range tc.nodes {
				if n, ok := c.nodes[v.key]; !ok {
					t.Errorf("Internal nodes does not include key '%s'", v.key)
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for key '%s'", v.key)
					}
				}
			}

			for _, v := range tc.roots {
				if n, ok := c.roots[v.key]; !ok {
					t.Errorf("Internal roots does not include key '%s'", v.key)
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for key '%s'", v.key)
					}
				}
			}
		})
	}
}

type connect struct {
	from *Foo
	to   *Foo
}

// Lots more to test here.
func Test_Caravan_Connect(t *testing.T) {
	f0 := &Foo{key: "123"}
	f1 := &Foo{key: "456"}

	tests := []struct {
		name     string
		inserts  []*Foo
		connects []connect
		nodes    []*Foo
		roots    []*Foo
	}{
		{
			name:     "foo",
			inserts:  []*Foo{f0, f1},
			connects: []connect{{from: f0, to: f1}},
			nodes:    []*Foo{f0, f1},
			roots:    []*Foo{f0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CreateCaravan()

			for _, i := range tc.inserts {
				c.Insert(i)
			}

			for _, conn := range tc.connects {
				err := c.Connect(conn.from, conn.to)
				if err != nil {
					t.Errorf("Connecting from %s to %s, error: %s", conn.from.key, conn.to.key, err.Error())
				}
			}

			if len(tc.nodes) != len(c.nodes) {
				t.Errorf("Incorrect number of nodes: expected %d, got %d", len(tc.nodes), len(c.nodes))
			}

			if len(tc.roots) != len(c.roots) {
				t.Errorf("Incorrect number of roots: expected %d, got %d", len(tc.roots), len(c.roots))
			}

			for _, v := range tc.nodes {
				if n, ok := c.nodes[v.key]; !ok {
					t.Errorf("Internal nodes does not include key '%s'", v.key)
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for key '%s'", v.key)
					}
				}
			}

			for _, v := range tc.roots {
				if n, ok := c.roots[v.key]; !ok {
					t.Errorf("Internal roots does not include key '%s'", v.key)
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for key '%s'", v.key)
					}
				}
			}
		})
	}
}

func Test_Caravan_Walk_Linear(t *testing.T) {
	foos := []*Foo{
		&Foo{key: "12"},
		&Foo{key: "34"},
		&Foo{key: "56"},
		&Foo{key: "78"},
	}

	tests := []struct {
		name     string
		dir      WalkDirection
		leafNode int
		rootNode int
		offset   func(int) int
	}{
		{
			name:     "Down",
			dir:      WalkDown,
			leafNode: len(foos) - 1,
			rootNode: 0,
			offset:   func(x int) int { return x },
		},
		{
			name:     "Up",
			dir:      WalkUp,
			leafNode: 0,
			rootNode: len(foos) - 1,
			offset:   func(x int) int { return len(foos) - x - 1 },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CreateCaravan()
			for _, v := range foos {
				c.Insert(v)
			}

			for i := 0; i < len(foos)-1; i++ {
				c.Connect(foos[i], foos[i+1])
			}

			i := 0
			c.Walk(tc.dir, func(n *Node) {
				isRoot := len(n.Ascendants) == 0
				isLeaf := len(n.Descendants) == 0
				if isRoot != (i == tc.rootNode) {
					t.Errorf("Got isRoot for non-root")
				}
				if isLeaf != (i == tc.leafNode) {
					t.Error("Got isLeaf for non-leaf")
				}
				if n.Element != foos[tc.offset(i)] {
					t.Errorf("Got wrong foo: expected %s, got %s", foos[tc.offset(i)].key, n.Element.Key())
				}
				i++
			})
		})
	}
}

func Test_Caravan_Walk_Flatten(t *testing.T) {
	foos := []*Foo{
		&Foo{key: "12"},
		&Foo{key: "34"},
		&Foo{key: "56"},
		&Foo{key: "78"},
	}

	tests := []struct {
		name string
		dir  WalkDirection
	}{
		{
			name: "Down",
			dir:  WalkDown,
		},
		{
			name: "Up",
			dir:  WalkUp,
		},
	}

	fannings := []struct {
		name      string
		fromIndex func(int) int
		toIndex   func(int) int
	}{
		{
			name:      "fan_out",
			fromIndex: func(_ int) int { return 0 },
			toIndex:   func(x int) int { return x + 1 },
		},
		{
			name:      "fan_in",
			fromIndex: func(x int) int { return x },
			toIndex:   func(_ int) int { return len(foos) - 1 },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, fan := range fannings {
				t.Run(fan.name, func(t *testing.T) {
					c := CreateCaravan()
					for _, v := range foos {
						c.Insert(v)
					}

					for i := 0; i < len(foos)-1; i++ {
						f0 := foos[fan.fromIndex(i)]
						f1 := foos[fan.toIndex(i)]
						c.Connect(f0, f1)
					}

					i := 0
					visited := map[string]bool{}
					c.Walk(tc.dir, func(n *Node) {
						// Use this to test the fan-in, fan out, etc.
						key := n.Element.Key()
						if _, ok := visited[key]; ok {
							t.Errorf("Revisiting node %s", key)
						}
						visited[key] = true

						i++
					})
					if len(visited) != len(foos) {
						t.Errorf("Did not visit all nodes; expected %d, got %d", len(foos), len(visited))
					}
				})
			}
		})
	}
}

type order struct {
	first  string
	second string
}

func Test_Caravan_Walk_Cross(t *testing.T) {
	c, _ := createCross(t)

	tests := []struct {
		name   string
		dir    WalkDirection
		orders []order
	}{
		{
			name: "Down",
			dir:  WalkDown,
			orders: []order{
				{first: "f0a", second: "f1a"},
				{first: "f0a", second: "f1b"},
				{first: "f0b", second: "f1a"},
				{first: "f0b", second: "f1b"},
			},
		},
		{
			name: "Up",
			dir:  WalkUp,
			orders: []order{
				{first: "f1a", second: "f0a"},
				{first: "f1a", second: "f0b"},
				{first: "f1b", second: "f0a"},
				{first: "f1b", second: "f0b"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Keyer{}

			c.Walk(tc.dir, func(n *Node) {
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate %s", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate %s", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [%s:%d] > [%s:%d]", o.first, i0, o.second, i1)
				}
			}
		})
	}
}

func Test_Caravan_Walk_Diamond(t *testing.T) {
	c, _ := createDiamond(t)

	tests := []struct {
		name   string
		dir    WalkDirection
		first  string
		orders []order
	}{
		{
			name:  "Down",
			dir:   WalkDown,
			first: "f0",
			orders: []order{
				{first: "f0", second: "f4"},
				{first: "f0", second: "f1a"},
				{first: "f0", second: "f1b"},
				{first: "f3a", second: "f4"},
				{first: "f3b", second: "f4"},
			},
		},
		{
			name:  "Up",
			dir:   WalkUp,
			first: "f4",
			orders: []order{
				{first: "f4", second: "f0"},
				{first: "f1a", second: "f0"},
				{first: "f1b", second: "f0"},
				{first: "f4", second: "f3a"},
				{first: "f4", second: "f3b"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Keyer{}

			c.Walk(tc.dir, func(n *Node) {
				if i == 0 && n.Element.Key() != tc.first {
					t.Errorf("Wrong first element; expected %s, got %s", tc.first, n.Element.Key())
				}
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate %s", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate %s", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [%s:%d] > [%s:%d]", o.first, i0, o.second, i1)
				}
			}
		})
	}
}

func Test_Caravan_Walk_Offsided(t *testing.T) {
	c, _ := createOffsided(t)

	tests := []struct {
		name   string
		dir    WalkDirection
		first  string
		orders []order
	}{
		{
			name:  "Down",
			dir:   WalkDown,
			first: "f0",
			orders: []order{
				{first: "f0", second: "f6"},
				{first: "f0", second: "f3a"},
				{first: "f3a", second: "f6"},
				{first: "f0", second: "f1"},
				{first: "f1", second: "f3b"},
				{first: "f3b", second: "f5"},
				{first: "f5", second: "f6"},
				{first: "f0", second: "f1"},
				{first: "f1", second: "f2"},
				{first: "f2", second: "f4"},
				{first: "f4", second: "f5"},
				{first: "f5", second: "f6"},
			},
		},
		{
			name:  "Up",
			dir:   WalkUp,
			first: "f6",
			orders: []order{
				{first: "f6", second: "f0"},
				{first: "f6", second: "f3a"},
				{first: "f3a", second: "f0"},
				{first: "f6", second: "f5"},
				{first: "f5", second: "f3b"},
				{first: "f3b", second: "f1"},
				{first: "f1", second: "f0"},
				{first: "f5", second: "f4"},
				{first: "f4", second: "f2"},
				{first: "f2", second: "f1"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Keyer{}

			c.Walk(tc.dir, func(n *Node) {
				if i == 0 && n.Element.Key() != tc.first {
					t.Errorf("Wrong first element; expected %s, got %s", tc.first, n.Element.Key())
				}
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate %s", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate %s", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [%s:%d] > [%s:%d]", o.first, i0, o.second, i1)
				}
			}
		})
	}
}

func Test_Caravan_Walk_Foo(t *testing.T) {
	// f0
	//  | \
	// f1  f2
	//  | / \
	// f3    f4
	//  |   /
	// f5  f6
	//  |/
	// f7

	f0 := &Foo{key: "f0"}
	f1 := &Foo{key: "f1"}
	f2 := &Foo{key: "f2"}
	f3 := &Foo{key: "f3"}
	f4 := &Foo{key: "f4"}
	f5 := &Foo{key: "f5"}
	f6 := &Foo{key: "f6"}
	f7 := &Foo{key: "f7"}

	c := CreateCaravan()

	c.Insert(f0)
	c.Insert(f1)
	c.Insert(f2)
	c.Insert(f3)
	c.Insert(f4)
	c.Insert(f5)
	c.Insert(f6)
	c.Insert(f7)

	if err := c.Connect(f0, f1); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f1.Key(), err.Error())
	}
	if err := c.Connect(f0, f2); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f2.Key(), err.Error())
	}
	if err := c.Connect(f1, f3); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1.Key(), f3.Key(), err.Error())
	}
	if err := c.Connect(f2, f3); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2.Key(), f3.Key(), err.Error())
	}
	if err := c.Connect(f2, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2.Key(), f4.Key(), err.Error())
	}
	if err := c.Connect(f3, f5); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3.Key(), f5.Key(), err.Error())
	}
	if err := c.Connect(f4, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f4.Key(), f6.Key(), err.Error())
	}
	if err := c.Connect(f5, f7); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f5.Key(), f7.Key(), err.Error())
	}
	if err := c.Connect(f6, f7); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f6.Key(), f7.Key(), err.Error())
	}

	tests := []struct {
		name   string
		dir    WalkDirection
		first  string
		orders []order
	}{
		{
			name:  "Down",
			dir:   WalkDown,
			first: "f0",
			orders: []order{
				{first: "f0", second: "f1"},
				{first: "f0", second: "f2"},
				{first: "f1", second: "f3"},
				{first: "f2", second: "f3"},
				{first: "f2", second: "f4"},
				{first: "f3", second: "f5"},
				{first: "f4", second: "f6"},
				{first: "f5", second: "f7"},
				{first: "f6", second: "f7"},
			},
		},
		{
			name:  "Up",
			dir:   WalkUp,
			first: "f7",
			orders: []order{
				{first: "f7", second: "f5"},
				{first: "f7", second: "f6"},
				{first: "f5", second: "f3"},
				{first: "f6", second: "f4"},
				{first: "f3", second: "f1"},
				{first: "f3", second: "f2"},
				{first: "f4", second: "f2"},
				{first: "f1", second: "f0"},
				{first: "f2", second: "f0"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Keyer{}

			for _, v := range c.nodes {
				f, _ := v.Element.(*Foo)
				f.checked = false
			}

			c.Walk(tc.dir, func(n *Node) {
				if i == 0 && n.Element.Key() != tc.first {
					t.Errorf("Wrong first element; expected %s, got %s", tc.first, n.Element.Key())
				}
				f, _ := n.Element.(*Foo)
				if f.checked {
					t.Errorf("Already checked element %s", f.Key())
				}

				if tc.dir == WalkDown {
					for _, v := range n.Ascendants {
						f1, _ := v.Element.(*Foo)
						if !f1.checked {
							t.Errorf("From %s, ascendant %s not checked", f.Key(), f1.Key())
						}
					}
				} else {
					for _, v := range n.Descendants {
						f1, _ := v.Element.(*Foo)
						if !f1.checked {
							t.Errorf("From %s, descendant %s not checked", f.Key(), f1.Key())
						}
					}
				}
				f.checked = true
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, v := range c.nodes {
				f, _ := v.Element.(*Foo)
				if !f.checked {
					t.Errorf("Element %s was not checked", f.Key())
				}
			}

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate %s", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate %s", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [%s:%d] > [%s:%d]", o.first, i0, o.second, i1)
				}
			}
		})
	}
}

func createCross(t *testing.T) (*Caravan, []*Foo) {
	// f0a   f0b
	//  |\   /|
	//  | \ / |
	//  | / \ |
	//  |/   \|
	// f1a   f1b

	f0a := &Foo{key: "f0a"}
	f0b := &Foo{key: "f0b"}
	f1a := &Foo{key: "f1a"}
	f1b := &Foo{key: "f1b"}

	c := CreateCaravan()

	c.Insert(f0a)
	c.Insert(f0b)
	c.Insert(f1a)
	c.Insert(f1b)

	if err := c.Connect(f0a, f1a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0a.Key(), f1a.Key(), err.Error())
	}
	if err := c.Connect(f0a, f1b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0a.Key(), f1b.Key(), err.Error())
	}
	if err := c.Connect(f0b, f1a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0b.Key(), f1a.Key(), err.Error())
	}
	if err := c.Connect(f0b, f1b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0b.Key(), f1b.Key(), err.Error())
	}

	return c, []*Foo{f0a, f0b, f1a, f1b}
}

func createDiamond(t *testing.T) (*Caravan, []*Foo) {
	//      f0
	//     /  \
	//   f1a  f1b
	//   /      \
	// f2a      f2b
	//   \      /
	//   f3a  f3b
	//     \  /
	//      f4

	f0 := &Foo{key: "f0"}
	f1a := &Foo{key: "f1a"}
	f1b := &Foo{key: "f1b"}
	f2a := &Foo{key: "f2a"}
	f2b := &Foo{key: "f2b"}
	f3a := &Foo{key: "f3a"}
	f3b := &Foo{key: "f3b"}
	f4 := &Foo{key: "f4"}

	c := CreateCaravan()

	c.Insert(f0)
	c.Insert(f1a)
	c.Insert(f1b)
	c.Insert(f2a)
	c.Insert(f2b)
	c.Insert(f3a)
	c.Insert(f3b)
	c.Insert(f4)

	if err := c.Connect(f0, f1a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f1a.Key(), err.Error())
	}
	if err := c.Connect(f0, f1b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f1b.Key(), err.Error())
	}
	if err := c.Connect(f1a, f2a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1a.Key(), f2a.Key(), err.Error())
	}
	if err := c.Connect(f1b, f2b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1b.Key(), f2b.Key(), err.Error())
	}
	if err := c.Connect(f2a, f3a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2a.Key(), f3a.Key(), err.Error())
	}
	if err := c.Connect(f2b, f3b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2b.Key(), f3b.Key(), err.Error())
	}
	if err := c.Connect(f3a, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3a.Key(), f4.Key(), err.Error())
	}
	if err := c.Connect(f3b, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3b.Key(), f4.Key(), err.Error())
	}

	return c, []*Foo{f0, f1a, f1b, f1a, f2b, f3a, f3b, f4}
}

func createOffsided(t *testing.T) (*Caravan, []*Foo) {
	//      f0
	//     / | \
	//    /  |   \
	//   /   |    f1
	//  |    |    /  \
	//  |    |   |   f2
	//  |    |   |    |
	//  |   f3a f3b   |
	//  |    |   |    |
	//  |    |   |   f4
	//  |    |    \  /
	//   \   |     f5
	//    \  |   /
	//     \ | /
	//      f6

	f0 := &Foo{key: "f0"}
	f1 := &Foo{key: "f1"}
	f2 := &Foo{key: "f2"}
	f3a := &Foo{key: "f3a"}
	f3b := &Foo{key: "f3b"}
	f4 := &Foo{key: "f4"}
	f5 := &Foo{key: "f5"}
	f6 := &Foo{key: "f6"}

	c := CreateCaravan()

	c.Insert(f0)
	c.Insert(f1)
	c.Insert(f2)
	c.Insert(f3a)
	c.Insert(f3b)
	c.Insert(f4)
	c.Insert(f5)
	c.Insert(f6)

	// Direct branch
	if err := c.Connect(f0, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f6.Key(), err.Error())
	}

	// 1-step branch
	if err := c.Connect(f0, f3a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f3a.Key(), err.Error())
	}
	if err := c.Connect(f3a, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3a.Key(), f6.Key(), err.Error())
	}

	// 2-step branch
	if err := c.Connect(f0, f1); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0.Key(), f1.Key(), err.Error())
	}
	if err := c.Connect(f1, f3b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1.Key(), f3b.Key(), err.Error())
	}
	if err := c.Connect(f3b, f5); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3b.Key(), f5.Key(), err.Error())
	}
	if err := c.Connect(f5, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f5.Key(), f6.Key(), err.Error())
	}

	// 3-step branch
	// Leg from f0 to f1 is already established.
	if err := c.Connect(f1, f2); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1.Key(), f2.Key(), err.Error())
	}
	if err := c.Connect(f2, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2.Key(), f4.Key(), err.Error())
	}
	if err := c.Connect(f4, f5); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f4.Key(), f5.Key(), err.Error())
	}
	// Leg from f5 to f6 is already established.

	return c, []*Foo{f0, f1, f2, f3a, f3b, f4, f5, f6}
}

func indexOf(walked []Keyer, key string) int {
	for k, v := range walked {
		if v.Key() == key {
			return k
		}
	}

	return -1
}
