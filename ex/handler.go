package ex

type Handler interface {
	NewError(err error)
}

type FuncHandler func(err error)

func (h FuncHandler) NewError(err error) {
	h(err)
}

type JoinError struct {
	Err error
}

func (j *JoinError) NewError(err error) {
	j.Err = Errors(j.Err, err)
}

func (j *JoinError) Error() string {
	return j.Err.Error()
}
