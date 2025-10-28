package slicelib

func Map[T any, N any](arr []T, block func(it T) N) []N {
	retArr := make([]N, 0, len(arr))
	for index := range arr {
		retArr = append(retArr, block(arr[index]))
	}
	return retArr
}

func MapToAny[T any](arr []T) []any {
	return Map(arr, func(it T) any {
		return it
	})
}

func Filter[T any](arr []T, block func(it T) bool) []T {
	var retArr []T
	for _, it := range arr {
		if block(it) {
			retArr = append(retArr, it)
		}
	}
	return retArr
}

func FilterNotNil[T any](arr []T) []T {
	return Filter(arr, func(it T) bool {
		var anyIt any = it
		return anyIt != nil
	})
}

func FilterNotDefault[T comparable](arr []T) []T {
	var defaultValue T
	return Filter(arr, func(it T) bool {
		return it != defaultValue
	})
}

func FilterIndexed[T any](arr []T, block func(index int, it T) bool) []T {
	var retArr []T
	for index, it := range arr {
		if block(index, it) {
			retArr = append(retArr, it)
		}
	}
	return retArr
}

func Uniq[T comparable](arr []T) []T {
	result := make([]T, 0, len(arr))
	seen := make(map[T]struct{}, len(arr))

	for _, item := range arr {
		if _, ok := seen[item]; ok {
			continue
		}

		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}

func UniqBy[T any, C comparable](arr []T, block func(it T) C) []T {
	result := make([]T, 0, len(arr))
	seen := make(map[C]struct{}, len(arr))

	for _, item := range arr {
		c := block(item)
		if _, ok := seen[c]; ok {
			continue
		}

		seen[c] = struct{}{}
		result = append(result, item)
	}

	return result
}

func UniqByLast[S ~[]T, T any, C comparable](arr S, block func(it T) C) S {
	result := make([]T, 0, len(arr))
	seen := make(map[C]struct{}, len(arr))
	for i := len(arr) - 1; i >= 0; i-- {
		c := block(arr[i])
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		result = append(result, arr[i])
	}

	// reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return S(result)
}
