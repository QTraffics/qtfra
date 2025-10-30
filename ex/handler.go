package ex

type Handler interface {
	NewError(err error)
}

type FuncHandler func(err error)

func (h FuncHandler) NewError(err error) {
	h(err)
}

type JoinError struct {
	e error
}

func (j *JoinError) NewError(err error) {
	j.e = Errors(j.e, err)
}

func (j *JoinError) Error() string {
	return j.e.Error()
}
