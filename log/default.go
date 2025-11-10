package log

import (
	"context"
	"log/slog"
	"os"
)

var (
	internalLogger ContextLogger = New(slog.NewTextHandler(os.Stderr, nil), WithAddCallerSkip(1))
	defaultLogger  ContextLogger = New(slog.NewTextHandler(os.Stderr, nil))
	NOP            ContextLogger = slog.New(slog.DiscardHandler)
)

func SetDefaultLogger(l Logger) Logger {
	old := defaultLogger
	defaultLogger = AsContextLogger(l)

	internalLogger = AsContextLogger(With(defaultLogger, WithAddCallerSkip(1)))
	return old
}

// GetDefaultLogger
// Deprecated, Use this is unsafe, may cause source info incorrect
func GetDefaultLogger() Logger {
	return internalLogger
}

func Default() ContextLogger {
	return defaultLogger
}

func Debug(msg string, v ...any) {
	Default().Debug(msg, v...)
}

func Info(msg string, v ...any) {
	Default().Info(msg, v...)
}

func Warn(msg string, v ...any) {
	Default().Warn(msg, v...)
}

func Error(msg string, v ...any) {
	Default().Error(msg, v...)
}

func DebugContext(ctx context.Context, msg string, v ...any) {
	Default().DebugContext(ctx, msg, v...)
}

func InfoContext(ctx context.Context, msg string, v ...any) {
	Default().InfoContext(ctx, msg, v...)
}

func WarnContext(ctx context.Context, msg string, v ...any) {
	Default().WarnContext(ctx, msg, v...)
}

func ErrorContext(ctx context.Context, msg string, v ...any) {
	Default().ErrorContext(ctx, msg, v...)
}
