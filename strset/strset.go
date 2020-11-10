package strset

var exists = struct{}{}

// Set is simple set for strings
type Set struct {
	m map[string]struct{}
}

// NewSet creates a new *set
func NewSet() *Set {
	s := &Set{}
	s.m = make(map[string]struct{})
	return s
}

// Add adds a new value to the set
func (s *Set) Add(value string) {
	s.m[value] = exists
}

// Remove removes a value from the set
func (s *Set) Remove(value string) {
	delete(s.m, value)
}

// Contains checks to see if a value exists in the set
func (s *Set) Contains(value string) bool {
	_, c := s.m[value]
	return c
}

// Len returns the length of the set
func (s *Set) Len() int {
	return len(s.m)
}
