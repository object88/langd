package langd

// nextPowerOfTwo takes a positive integer and finds the next greater number
// that's a power of two.  Code directly copied from
// http://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
func nextPowerOfTwo(size int) int {
	size--
	size |= size >> 1
	size |= size >> 2
	size |= size >> 4
	size |= size >> 8
	size |= size >> 16
	size |= size >> 32
	size++
	return size
}
