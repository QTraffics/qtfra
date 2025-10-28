package maplib

func Merge[K comparable, V any](destination map[K]V, source map[K]V) map[K]V {
	if destination == nil {
		return source
	}
	if source == nil {
		return destination
	}
	for k, v := range source {
		destination[k] = v
	}
	return destination
}

func Merge0[K comparable, S ~[]E, E any](destination map[K]S, source map[K]S) map[K]S {
	if destination == nil {
		return source
	}
	if source == nil {
		return destination
	}
	for k, v := range source {
		vv := make(S, len(v), cap(v))
		copy(vv, v)
		destination[k] = vv
	}
	return destination
}

func Copy[K comparable, V any](source map[K]V) map[K]V {
	if source == nil {
		return source
	}
	dup := make(map[K]V, len(source))
	for k, v := range source {
		dup[k] = v
	}
	return dup
}

func Copy0[K comparable, S ~[]E, E any](source map[K]S) map[K]S {
	if source == nil {
		return source
	}

	dup := make(map[K]S, len(source))
	for k, v := range source {
		vv := make(S, len(v), cap(v))
		copy(vv, v)
		dup[k] = vv
	}
	return dup
}
