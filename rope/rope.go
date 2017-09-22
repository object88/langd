package rope

import "fmt"

const (
	splitLength = 512
	joinLength  = 256

	rebalanceRatio = 1.2
)

// Rope is a data structure composed of small strings that acts like a
// contiguous long string.
// https://en.wikipedia.org/wiki/Rope_(data_structure)
type Rope struct {
	right  *Rope
	left   *Rope
	value  *string
	length int
}

// CreateRope returns a new Rope struct with the given initial value
func CreateRope(initial string) *Rope {
	r := &Rope{nil, nil, &initial, len(initial)}
	r.adjust()
	return r
}

func (r *Rope) Insert(position int, value string) error {
	if r == nil {
		return fmt.Errorf("Nil pointer receiver")
	}

	if position < 0 || position > r.length {
		return fmt.Errorf("position is not within rope bounds")
	}

	return r.insert(position, value)
}

// Rebalance rebalances the b-tree structure
func (r *Rope) Rebalance() {
	if r.value == nil {
		leftLength := r.left.length
		rightLength := r.right.length

		if float32(leftLength)/float32(rightLength) > rebalanceRatio ||
			float32(rightLength)/float32(leftLength) > rebalanceRatio {
			r.rebuild()
		} else {
			r.left.Rebalance()
			r.right.Rebalance()
		}
	}
}

func (r *Rope) Remove(start, end int) error {
	if r == nil {
		return fmt.Errorf("Nil pointer receiver")
	}

	if start < 0 || start > r.length {
		return fmt.Errorf("Start is not within rope bounds")
	}
	if end < 0 || end > r.length {
		return fmt.Errorf("End is not within rope bounds")
	}
	if start > end {
		return fmt.Errorf("Start is greater than end")
	}

	return r.remove(start, end)
}

func (r *Rope) String() string {
	if r.value != nil {
		return *r.value
	}
	return r.left.String() + r.right.String()
}

func (r *Rope) adjust() {
	if r.value != nil {
		if r.length > splitLength {
			divide := r.length >> 1
			r.left = CreateRope((*r.value)[:divide])
			r.right = CreateRope((*r.value)[divide:])
			r.value = nil
		}
	} else {
		if r.length < joinLength {
			v := r.left.String() + r.right.String()
			r.value = &v
			r.left = nil
			r.right = nil
		}
	}
}

func (r *Rope) insert(position int, value string) error {
	if r.value != nil {
		v := (*r.value)[0:position] + value + (*r.value)[position:]
		r.value = &v
		r.length = len(*r.value)
	} else {
		leftLength := r.left.length
		if position < leftLength {
			r.left.insert(position, value)
			r.length = r.left.length + r.right.length
		} else {
			r.right.insert(position-leftLength, value)
		}
	}
	r.adjust()
	return nil
}

func (r *Rope) rebuild() {
	if r.value == nil {
		v := r.left.String() + r.right.String()
		r.value = &v
		r.left = nil
		r.right = nil
		r.adjust()
	}
}

func (r *Rope) remove(start, end int) error {
	if r.value != nil {
		v := (*r.value)[0:start] + (*r.value)[end:]
		r.value = &v
		r.length = len(*r.value)
	} else {
		leftLength := r.left.length
		leftStart := min(start, leftLength)
		leftEnd := min(end, leftLength)
		rightLength := r.right.length
		rightStart := max(0, min(start-leftLength, rightLength))
		rightEnd := max(0, min(end-leftLength, rightLength))
		if leftStart < leftLength {
			r.left.remove(leftStart, leftEnd)
		}
		if rightEnd > 0 {
			r.right.remove(rightStart, rightEnd)
		}
		r.length = r.left.length + r.right.length
	}

	r.adjust()
	return nil
}

func (r *Rope) substring(start, end int) string {
	if end == -1 {
		end = r.length
	}
	if start < 0 {
		start = 0
	} else if start > r.length {
		start = r.length
	}
	if end < 0 {
		end = 0
	} else if end > r.length {
		end = r.length
	}

	if r.value != nil {
		return (*r.value)[start:end]
	}

	leftLength := r.left.length
	leftStart := min(start, leftLength)
	leftEnd := min(end, leftLength)
	rightLength := r.right.length
	rightStart := max(0, min(start-leftLength, rightLength))
	rightEnd := max(0, min(end-leftLength, rightLength))

	if leftStart != leftEnd {
		if rightStart != rightEnd {
			return r.left.substring(leftStart, leftEnd) + r.right.substring(rightStart, rightEnd)
		}

		return r.left.substring(leftStart, leftEnd)
	}

	if rightStart != rightEnd {
		return r.right.substring(rightStart, rightEnd)
	}

	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
