package collections

import "errors"

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
type CaravanWalker func(k Keyer, isRoot, isLeaf bool)

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
}

// Node is an element in a caravan graph
type Node struct {
	ascendants  map[Key]*Node
	descendants map[Key]*Node
	k           Keyer
}

// Key is some unique identifier for an element in the graph
type Key string

// Keyer is the interface by which an element in a graph exposes its key
type Keyer interface {
	Key() Key
}

// CreateCaravan returns an initialized caravan struct
func CreateCaravan() *Caravan {
	return &Caravan{
		nodes: map[Key]*Node{},
		roots: map[Key]*Node{},
	}
}

// Find returns the element with the given key
func (c *Caravan) Find(key Key) (Keyer, bool) {
	n, ok := c.nodes[key]
	if !ok {
		return nil, false
	}
	return n.k, true
}

// Insert adds an element to the catavan at the root level
func (c *Caravan) Insert(k Keyer) {
	key := k.Key()
	if _, ok := c.nodes[key]; ok {
		// Node already exists.
		return
	}

	n := &Node{
		ascendants:  map[Key]*Node{},
		descendants: map[Key]*Node{},
		k:           k,
	}
	c.nodes[key] = n
	c.roots[key] = n
}

// Connect establishes an edge between two elements
func (c *Caravan) Connect(from, to Keyer) error {
	fromKey, toKey := from.Key(), to.Key()

	var ok bool
	var fromNode, toNode *Node

	fromNode, ok = c.nodes[fromKey]
	if !ok {
		return errors.New("Element `from` not in caravan")
	}

	toNode, ok = c.nodes[toKey]
	if !ok {
		return errors.New("Element `to` not in caravan")
	}

	if _, ok = fromNode.descendants[toKey]; ok {
		return errors.New("`to` is already a direct descendent of `from`")
	}

	circular := false
	visited := map[Key]bool{}
	c.walkNodeDown(visited, toNode, func(k Keyer, isRoot, isLeaf bool) {
		if fromNode.k == k {
			circular = true
		}
	})
	if circular {
		return errors.New("Connect would create circular loop")
	}

	if _, ok := c.roots[toKey]; ok {
		delete(c.roots, toKey)
	}

	fromNode.descendants[toKey] = toNode
	toNode.ascendants[fromKey] = fromNode

	return nil
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

	if direction == WalkDown {
		for _, v := range c.roots {
			c.walkNodeDown(visits, v, walker)
		}
	} else {
		for _, v := range c.roots {
			c.walkNodeUp(visits, v, walker)
		}
	}
}

func (c *Caravan) walkNodeDown(visits map[Key]bool, node *Node, walker CaravanWalker) {
	for _, v := range node.ascendants {
		if _, ok := visits[v.k.Key()]; !ok {
			// An ascendent hasn't been visited; can't process this node yet.
			return
		}
	}

	isRoot := len(node.ascendants) == 0
	isLeaf := len(node.descendants) == 0

	visits[node.k.Key()] = true

	walker(node.k, isRoot, isLeaf)

	for _, v := range node.descendants {
		if _, ok := visits[v.k.Key()]; ok {
			// This node was already visited
			continue
		}
		c.walkNodeDown(visits, v, walker)
	}
}

func (c *Caravan) walkNodeUp(visits map[Key]bool, node *Node, walker CaravanWalker) {
	isRoot := len(node.ascendants) == 0
	isLeaf := len(node.descendants) == 0

	visits[node.k.Key()] = true

	for _, v := range node.descendants {
		if _, ok := visits[v.k.Key()]; ok {
			// This node was already visited
			continue
		}
		c.walkNodeUp(visits, v, walker)
	}

	for _, v := range node.descendants {
		if _, ok := visits[v.k.Key()]; !ok {
			// An ascendent hasn't been visited; can't process this node yet.
			return
		}
	}

	walker(node.k, isRoot, isLeaf)
}
