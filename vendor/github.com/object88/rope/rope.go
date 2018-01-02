package rope

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// This code is a mostly-direct translation of
// https://github.com/component/rope.  Many thanks to the contributers and
// maintainers of http://component.github.io/ for their unknown contributions
// to this project.

const (
	splitLength = 512
	joinLength  = 256

	rebalanceRatio = 1.2
)

// Rope is a data structure that represents a string.  Internally, the string
// is fractured into smaller strings to facilitate faster editing at the
// expense of memory usage
type Rope struct {
	right      *Rope
	left       *Rope
	value      *string
	length     int
	byteLength int
}

// CreateRope creates a Rope with the given initial value
func CreateRope(initial string) *Rope {
	r := &Rope{nil, nil, &initial, utf8.RuneCountInString(initial), len(initial)}
	r.adjust()
	return r
}

func (r *Rope) Alter(start, end int, value string) error {
	if r == nil {
		return fmt.Errorf("Nil pointer receiver")
	}

	if start < 0 || start > r.length {
		return fmt.Errorf("start is not within rope bounds")
	}

	if end < 0 || end > r.length {
		return fmt.Errorf("end is not within rope bounds")
	}

	if start > end {
		return fmt.Errorf("start is after end")
	}

	if start == end {
		// This is a pure insert
		if value == "" {
			// No-op; nothing to insert
			return nil
		}

		return r.insert(start, value)

	} else if value == "" {
		// This is a pure remove
		return r.remove(start, end)
	}

	return r.alter(start, end, value)
}

// ByteLength returns the number of bytes necessary to store a contiguous
// representation of the Rope's contents
func (r *Rope) ByteLength() int {
	return r.byteLength
}

// Insert adds the provided value to the rope at the given rune-offset
// position
func (r *Rope) Insert(position int, value string) error {
	if r == nil {
		return fmt.Errorf("Nil pointer receiver")
	}

	if position < 0 || position > r.length {
		return fmt.Errorf("position is not within rope bounds")
	}

	return r.insert(position, value)
}

// Length returns the number of runes in the Rope
func (r *Rope) Length() int {
	return r.length
}

// NewReader returns an `io.Reader` that will allow consuming the rope as a
// contiguous stream of bytes.
func (r *Rope) NewReader() io.Reader {
	return &Reader{0, r}
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

// Remove deletes the runes between the start and end point.  The start
// and end are the rune offsets from the start of the rope.
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
	var buf bytes.Buffer
	buf.Grow(r.byteLength)
	read := r.NewReader()
	io.Copy(&buf, read)
	return string(buf.Bytes())
}

func (r *Rope) adjust() {
	if r.value != nil {
		if r.length > splitLength {
			divide := r.length >> 1
			offset := r.findByteOffsets(divide)
			r.left = CreateRope((*r.value)[:offset])
			r.right = CreateRope((*r.value)[offset:])
			r.value = nil
		}
	} else {
		if r.length < joinLength {
			r.join()
		}
	}
}

func (r *Rope) alter(start, end int, value string) error {
	valueLength := utf8.RuneCountInString(value)
	valueByteLength := len(value)

	if r.value != nil {
		var buf bytes.Buffer
		byteStart := r.findByteOffsets(start)
		byteEnd := r.findByteOffsets(end)
		buf.Grow(len(*r.value) - byteEnd + byteStart + valueByteLength)
		buf.WriteString((*r.value)[0:byteStart])
		buf.WriteString(value)
		buf.WriteString((*r.value)[byteEnd:])
		s := buf.String()
		r.value = &s
		r.byteLength -= byteEnd - byteStart - valueByteLength
		r.length -= end - start - valueLength
	} else {
		leftLength := r.left.length
		leftStart := min(start, leftLength)
		rightLength := r.right.length
		rightEnd := max(0, min(end-leftLength, rightLength))

		valueCutoff := findByteOffset(value, min(valueLength, leftLength-leftStart))

		if leftStart < leftLength {
			leftEnd := min(end, leftLength)
			r.left.alter(leftStart, leftEnd, value[:valueCutoff])
		}
		if rightEnd > 0 || valueCutoff < valueByteLength {
			rightStart := max(0, min(start-leftLength, rightLength))
			valueStart := findByteOffset(value, min(valueLength, leftLength-leftStart))
			r.right.alter(rightStart, rightEnd, value[valueStart:])
		}
		r.byteLength = r.left.byteLength + r.right.byteLength
		r.length = r.left.length + r.right.length
	}

	r.adjust()
	return nil
}

func (r *Rope) findByteOffsets(position int) int {
	offset := 0
	rs := []rune(*r.value)

	for i := 0; i < position; i++ {
		offset += utf8.RuneLen(rs[i])
	}

	return offset
}

func (r *Rope) insert(position int, value string) error {
	if r.value != nil {
		var buf bytes.Buffer
		offset := r.findByteOffsets(position)
		valueLength := utf8.RuneCountInString(value)
		valueBytesLength := len(value)
		buf.Grow(r.byteLength + valueBytesLength)
		buf.WriteString((*r.value)[0:offset])
		buf.WriteString(value)
		buf.WriteString((*r.value)[offset:])
		s := buf.String()
		r.value = &s
		r.byteLength += valueBytesLength
		r.length += valueLength
	} else {
		leftLength := r.left.length
		if position < leftLength {
			r.left.insert(position, value)
		} else {
			r.right.insert(position-leftLength, value)
		}
		r.byteLength = r.left.byteLength + r.right.byteLength
		r.length = r.left.length + r.right.length
	}
	r.adjust()
	return nil
}

func (r *Rope) join() {
	c := r.left.byteLength + r.right.byteLength
	var buf bytes.Buffer
	buf.Grow(c)
	io.Copy(&buf, r.left.NewReader())
	io.Copy(&buf, r.right.NewReader())
	s := buf.String()
	r.value = &s
	r.left = nil
	r.right = nil
}

func (r *Rope) locate(position int) (*Rope, int) {
	if r.value != nil {
		return r, position
	}

	leftLength := r.left.length
	if position < leftLength {
		return r.left.locate(position)
	}

	return r.right.locate(position - leftLength)
}

func (r *Rope) rebuild() {
	if r.value == nil {
		r.join()
		r.adjust()
	}
}

func (r *Rope) remove(start, end int) error {
	if r.value != nil {
		var buf bytes.Buffer
		byteStart := r.findByteOffsets(start)
		byteEnd := r.findByteOffsets(end)
		buf.Grow(len(*r.value) - byteEnd + byteStart)
		buf.WriteString((*r.value)[0:byteStart])
		buf.WriteString((*r.value)[byteEnd:])
		s := buf.String()
		r.value = &s
		r.byteLength -= byteEnd - byteStart
		r.length -= end - start
	} else {
		leftLength := r.left.length
		leftStart := min(start, leftLength)
		rightLength := r.right.length
		rightEnd := max(0, min(end-leftLength, rightLength))
		if leftStart < leftLength {
			leftEnd := min(end, leftLength)
			r.left.remove(leftStart, leftEnd)
		}
		if rightEnd > 0 {
			rightStart := max(0, min(start-leftLength, rightLength))
			r.right.remove(rightStart, rightEnd)
		}
		r.byteLength = r.left.byteLength + r.right.byteLength
		r.length = r.left.length + r.right.length
	}

	r.adjust()
	return nil
}

// Reader implements io.Reader and io.WriterTo for a Rope rope
type Reader struct {
	pos int
	r   *Rope
}

func (read *Reader) Read(p []byte) (n int, err error) {
	if read.pos == read.r.length {
		return 0, io.EOF
	}

	node, offset := read.r.locate(read.pos)

	copied := copy(p, []byte(string([]rune(*node.value)[offset:])))
	read.pos += copied
	return copied, nil
}

// WriteTo writes the contents of a Rope to the provided io.Writer
func (read *Reader) WriteTo(w io.Writer) (int64, error) {
	n, err := read.writeNodeTo(read.r, w)
	return int64(n), err
}

func (read *Reader) writeNodeTo(r *Rope, w io.Writer) (int, error) {
	if r.value != nil {
		copied, err := io.WriteString(w, *r.value)
		if copied != len(*r.value) && err == nil {
			err = io.ErrShortWrite
		}
		return copied, err
	}

	var err error
	var n, m int

	n, err = read.writeNodeTo(r.left, w)
	if err != nil {
		return n, err
	}

	m, err = read.writeNodeTo(r.right, w)

	return int(n + m), err
}

func findByteOffset(s string, position int) int {
	offset := 0
	for i := 0; i < position; i++ {
		r, n := utf8.DecodeRuneInString(s[offset:])
		if r == utf8.RuneError {
			return -1
		}
		offset += n
	}
	return offset
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
