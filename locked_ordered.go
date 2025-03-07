package set

import (
	"iter"
	"sync"
)

// LockedOrdered is a set implementation that maintains the order of elements and is safe for concurrent use. If the
// same item is added multiple times, the first insertion determines the order.
type LockedOrdered[M comparable] struct {
	*sync.Cond
	iterating bool
	idx       map[M]int
	values    []M
}

var _ Set[int] = new(LockedOrdered[int])

// NewLockedOrdered returns an empty LockedOrderedSet instance.
func NewLockedOrdered[M comparable]() *LockedOrdered[M] {
	return &LockedOrdered[M]{
		idx:    make(map[M]int),
		values: make([]M, 0),
		Cond:   sync.NewCond(&sync.Mutex{}),
	}
}

// NewLockedOrderedFrom returns a new LockedOrderedSet instance filled with the values from the sequence.
func NewLockedOrderedFrom[M comparable](seq iter.Seq[M]) *LockedOrdered[M] {
	s := NewLockedOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *LockedOrdered[M]) Contains(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	return s.contains(m)
}

func (s *LockedOrdered[M]) Clear() int {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	n := len(s.values)
	for k := range s.idx {
		delete(s.idx, k)
	}
	s.values = s.values[:0]
	return n
}

func (s *LockedOrdered[M]) contains(m M) bool {
	_, ok := s.idx[m]
	return ok
}

func (s *LockedOrdered[M]) Add(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	if s.contains(m) {
		return false
	}
	s.values = append(s.values, m)
	s.idx[m] = len(s.values) - 1
	return true
}

func (s *LockedOrdered[M]) Remove(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	if !s.contains(m) {
		return false
	}
	d := s.idx[m]
	s.values = append(s.values[:d], s.values[d+1:]...)
	for i, v := range s.values[d:] {
		s.idx[v] = d + i
	}
	delete(s.idx, m)
	return true
}

func (s *LockedOrdered[M]) Cardinality() int {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	return len(s.values)
}

func (s *LockedOrdered[M]) Iterator(yield func(M) bool) {
	s.Cond.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Cond.Broadcast()
		s.Cond.L.Unlock()
	}()

	for _, k := range s.values {
		if !yield(k) {
			return
		}
	}
}

func (s *LockedOrdered[M]) Clone() Set[M] {
	return NewLockedOrderedFrom(s.Iterator)
}
