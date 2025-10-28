package ex

type Handler interface {
	NewError(err error)
}

type FuncHandler func(err error)

func (h FuncHandler) NewError(err error) {
	h(err)
}

type JoinHandler struct {
	e error
}

func (j *JoinHandler) NewError(err error) {
	j.e = Errors(j.e, err)
}
