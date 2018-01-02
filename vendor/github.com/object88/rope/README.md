# rope

Implementation of the [rope data structure](https://en.wikipedia.org/wiki/Rope_(data_structure)) in Go.  This library can be used by a text editor to improve performance as the user is making ad-hoc changes at arbirary locations in the text.

The implementation is based on work in the [rope-experiment](https://github.com/object88/rope-experiment) repository, where different implentations are tested and benchmarked against each other.  This implementation is `V2`.  This implementation may change over time if further enhancements are realized.

Performance differences vs. plain string operations are published in the rope-experiment repository.
