package log

type featureLogger struct {
	h Handler
}

func (l *featureLogger) With(v ...any) Logger {
}
