package log

import (
	"log/slog"
	"os"
)

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
