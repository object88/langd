package rope

import (
	"strings"
	"testing"
)

func Test_CreateRope(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"123", "123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := CreateRope(tc.value)
			if r == nil {
				t.Fatal("got nil")
			}
			if r.length != len(tc.value) {
				t.Fatalf("got length %d", r.length)
			}
			if r.value == nil {
				t.Fatal("Got nil value")
			}
			if *r.value != tc.value {
				t.Fatalf("Got mangled value: '%s'", tc)
			}
			if r.left != nil {
				t.Fatal("got non-nil left")
			}
			if r.right != nil {
				t.Fatal("got non-nil right")
			}
		})
	}
}

func Test_CreateRope_Large(t *testing.T) {
	s := strings.Repeat("0123456789", 60)

	r := CreateRope(s)
	if r == nil {
		t.Fatal("got nil")
	}
	if r.length != len(s) {
		t.Fatalf("Total length is incorrect: %d", r.length)
	}
	if r.value != nil {
		t.Fatal("Got non-nil value at root")
	}
	if r.left == nil {
		t.Fatal("Got nil left")
	}
	if r.right == nil {
		t.Fatal("Got nil right")
	}
	if r.String() != s {
		t.Fatal("Got incorrect string")
	}
}

func Test_Rebalance(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"123", "123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := CreateRope(tc.value)
			r.Rebalance()
			if r == nil {
				t.Fatal("got nil")
			}
			if r.length != len(tc.value) {
				t.Fatalf("got length %d", r.length)
			}
			if r.value == nil {
				t.Fatal("Got nil value")
			}
			if *r.value != tc.value {
				t.Fatalf("Got mangled value: '%s'", tc)
			}
			if r.left != nil {
				t.Fatal("got non-nil left")
			}
			if r.right != nil {
				t.Fatal("got non-nil right")
			}
		})
	}
}

func Test_Insert(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		position int
		addition string
		success  bool
		result   string
	}{
		{"empty", "", 0, "abc", true, "abc"},
		{"insert to start", "123", 0, "abc", true, "abc123"},
		{"insert to middle", "123", 2, "abc", true, "12abc3"},
		{"insert to end", "123", 3, "abc", true, "123abc"},
		{"insert to past end", "123", 4, "abc", false, "123"},
		{"insert to before start", "123", -1, "abc", false, "123"},
	}

	for _, tc := range tests {
		r := CreateRope(tc.value)
		err := r.Insert(tc.position, tc.addition)
		if tc.success && err != nil {
			t.Fatal("Did not expect failure, but got err")
		}
		if !tc.success && err == nil {
			t.Fatal("Expected failure, but did not get err")
		}
		if r.String() != tc.result {
			t.Fatalf("Expected '%s', got '%s'", tc.result, r.String())
		}
	}
}

func Test_Remove(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		start   int
		end     int
		success bool
		result  string
	}{
		{"remove from empty", "", 0, 0, true, ""},
		{"remove from start", "abcd", 0, 1, true, "bcd"},
		{"remove from middle", "abcd", 1, 2, true, "acd"},
		{"remove from end", "abcd", 3, 4, true, "abc"},
		{"remove past end", "abcd", 3, 5, false, "abcd"},
		{"remove before start", "abcd", -1, 1, false, "abcd"},
		{"remove nothing", "abcd", 2, 2, true, "abcd"},
	}

	for _, tc := range tests {
		r := CreateRope(tc.value)
		err := r.Remove(tc.start, tc.end)
		if tc.success && err != nil {
			t.Fatal("Did not expect failure, but got err")
		}
		if !tc.success && err == nil {
			t.Fatal("Expected failure, but did not get err")
		}
		if r.String() != tc.result {
			t.Fatalf("Expected '%s', got '%s'", tc.result, r.String())
		}
	}
}

func Test_Remove_Large(t *testing.T) {
	s600 := strings.Repeat("0123456789", 60)
	tests := []struct {
		name     string
		value    string
		start    int
		end      int
		success  bool
		expected string
		joined   bool
	}{
		{"remove one from start", s600, 0, 1, true, s600[1:], false},
		{"remove one from left middle", s600, 1, 2, true, s600[0:1] + s600[2:], false},
		{"remove one from end", s600, 599, 600, true, s600[0:599], false},
	}

	for _, tc := range tests {
		r := CreateRope(tc.value)
		err := r.Remove(tc.start, tc.end)
		if tc.success && err != nil {
			t.Fatal("Did not expect failure, but got err")
		}
		if !tc.success && err == nil {
			t.Fatal("Expected failure, but did not get err")
		}
		if r.String() != tc.expected {
			t.Fatalf("Expected '%s', got '%s'", tc.expected, r.String())
		}
		if r.length != len(tc.expected) {
			t.Fatalf("Expected length %d, got %d", len(tc.expected), len(r.String()))
		}
		if tc.joined {
			if r.left != nil || r.right != nil {
				t.Fatal("Expected to not have left and right pointers")
			}
			if r.value == nil {
				t.Fatal("Expected to have value pointer")
			}
		} else {
			if r.left == nil || r.right == nil {
				t.Fatal("Expected to have left and right pointers")
			}
			if r.value != nil {
				t.Fatal("Expected to not have value pointer")
			}
		}
	}
}
