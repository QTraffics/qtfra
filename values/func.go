package values

type FilterFunc[T any] func(v T) bool

func (f FilterFunc[T]) Filter(v T) bool {
	return f(v)
}
