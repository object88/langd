package collections

import (
	"fmt"
	"testing"
)

type Foo struct {
	hash    Hash
	checked bool
}

func (f *Foo) Hash() Hash {
	return f.hash
}

func (f *Foo) String() string {
	return fmt.Sprintf("0x%x", f.hash)
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
	f0 := &Foo{hash: 123}
	c.Insert(f0)

	n, created := c.Ensure(123, func() Hasher {
		return &Foo{hash: 123}
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
	f := n.Element
	if f != f0 {
		t.Error("Returned Foo pointer does not match the original inserted Foo")
	}
}

func Test_Caravan_Ensure_New(t *testing.T) {
	c := CreateCaravan()
	f0 := &Foo{hash: 123}
	c.Insert(f0)

	f1 := &Foo{hash: 234}
	n, created := c.Ensure(234, func() Hasher {
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
	f := n.Element
	if f != f1 {
		t.Error("Returned Foo pointer does not match the newly inserted Foo")
	}
}

func Test_Caravan_Insert(t *testing.T) {
	f0 := &Foo{hash: 123}
	f0a := &Foo{hash: 123}
	f1 := &Foo{hash: 456}

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
				if n, ok := c.nodes[v.Hash()]; !ok {
					t.Errorf("Internal nodes does not include hash '%x'", v.Hash())
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for hash '%x'", v.Hash())
					}
				}
			}

			for _, v := range tc.roots {
				if n, ok := c.roots[v.Hash()]; !ok {
					t.Errorf("Internal roots does not include hash '%x'", v.Hash())
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for hash '%x'", v.Hash())
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
	f0 := &Foo{hash: 123}
	f1 := &Foo{hash: 456}

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
					t.Errorf("Connecting from %s to %s, error: %s", conn.from, conn.to, err.Error())
				}
			}

			if len(tc.nodes) != len(c.nodes) {
				t.Errorf("Incorrect number of nodes: expected %d, got %d", len(tc.nodes), len(c.nodes))
			}

			if len(tc.roots) != len(c.roots) {
				t.Errorf("Incorrect number of roots: expected %d, got %d", len(tc.roots), len(c.roots))
			}

			for _, v := range tc.nodes {
				if n, ok := c.nodes[v.Hash()]; !ok {
					t.Errorf("Internal nodes does not include hash '%s'", v)
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for hash '%s'", v)
					}
				}
			}

			for _, v := range tc.roots {
				if n, ok := c.roots[v.Hash()]; !ok {
					t.Errorf("Internal roots does not include hash '%s'", v)
				} else {
					if n.Element != v {
						t.Errorf("Incorrect reference for hash '%s'", v)
					}
				}
			}
		})
	}
}

func Test_Caravan_Walk_Linear(t *testing.T) {
	foos := []Hasher{
		&Foo{hash: 12},
		&Foo{hash: 34},
		&Foo{hash: 56},
		&Foo{hash: 78},
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
					t.Errorf("Got wrong foo: expected %s, got %s", foos[tc.offset(i)], n.Element)
				}
				i++
			})
		})
	}
}

func Test_Caravan_Walk_Flatten(t *testing.T) {
	foos := []Hasher{
		&Foo{hash: 12},
		&Foo{hash: 34},
		&Foo{hash: 56},
		&Foo{hash: 78},
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
					visited := map[Hash]bool{}
					c.Walk(tc.dir, func(n *Node) {
						// Use this to test the fan-in, fan out, etc.
						hash := n.Element.Hash()
						if _, ok := visited[hash]; ok {
							t.Errorf("Revisiting node %s", n.Element)
						}
						visited[hash] = true

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
	first  Hash
	second Hash
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
				{first: 0xf0a, second: 0xf1a},
				{first: 0xf0a, second: 0xf1b},
				{first: 0xf0b, second: 0xf1a},
				{first: 0xf0b, second: 0xf1b},
			},
		},
		{
			name: "Up",
			dir:  WalkUp,
			orders: []order{
				{first: 0xf1a, second: 0xf0a},
				{first: 0xf1a, second: 0xf0b},
				{first: 0xf1b, second: 0xf0a},
				{first: 0xf1b, second: 0xf0b},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Hasher{}

			c.Walk(tc.dir, func(n *Node) {
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate 0x%x", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate 0x%x", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [0x%x:%d] > [0x%x:%d]", o.first, i0, o.second, i1)
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
		first  Hash
		orders []order
	}{
		{
			name:  "Down",
			dir:   WalkDown,
			first: 0xf00,
			orders: []order{
				{first: 0xf00, second: 0xf40},
				{first: 0xf00, second: 0xf1a},
				{first: 0xf00, second: 0xf1b},
				{first: 0xf3a, second: 0xf40},
				{first: 0xf3b, second: 0xf40},
			},
		},
		{
			name:  "Up",
			dir:   WalkUp,
			first: 0xf40,
			orders: []order{
				{first: 0xf40, second: 0xf00},
				{first: 0xf1a, second: 0xf00},
				{first: 0xf1b, second: 0xf00},
				{first: 0xf40, second: 0xf3a},
				{first: 0xf40, second: 0xf3b},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Hasher{}

			c.Walk(tc.dir, func(n *Node) {
				if i == 0 && n.Element.Hash() != tc.first {
					t.Errorf("Wrong first element; expected 0x%x, got %s", tc.first, n.Element)
				}
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate 0x%x", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate 0x%x", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [0x%x:%d] > [0x%x:%d]", o.first, i0, o.second, i1)
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
		first  Hash
		orders []order
	}{
		{
			name:  "Down",
			dir:   WalkDown,
			first: 0xf00,
			orders: []order{
				{first: 0xf00, second: 0xf60},
				{first: 0xf00, second: 0xf3a},
				{first: 0xf3a, second: 0xf60},
				{first: 0xf00, second: 0xf10},
				{first: 0xf10, second: 0xf3b},
				{first: 0xf3b, second: 0xf50},
				{first: 0xf50, second: 0xf60},
				{first: 0xf00, second: 0xf10},
				{first: 0xf10, second: 0xf20},
				{first: 0xf20, second: 0xf40},
				{first: 0xf40, second: 0xf50},
				{first: 0xf50, second: 0xf60},
			},
		},
		{
			name:  "Up",
			dir:   WalkUp,
			first: 0xf60,
			orders: []order{
				{first: 0xf60, second: 0xf00},
				{first: 0xf60, second: 0xf3a},
				{first: 0xf3a, second: 0xf00},
				{first: 0xf60, second: 0xf50},
				{first: 0xf50, second: 0xf3b},
				{first: 0xf3b, second: 0xf10},
				{first: 0xf10, second: 0xf00},
				{first: 0xf50, second: 0xf40},
				{first: 0xf40, second: 0xf20},
				{first: 0xf20, second: 0xf10},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Hasher{}

			c.Walk(tc.dir, func(n *Node) {
				if i == 0 && n.Element.Hash() != tc.first {
					t.Errorf("Wrong first element; expected 0x%x, got %s", tc.first, n.Element)
				}
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate 0x%x", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate 0x%x", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [0x%x:%d] > [0x%x:%d]", o.first, i0, o.second, i1)
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

	f0 := &Foo{hash: 0xf0}
	f1 := &Foo{hash: 0xf1}
	f2 := &Foo{hash: 0xf2}
	f3 := &Foo{hash: 0xf3}
	f4 := &Foo{hash: 0xf4}
	f5 := &Foo{hash: 0xf5}
	f6 := &Foo{hash: 0xf6}
	f7 := &Foo{hash: 0xf7}

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
		t.Errorf("Connect from %s to %s has err: %s", f0, f1, err.Error())
	}
	if err := c.Connect(f0, f2); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0, f2, err.Error())
	}
	if err := c.Connect(f1, f3); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1, f3, err.Error())
	}
	if err := c.Connect(f2, f3); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2, f3, err.Error())
	}
	if err := c.Connect(f2, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2, f4, err.Error())
	}
	if err := c.Connect(f3, f5); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3, f5, err.Error())
	}
	if err := c.Connect(f4, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f4, f6, err.Error())
	}
	if err := c.Connect(f5, f7); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f5, f7, err.Error())
	}
	if err := c.Connect(f6, f7); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f6, f7, err.Error())
	}

	tests := []struct {
		name   string
		dir    WalkDirection
		first  Hash
		orders []order
	}{
		{
			name:  "Down",
			dir:   WalkDown,
			first: 0xf0,
			orders: []order{
				{first: 0xf0, second: 0xf1},
				{first: 0xf0, second: 0xf2},
				{first: 0xf1, second: 0xf3},
				{first: 0xf2, second: 0xf3},
				{first: 0xf2, second: 0xf4},
				{first: 0xf3, second: 0xf5},
				{first: 0xf4, second: 0xf6},
				{first: 0xf5, second: 0xf7},
				{first: 0xf6, second: 0xf7},
			},
		},
		{
			name:  "Up",
			dir:   WalkUp,
			first: 0xf7,
			orders: []order{
				{first: 0xf7, second: 0xf5},
				{first: 0xf7, second: 0xf6},
				{first: 0xf5, second: 0xf3},
				{first: 0xf6, second: 0xf4},
				{first: 0xf3, second: 0xf1},
				{first: 0xf3, second: 0xf2},
				{first: 0xf4, second: 0xf2},
				{first: 0xf1, second: 0xf0},
				{first: 0xf2, second: 0xf0},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			walkedNodes := []Hasher{}

			for _, v := range c.nodes {
				f := v.Element.(*Foo)
				f.checked = false
			}

			c.Walk(tc.dir, func(n *Node) {
				f := n.Element.(*Foo)
				if i == 0 && f.Hash() != tc.first {
					t.Errorf("Wrong first element; expected 0x%x, got %s", tc.first, f)
				}
				if f.checked {
					t.Errorf("Already checked element %s", f)
				}

				if tc.dir == WalkDown {
					for _, v := range n.Ascendants {
						f1 := v.Element.(*Foo)
						if !f1.checked {
							t.Errorf("From %s, ascendant %s not checked", f, f1)
						}
					}
				} else {
					for _, v := range n.Descendants {
						f1 := v.Element.(*Foo)
						if !f1.checked {
							t.Errorf("From %s, descendant %s not checked", f, f1)
						}
					}
				}
				f.checked = true
				walkedNodes = append(walkedNodes, n.Element)
				i++
			})

			for _, v := range c.nodes {
				f := v.Element.(*Foo)
				if !f.checked {
					t.Errorf("Element %s was not checked", f)
				}
			}

			for _, o := range tc.orders {
				i0 := indexOf(walkedNodes, o.first)
				if i0 == -1 {
					t.Errorf("Failed to locate 0x%x", o.first)
				}
				i1 := indexOf(walkedNodes, o.second)
				if i1 == -1 {
					t.Errorf("Failed to locate 0x%x", o.second)
				}
				if i0 > i1 {
					t.Errorf("Incorrect walking order; [0x%x:%d] > [0x%x:%d]", o.first, i0, o.second, i1)
				}
			}
		})
	}
}

func createCross(t *testing.T) (*Caravan, []Hasher) {
	// f0a   f0b
	//  |\   /|
	//  | \ / |
	//  | / \ |
	//  |/   \|
	// f1a   f1b

	f0a := &Foo{hash: 0xf0a}
	f0b := &Foo{hash: 0xf0b}
	f1a := &Foo{hash: 0xf1a}
	f1b := &Foo{hash: 0xf1b}

	c := CreateCaravan()

	c.Insert(f0a)
	c.Insert(f0b)
	c.Insert(f1a)
	c.Insert(f1b)

	if err := c.Connect(f0a, f1a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0a, f1a, err.Error())
	}
	if err := c.Connect(f0a, f1b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0a, f1b, err.Error())
	}
	if err := c.Connect(f0b, f1a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0b, f1a, err.Error())
	}
	if err := c.Connect(f0b, f1b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0b, f1b, err.Error())
	}

	return c, []Hasher{f0a, f0b, f1a, f1b}
}

func createDiamond(t *testing.T) (*Caravan, []Hasher) {
	//      f0
	//     /  \
	//   f1a  f1b
	//   /      \
	// f2a      f2b
	//   \      /
	//   f3a  f3b
	//     \  /
	//      f4

	f0 := &Foo{hash: 0xf00}
	f1a := &Foo{hash: 0xf1a}
	f1b := &Foo{hash: 0xf1b}
	f2a := &Foo{hash: 0xf2a}
	f2b := &Foo{hash: 0xf2b}
	f3a := &Foo{hash: 0xf3a}
	f3b := &Foo{hash: 0xf3b}
	f4 := &Foo{hash: 0xf40}

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
		t.Errorf("Connect from %s to %s has err: %s", f0, f1a, err.Error())
	}
	if err := c.Connect(f0, f1b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0, f1b, err.Error())
	}
	if err := c.Connect(f1a, f2a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1a, f2a, err.Error())
	}
	if err := c.Connect(f1b, f2b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1b, f2b, err.Error())
	}
	if err := c.Connect(f2a, f3a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2a, f3a, err.Error())
	}
	if err := c.Connect(f2b, f3b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2b, f3b, err.Error())
	}
	if err := c.Connect(f3a, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3a, f4, err.Error())
	}
	if err := c.Connect(f3b, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3b, f4, err.Error())
	}

	return c, []Hasher{f0, f1a, f1b, f1a, f2b, f3a, f3b, f4}
}

func createOffsided(t *testing.T) (*Caravan, []Hasher) {
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

	f0 := &Foo{hash: 0xf00}
	f1 := &Foo{hash: 0xf10}
	f2 := &Foo{hash: 0xf20}
	f3a := &Foo{hash: 0xf3a}
	f3b := &Foo{hash: 0xf3b}
	f4 := &Foo{hash: 0xf40}
	f5 := &Foo{hash: 0xf50}
	f6 := &Foo{hash: 0xf60}

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
		t.Errorf("Connect from %s to %s has err: %s", f0, f6, err.Error())
	}

	// 1-step branch
	if err := c.Connect(f0, f3a); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0, f3a, err.Error())
	}
	if err := c.Connect(f3a, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3a, f6, err.Error())
	}

	// 2-step branch
	if err := c.Connect(f0, f1); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f0, f1, err.Error())
	}
	if err := c.Connect(f1, f3b); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1, f3b, err.Error())
	}
	if err := c.Connect(f3b, f5); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f3b, f5, err.Error())
	}
	if err := c.Connect(f5, f6); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f5, f6, err.Error())
	}

	// 3-step branch
	// Leg from f0 to f1 is already established.
	if err := c.Connect(f1, f2); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f1, f2, err.Error())
	}
	if err := c.Connect(f2, f4); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f2, f4, err.Error())
	}
	if err := c.Connect(f4, f5); err != nil {
		t.Errorf("Connect from %s to %s has err: %s", f4, f5, err.Error())
	}
	// Leg from f5 to f6 is already established.

	return c, []Hasher{f0, f1, f2, f3a, f3b, f4, f5, f6}
}

func indexOf(walked []Hasher, hash Hash) int {
	for k, v := range walked {
		if v.Hash() == hash {
			return k
		}
	}

	return -1
}
