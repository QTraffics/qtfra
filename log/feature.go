package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"
)

type fullFeaturedLogger interface {
	handler
	with
	group

	ContextLogger
}

var _ fullFeaturedLogger = (*featureLogger)(nil)

const initializeCallerSkip = 3

type featureLogger struct {
	h Handler

	strict      bool
	callerSkip  int
	enableTrace bool
	errorLogger Logger
}

func newFeatureLogger(h Handler, vv ...any) *featureLogger {
	l := &featureLogger{
		h:           h,
		enableTrace: true,
	}
	return l.With(vv...).(*featureLogger)
}

func (l *featureLogger) clone() *featureLogger {
	return &featureLogger{
		h:           l.h,
		strict:      l.strict,
		callerSkip:  l.callerSkip,
		enableTrace: l.enableTrace,
		errorLogger: l.errorLogger,
	}
}

func (l *featureLogger) log(ctx context.Context, level Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !l.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	if l.enableTrace {
		var pcs [1]uintptr
		// skip [runtime.Callers, this function, this function's caller, l.callerSkip ]
		runtime.Callers(initializeCallerSkip+l.callerSkip, pcs[:])
		pc = pcs[0]
	}

	record := slog.NewRecord(time.Now(), level, msg, pc)
	if !l.strict {
		record.Add(args...)
	} else {
		var attrs []slog.Attr
		for _, v := range args {
			if attr, ok := v.(slog.Attr); ok {
				attrs = append(attrs, attr)
				continue
			}

			if l.errorLogger != nil {
				l.errorLogger.Warn("Unexcepted logging attr",
					slog.String("Type", fmt.Sprintf("%T", v)),
					slog.String("Value", fmt.Sprintf("%v", v)))
			} else {
				panic(fmt.Sprintf("Unexcepted logging attr, Type: %T Value: %v", v, v))
			}

		}
		record.AddAttrs(attrs...)
	}

	err := l.h.Handle(ctx, record)
	if l.errorLogger != nil && err != nil {
		l.errorLogger.Error("An unexcepted error occurred during logging", AttrError(err))
	}
}

func (l *featureLogger) Handler() Handler {
	return l.h
}

func (l *featureLogger) With(a ...any) Logger {
	if len(a) == 0 {
		return l
	}

	l2 := l.clone()
	var (
		attrs   []slog.Attr
		options []Option
	)
	flushOptions := func() {
		for _, opt := range options {
			opt.apply(l2)
		}
		options = []Option{}
	}
	flushAttrs := func() {
		l2.h = l2.h.WithAttrs(attrs)
		attrs = []slog.Attr{}
	}
	defer flushOptions()
	defer flushAttrs()

	for _, aa := range a {
		switch x := aa.(type) {
		case slog.Attr:
			flushOptions()
			attrs = append(attrs, x)
		case Option:
			flushAttrs()
			options = append(options, x)
		default:
			panic(fmt.Sprintf("unsupported type: %T: %v", aa, aa))
		}
	}

	return l2
}

func (l *featureLogger) WithGroup(s string) Logger {
	newH := l.h.WithGroup(s)
	newL := l.clone()
	newL.h = newH

	return newL
}

func (l *featureLogger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	return l.h.Enabled(ctx, level)
}

func (l *featureLogger) Debug(msg string, args ...any) {
	l.log(context.Background(), LevelDebug, msg, args...)
}

func (l *featureLogger) Info(msg string, args ...any) {
	l.log(context.Background(), LevelInfo, msg, args...)
}

func (l *featureLogger) Warn(msg string, args ...any) {
	l.log(context.Background(), LevelWarn, msg, args...)
}

func (l *featureLogger) Error(msg string, args ...any) {
	l.log(context.Background(), LevelError, msg, args...)
}

func (l *featureLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelDebug, msg, args...)
}

func (l *featureLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelInfo, msg, args...)
}

func (l *featureLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelWarn, msg, args...)
}

func (l *featureLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelError, msg, args...)
}
