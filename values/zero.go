package values

import "reflect"

func IsNil(v any) bool {
	anyv := v
	if anyv == nil {
		return true
	}
	rfv := reflect.ValueOf(v)
	return (rfv.Kind() == reflect.Ptr ||
		rfv.Kind() == reflect.Interface ||
		rfv.Kind() == reflect.Slice ||
		rfv.Kind() == reflect.Map ||
		rfv.Kind() == reflect.Chan ||
		rfv.Kind() == reflect.Func) && rfv.IsNil()
}

func ZeroValue[T any](v T) T {
	return Zero[T]()
}

func Zero[T any]() T {
	var vv T
	return vv
}
