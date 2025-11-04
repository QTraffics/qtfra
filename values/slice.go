package values

type Slice[T any] struct {
	vv []T
}

func NewSlice[T any](vv ...T) *Slice[T] {
	s := &Slice[T]{vv: vv}
	return s
}

func (s *Slice[T]) Append(v ...T) {
	s.vv = append(s.vv, v...)
}

func (s *Slice[T]) Slice() []T {
	return s.vv
}
