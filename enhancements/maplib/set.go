package maplib

type Set[K comparable] map[K]bool

func NewSet[E comparable]() Set[E] {
	return make(Set[E])
}

func NewSetFromSlice[S ~[]E, E comparable](s S) Set[E] {
	set := NewSet[E]()
	for i := 0; i < len(s); i++ {
		set.Add(s[i])
	}
	return set
}

func (s Set[K]) Add(k K) {
	s[k] = true
}

func (s Set[K]) Del(k K) {
	delete(s, k)
}

func (s Set[K]) Contains(k K) bool {
	_, ok := s[k]
	return ok
}

func (s Set[K]) ContainAll(S []K) bool {
	contain := true
	for i := 0; i < len(S) && contain; i++ {
		contain = s.Contains(S[i])
	}
	return contain
}

func Contain[K comparable, V any](m map[K]V, k K) bool {
	if m == nil {
		return false
	}
	_, ok := m[k]
	return ok
}
