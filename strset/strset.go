package strset

var exists = struct{}{}

// Set is simple set for strings
type Set struct {
	m map[string]struct{}
}

// NewSet creates a new *set
func NewSet() *Set {
	return &Set{m: make(map[string]struct{})}
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

// Range lets use range over the struct values
func (s *Set) Range() <-chan string {
	chnl := make(chan string)
	go func() {
		for v := range s.m {
			chnl <- v
		}

		// Ensure that at the end of the loop we close the channel!
		close(chnl)
	}()
	return chnl
}
