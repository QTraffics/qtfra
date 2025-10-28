package values

import (
	"cmp"
)

func UseDefaultNil[T any](real, dft T) T {
	if IsNil(real) {
		return dft
	}
	return real
}

func UseDefault[T comparable](real, dft T) T {
	zero := Zero[T]()
	if zero == real {
		return dft
	}
	return real
}

func UseDefaultIF[T any](real, dft T, IF func(v T) bool) T {
	if IF(real) {
		return dft
	}
	return real
}

func UseBetween[T cmp.Ordered](real, mn, mx T) T {
	if real < mn {
		return mn
	}
	if real > mx {
		return mx
	}
	return real
}
