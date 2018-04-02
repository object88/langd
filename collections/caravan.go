package collections

import (
	"errors"
	"fmt"
	"sync"
)

const (
	// WalkUp indicates that the walk function will start with leaves, and
	// traverse the caravan towards roots.
	WalkUp WalkDirection = iota

	// WalkDown indicates that the walk function will start with the roots,
	// and will traverse the caravan towards the leaves.
	WalkDown
)

// WalkDirection provides the Walk function with a direction to traverse the
// caravan
type WalkDirection int

// CaravanWalker function is the callback for the `Walk` function
type CaravanWalker func(node *Node)

// Caravan is a collection of independent directed acyclic graphs (DAGs).
// Items are inserted into a caravan, and the graph structure is constructed.
// The caravan exposes all items which are root level, and allows the
// structure to be walked.
// "Good dags.  D'ya like dags?"
// "Dags?"
// "What?"
// "Yeah, dags"
// "Oh, dogs.  Sure, I like dags.  I like caravans more."
// -- http://www.imdb.com/character/ch0003626/quotes
type Caravan struct {
	nodes map[Key]*Node
	roots map[Key]*Node
	m     sync.Mutex
}

// Node is an element in a caravan graph
type Node struct {
	Ascendants      map[Key]*Node
	Descendants     map[Key]*Node
	Element         Keyer
	WeakDescendants map[Key]*Node
}

type Key int

// Keyer is the interface by which an element in a graph exposes its key
type Keyer interface {
	Key() Key
	String() string
}

// CreateCaravan returns an initialized caravan struct
func CreateCaravan() *Caravan {
	return &Caravan{
		nodes: map[Key]*Node{},
		roots: map[Key]*Node{},
	}
}

// Ensure will look up a key.  If there is a match in the caravan, that node
// is returned.  Otherwise, the create function is invoked.  The results of
// that are put into the caravan, and the resulting node is returned.  The
// returned bool is true if the node already existed, and false if the create
// method was invoked.
func (c *Caravan) Ensure(key Key, create func() Keyer) (*Node, bool) {
	c.m.Lock()
	n, ok := c.nodes[key]
	if !ok {
		newP := create()
		n = c.insert(newP)
	}
	c.m.Unlock()

	return n, !ok
}

// Find returns the element with the given key
func (c *Caravan) Find(key Key) (*Node, bool) {
	c.m.Lock()
	n, ok := c.nodes[key]
	c.m.Unlock()
	if !ok {
		return nil, false
	}
	return n, true
}

// Insert adds an element to the caravan at the root level
func (c *Caravan) Insert(p Keyer) {
	key := p.Key()
	c.m.Lock()
	if _, ok := c.nodes[key]; ok {
		// Node already exists.
		c.m.Unlock()
		return
	}

	c.insert(p)

	c.m.Unlock()
}

func (c *Caravan) insert(p Keyer) *Node {
	key := p.Key()
	n := &Node{
		Ascendants:      map[Key]*Node{},
		Descendants:     map[Key]*Node{},
		Element:         p,
		WeakDescendants: map[Key]*Node{},
	}
	c.nodes[key] = n
	c.roots[key] = n

	return n
}

// Connect establishes an edge between two elements
func (c *Caravan) Connect(from, to Keyer) error {
	fromKey, toKey := from.Key(), to.Key()

	var ok bool
	var fromNode, toNode *Node

	c.m.Lock()

	fromNode, ok = c.nodes[fromKey]
	if !ok {
		c.m.Unlock()
		return errors.New("Element `from` not in caravan")
	}

	toNode, ok = c.nodes[toKey]
	if !ok {
		c.m.Unlock()
		return errors.New("Element `to` not in caravan")
	}

	if _, ok = fromNode.Descendants[toKey]; ok {
		// The connection was already made; just accept it.
		c.m.Unlock()
		return nil
	}

	checkedNodes := map[Key]bool{}
	err := checkLoop(fromKey, toNode, checkedNodes)
	if err != nil {
		c.m.Unlock()
		return fmt.Errorf("Connect would create circular loop:\n\t%s", err.Error())
	}

	if _, ok := c.roots[toKey]; ok {
		delete(c.roots, toKey)
	}

	fromNode.Descendants[toKey] = toNode
	toNode.Ascendants[fromKey] = fromNode

	c.m.Unlock()
	return nil
}

func (c *Caravan) WeakConnect(from, to Keyer) error {
	fromKey, toKey := from.Key(), to.Key()

	var ok bool
	var fromNode, toNode *Node

	c.m.Lock()

	fromNode, ok = c.nodes[fromKey]
	if !ok {
		c.m.Unlock()
		return errors.New("Element `from` not in caravan")
	}

	toNode, ok = c.nodes[toKey]
	if !ok {
		c.m.Unlock()
		return errors.New("Element `to` not in caravan")
	}

	if _, ok := c.roots[toKey]; ok {
		delete(c.roots, toKey)
	}

	if _, ok := fromNode.Descendants[toKey]; !ok {
		fromNode.WeakDescendants[toKey] = toNode
	}

	c.m.Unlock()
	return nil
}

func checkLoop(fromKey Key, n *Node, checkedNodes map[Key]bool) error {
	key := n.Element.Key()
	if fromKey == key {
		return fmt.Errorf("Found loop:\n\t%s", n.Element)
	}

	for k, v := range n.Descendants {
		if _, ok := checkedNodes[k]; ok {
			continue
		}
		if err := checkLoop(fromKey, v, checkedNodes); err != nil {
			return fmt.Errorf("%s\n\t%s", err.Error(), n.Element)
		}
	}

	checkedNodes[key] = true

	return nil
}

// Iter provides a channel used for iterating over all nodes.
// Example:
// c := NewCaravan()
// // ...
// for i := range c.Iter() {
//	// ...
// }
func (c *Caravan) Iter(iter func(Key, *Node) bool) {
	c.m.Lock()

	for k, n := range c.nodes {
		if !iter(k, n) {
			break
		}
	}

	c.m.Unlock()
}

// Walk will traverse the caravan structure, calling the provided `walker`
// function at every node.  The `direction` parameter indicates whether the
// traversal will start at the top (roots) or at the bottom (leaves).  When
// walking down, all ascendents of a node will be walked before that node.
// Conversely, when walking up, all descendents of a node will be walked before
// that node.  Walk will visit each node exactly once.
//
// Note that because the caravan may be disjointed, the walk function may
// start at a leaf or root, pass through some interior nodes, then return
// to a different root or leaf.
func (c *Caravan) Walk(direction WalkDirection, walker CaravanWalker) {
	visits := map[Key]bool{}

	c.m.Lock()

	if direction == WalkDown {
		for _, v := range c.roots {
			c.walkNodeDown(visits, v, walker)
		}
	} else {
		for _, v := range c.roots {
			c.walkNodeUp(visits, v, walker)
		}
	}

	c.m.Unlock()
}

func (c *Caravan) walkNodeDown(visits map[Key]bool, node *Node, walker CaravanWalker) {
	for k := range node.Ascendants {
		if _, ok := visits[k]; !ok {
			// An ascendent hasn't been visited; can't process this node yet.
			return
		}
	}

	visits[node.Element.Key()] = true

	walker(node)

	for k, v := range node.Descendants {
		if _, ok := visits[k]; ok {
			// This node was already visited
			continue
		}
		c.walkNodeDown(visits, v, walker)
	}
}

func (c *Caravan) walkNodeUp(visits map[Key]bool, node *Node, walker CaravanWalker) {
	visits[node.Element.Key()] = true

	for k, v := range node.Descendants {
		if _, ok := visits[k]; ok {
			// This node was already visited
			continue
		}
		c.walkNodeUp(visits, v, walker)
	}

	for k, v := range node.WeakDescendants {
		if _, ok := visits[k]; ok {
			continue
		}
		c.walkNodeUp(visits, v, walker)
	}

	// for _, v := range node.Descendants {
	// 	if _, ok := visits[v.Element.Key()]; !ok {
	// 		// A descendant hasn't been visited; can't process this node yet.
	// 		return
	// 	}
	// }

	walker(node)
}
