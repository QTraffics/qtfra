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
	internalLogger.Debug(msg, v...)
}

func Info(msg string, v ...any) {
	internalLogger.Info(msg, v...)
}

func Warn(msg string, v ...any) {
	internalLogger.Warn(msg, v...)
}

func Error(msg string, v ...any) {
	internalLogger.Error(msg, v...)
}

func DebugContext(ctx context.Context, msg string, v ...any) {
	internalLogger.DebugContext(ctx, msg, v...)
}

func InfoContext(ctx context.Context, msg string, v ...any) {
	internalLogger.InfoContext(ctx, msg, v...)
}

func WarnContext(ctx context.Context, msg string, v ...any) {
	internalLogger.WarnContext(ctx, msg, v...)
}

func ErrorContext(ctx context.Context, msg string, v ...any) {
	internalLogger.ErrorContext(ctx, msg, v...)
}
