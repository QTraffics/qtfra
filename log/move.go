package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/qtraffics/qtfra/enhancements/slicelib"
)

type Logger interface {
	Enabled(ctx context.Context, level Level) bool
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type ContextLogger interface {
	Logger
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

var _ ContextLogger = (*slog.Logger)(nil)

type Handler = slog.Handler

func New(handler Handler) ContextLogger {
	return slog.New(handler)
}

func NewSlog(handler Handler) *slog.Logger {
	return slog.New(handler)
}

func WithAttr(raw Logger, attr ...slog.Attr) Logger {
	logger := SlogLogger(raw)
	if logger == nil {
		return raw
	}
	return logger.With(slicelib.MapToAny(attr)...)
}

func WithGroup(raw Logger, name string) Logger {
	logger := SlogLogger(raw)
	if logger == nil {
		return raw
	}
	return logger.WithGroup(name)
}

type Level = slog.Level

const (
	LevelDebug   = slog.LevelDebug
	LevelInfo    = slog.LevelInfo
	LevelWarn    = slog.LevelWarn
	LevelError   = slog.LevelError
	LevelDisable = slog.LevelError + 1
)

func SlogLogger(l Logger) *slog.Logger {
	if l == nil {
		return nil
	}
	if sl, ok := l.(*slog.Logger); ok {
		return sl
	}
	return nil
}

func GetHandler(l Logger) Handler {
	type handler interface {
		Handler() Handler
	}

	if hh, ok := l.(handler); ok {
		return hh.Handler()
	}
	return nil
}

var (
	defaultLogger Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	NOP           Logger = slog.New(slog.DiscardHandler)
)

func SetDefaultLogger(l Logger) Logger {
	old := defaultLogger
	defaultLogger = l
	return old
}

func GetDefaultLogger() Logger {
	return defaultLogger
}
