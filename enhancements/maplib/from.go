package maplib

func IndexMap[S ~[]E, E comparable](s S) map[E]int {
	ret := make(map[E]int)

	for i, v := range s {
		ret[v] = i
	}
	return ret
}

func FromSliceFunc[K comparable, S ~[]E, E any](s S, fn func(index int, value E) K) map[K]E {
	if len(s) == 0 {
		return make(map[K]E)
	}

	ret := make(map[K]E, len(s))

	for i, v := range s {
		k := fn(i, v)
		ret[k] = v
	}
	return ret
}
