package ex

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Must0[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Only[T any](_ T, err error) error {
	return err
}
