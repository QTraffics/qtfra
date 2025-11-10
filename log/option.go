package log

type Option interface {
	apply(l *featureLogger)
}

type funcOption func(l *featureLogger)

func (fo funcOption) apply(l *featureLogger) {
	fo(l)
}

func WithErrorLogger(l Logger) Option {
	return funcOption(func(l *featureLogger) {
		l.errorLogger = l
	})
}

func WithSetCallerSkip(n int) Option {
	return funcOption(func(l *featureLogger) {
		l.callerSkip = n
	})
}

func WithAddCallerSkip(n int) Option {
	return funcOption(func(l *featureLogger) {
		l.callerSkip += n
	})
}

func WithStrict() Option {
	return funcOption(func(l *featureLogger) {
		l.strict = true
	})
}

func WithDisableSource() Option {
	return funcOption(func(l *featureLogger) {
		l.enableTrace = false
	})
}

func WithEnableSource() Option {
	return funcOption(func(l *featureLogger) {
		l.enableTrace = true
	})
}
